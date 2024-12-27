package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"aqua-speed-tools/internal/config"

	"github.com/schollz/progressbar/v3"
	"github.com/ulikunitz/xz"
	"go.uber.org/zap"
)

// Error types
type UpdateError struct {
	Op  string
	Err error
}

func (e *UpdateError) Error() string {
	if e.Op != "" {
		return e.Op + ": " + e.Err.Error()
	}
	return e.Err.Error()
}

// WrapError wraps an error with an operation context
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &UpdateError{Op: op, Err: err}
}

// Base errors
var (
	errNoExecutableFound = fmt.Errorf("no executable found in archive")
	errDownloadFailed    = fmt.Errorf("update download failed")
	errChecksumMismatch  = fmt.Errorf("file checksum mismatch")
	errInvalidVersion    = fmt.Errorf("invalid version file format")

	// Wrapped errors
	ErrNoExecutableFound = &UpdateError{Op: "archive scan", Err: errNoExecutableFound}
	ErrDownloadFailed    = &UpdateError{Op: "download", Err: errDownloadFailed}
	ErrChecksumMismatch  = &UpdateError{Op: "checksum", Err: errChecksumMismatch}
	ErrInvalidVersion    = &UpdateError{Op: "version", Err: errInvalidVersion}
)

const (
	downloadTimeout = 30 * time.Second
	executablePerm  = 0755
	versionPerm     = 0644
)

// GitHubRelease represents the GitHub release API response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Updater handles program update related operations
type Updater struct {
	Version        string
	InstallDir     string
	BinaryName     string
	CompressedName string
	logger         *zap.Logger
	client         *http.Client
}

// New creates a new Updater instance
func New(version string) *Updater {
	logger := initLogger()

	arch := normalizeArch(runtime.GOARCH)
	binaryName := formatBinaryName(arch)
	compressedName := formatCompressedName(arch, version)

	return &Updater{
		Version:        version,
		InstallDir:     getInstallDir(),
		BinaryName:     binaryName,
		CompressedName: compressedName,
		logger:         logger,
		client:         &http.Client{Timeout: downloadTimeout},
	}
}

// NewWithLocalVersion creates a new Updater instance with local version
func NewWithLocalVersion(defaultVersion string) (*Updater, error) {
	versionFile := filepath.Join(getInstallDir(), "version.txt")
	content, err := os.ReadFile(versionFile)
	if err != nil {
		// If read failed, use default version
		return New(defaultVersion), nil
	}

	parts := strings.Fields(string(content))
	if len(parts) > 0 {
		return New(parts[0]), nil
	}

	return New(defaultVersion), nil
}

func initLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return zap.NewExample()
	}
	return logger
}

func normalizeArch(arch string) string {
	if arch == "amd64" {
		return "x64"
	}
	return arch
}

func formatBinaryName(arch string) string {
	name := fmt.Sprintf("%s-%s-%s", config.ConfigReader.Binary.Prefix, runtime.GOOS, arch)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func formatCompressedName(arch, version string) string {
	version = strings.TrimPrefix(version, "v")
	if arch == "amd64" {
		arch = "x64"
	}

	name := fmt.Sprintf("%s-%s-%s_v%s", config.ConfigReader.Binary.Prefix, runtime.GOOS, arch, version)

	switch runtime.GOOS {
	case "windows", "darwin":
		return name + ".zip"
	default:
		return name + ".tar.xz"
	}
}

func getInstallDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "aqua-speed")
	}
	return "/etc/aqua-speed"
}

// GitHubClient defines the interface for fetching releases
type GitHubClient interface {
	GetLatestRelease(ctx context.Context, apiURL string) (*GitHubRelease, error)
}

// DefaultGitHubClient is the default implementation of GitHubClient
type DefaultGitHubClient struct {
	client *http.Client
	logger *zap.Logger
}

func NewDefaultGitHubClient(client *http.Client, logger *zap.Logger) *DefaultGitHubClient {
	return &DefaultGitHubClient{
		client: client,
		logger: logger,
	}
}

func (c *DefaultGitHubClient) GetLatestRelease(ctx context.Context, apiURL string) (*GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := fmt.Sprintf("Aqua-Speed-Updater/%s", strings.TrimSpace(config.ConfigReader.Script.Version))
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		resetTime := resp.Header.Get("X-RateLimit-Reset")
		return nil, fmt.Errorf("rate limit exceeded, reset at: %s", resetTime)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	bodyReader := io.LimitReader(resp.Body, 10<<20) // 10MB limit
	var release GitHubRelease
	if err := json.NewDecoder(bodyReader).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return &release, nil
}

// GetLatestVersion fetches the latest version and its download URL from GitHub
func (u *Updater) GetLatestVersion() (string, string, error) {
	if u == nil {
		return "", "", fmt.Errorf("updater instance is nil")
	}
	if u.client == nil {
		return "", "", fmt.Errorf("http client is nil")
	}

	// Validate configuration
	if strings.TrimSpace(config.ConfigReader.GithubApiBaseUrl) == "" ||
		strings.TrimSpace(config.ConfigReader.GithubRepo) == "" {
		return "", "", fmt.Errorf("invalid configuration: empty GitHub API URL or repo")
	}

	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest",
		strings.TrimRight(config.ConfigReader.GithubApiBaseUrl, "/"),
		strings.TrimSpace(config.ConfigReader.GithubRepo))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	githubClient := NewDefaultGitHubClient(u.client, u.logger)
	release, err := githubClient.GetLatestRelease(ctx, apiURL)
	if err != nil {
		return "", "", err
	}

	// Validate response data
	if strings.TrimSpace(release.TagName) == "" {
		return "", "", fmt.Errorf("invalid release: empty tag name")
	}

	version := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	if version == "" {
		return "", "", fmt.Errorf("invalid version format in tag: %s", release.TagName)
	}

	arch := normalizeArch(runtime.GOARCH)
	if arch == "" {
		return "", "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	assetName := formatCompressedName(arch, version)
	if assetName == "" {
		return "", "", fmt.Errorf("failed to format asset name for arch: %s, version: %s", arch, version)
	}

	u.logger.Debug("Looking for asset",
		zap.String("assetName", assetName),
		zap.String("version", version),
		zap.Int("totalAssets", len(release.Assets)))

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", "", fmt.Errorf("no matching asset found for %s (available assets: %d)", assetName, len(release.Assets))
	}

	// Validate download URL
	if _, err := url.Parse(downloadURL); err != nil {
		return "", "", fmt.Errorf("invalid download URL %q: %w", downloadURL, err)
	}

	return version, downloadURL, nil
}

// NeedsUpdate checks if an update is needed
func (u *Updater) NeedsUpdate() (bool, string) {
	latestVersion, downloadURL, err := u.GetLatestVersion()
	if err != nil {
		u.logger.Error("Failed to get latest version", zap.Error(err))
		return false, ""
	}

	// Compare versions
	if latestVersion == u.Version {
		return false, ""
	}

	return true, downloadURL
}

// CheckAndUpdate checks and updates the program
func (u *Updater) CheckAndUpdate() error {
	u.logger.Info("Starting update check",
		zap.String("current version", u.Version))

	// Create installation directory
	binDir := filepath.Join(u.InstallDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		u.logger.Error("Failed to create directory", zap.Error(err))
		return WrapError("create directory", err)
	}

	// Check if update is needed
	needsUpdate, downloadURL := u.NeedsUpdate()
	if !needsUpdate {
		u.logger.Info("Current version is already the latest")
		return nil
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "aqua-speed-update")
	if err != nil {
		u.logger.Error("Failed to create temporary directory", zap.Error(err))
		return WrapError("create temporary directory", err)
	}
	defer os.RemoveAll(tempDir)

	// Download and extract files
	if err := u.performUpdate(tempDir, downloadURL); err != nil {
		u.logger.Error("Update failed", zap.Error(err))
		return err
	}

	u.logger.Info("Update completed successfully")
	return nil
}

// performUpdate handles the update process: download, extract, verify, and install
func (u *Updater) performUpdate(tempDir, downloadURL string) error {
	binDir := filepath.Join(u.InstallDir, "bin")
	compressedPath := filepath.Join(tempDir, u.CompressedName)

	// Download file with progress
	downloadedData, err := u.downloadWithProgress(downloadURL)
	if err != nil {
		return WrapError("download file", err)
	}

	// Save the downloaded archive temporarily
	if err := os.WriteFile(compressedPath, downloadedData, 0644); err != nil {
		return WrapError("save downloaded archive", err)
	}

	// Read checksum and binary data from archive
	checksum, binaryData, err := u.readArchiveContents(compressedPath)
	if err != nil {
		return WrapError("read archive contents", err)
	}

	// Verify and save binary file
	destPath := filepath.Join(binDir, u.BinaryName)
	if err := u.verifyAndSaveBinary(destPath, binaryData, checksum); err != nil {
		return err
	}

	return nil
}

// downloadWithProgress downloads a file from the given URL and shows a progress bar
func (u *Updater) downloadWithProgress(downloadURL string) ([]byte, error) {
	resp, err := u.client.Get(downloadURL)
	if err != nil {
		return nil, WrapError("download", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, WrapError("download", fmt.Errorf("failed with status: %s", resp.Status))
	}

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading update",
	)

	buf := new(bytes.Buffer)
	_, err = io.Copy(io.MultiWriter(buf, bar), resp.Body)
	if err != nil {
		return nil, WrapError("download", err)
	}

	return buf.Bytes(), nil
}

// ArchiveReader defines the interface for archive readers
type ArchiveReader interface {
	Next() (string, io.Reader, error)
	Close() error
}

// NewArchiveReader creates a new ArchiveReader based on the archive type
func NewArchiveReader(path string) (ArchiveReader, error) {
	if strings.HasSuffix(path, ".zip") {
		return NewZipArchiveReader(path)
	}
	return NewTarXzArchiveReader(path)
}

// ZipArchiveReader implements ArchiveReader for ZIP files
type ZipArchiveReader struct {
	reader *zip.ReadCloser
	files  []*zip.File
	index  int
}

// NewZipArchiveReader creates a new ZipArchiveReader
func NewZipArchiveReader(path string) (*ZipArchiveReader, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	return &ZipArchiveReader{
		reader: reader,
		files:  reader.File,
		index:  0,
	}, nil
}

// Next returns the next file in the ZIP archive
func (z *ZipArchiveReader) Next() (string, io.Reader, error) {
	if z.index >= len(z.files) {
		return "", nil, io.EOF
	}
	file := z.files[z.index]
	z.index++
	rc, err := file.Open()
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file %s: %w", file.Name, err)
	}
	return file.Name, rc, nil
}

// Close closes the ZIP archive
func (z *ZipArchiveReader) Close() error {
	return z.reader.Close()
}

// TarXzArchiveReader implements ArchiveReader for TAR.XZ files
type TarXzArchiveReader struct {
	file      *os.File
	tarReader *tar.Reader
}

// NewTarXzArchiveReader creates a new TarXzArchiveReader
func NewTarXzArchiveReader(path string) (*TarXzArchiveReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open TAR.XZ file: %w", err)
	}

	xzReader, err := xz.NewReader(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create XZ reader: %w", err)
	}

	tarReader := tar.NewReader(xzReader)
	return &TarXzArchiveReader{
		file:      f,
		tarReader: tarReader,
	}, nil
}

// Next returns the next file in the TAR.XZ archive
func (t *TarXzArchiveReader) Next() (string, io.Reader, error) {
	header, err := t.tarReader.Next()
	if err != nil {
		return "", nil, err
	}
	return header.Name, t.tarReader, nil
}

// Close closes the TAR.XZ archive
func (t *TarXzArchiveReader) Close() error {
	return t.file.Close()
}

// readArchiveContents reads checksum and binary data from archive
func (u *Updater) readArchiveContents(archivePath string) (string, []byte, error) {
	archiveReader, err := NewArchiveReader(archivePath)
	if err != nil {
		return "", nil, WrapError("create archive reader", err)
	}
	defer archiveReader.Close()

	var checksum string
	var binaryData []byte
	var foundBinary, foundChecksum bool

	for {
		name, reader, err := archiveReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, WrapError("read archive", err)
		}

		u.logger.Debug("Scanning archive file", zap.String("filename", name))

		if strings.HasSuffix(name, "checksum.txt") {
			content, err := io.ReadAll(reader)
			if err != nil {
				return "", nil, WrapError("read checksum file", err)
			}
			checksum = readChecksumFromContent(string(content))
			foundChecksum = true
			u.logger.Debug("Found checksum file",
				zap.String("filename", name),
				zap.String("extracted checksum", checksum))
			continue
		}

		if u.isTargetBinary(name) {
			binaryData, err = io.ReadAll(reader)
			if err != nil {
				return "", nil, WrapError("read binary file", err)
			}
			foundBinary = true
			u.logger.Debug("Found binary file",
				zap.String("filename", name),
				zap.Int("size", len(binaryData)))
		}

		if foundBinary && foundChecksum {
			break
		}
	}

	if !foundBinary {
		return "", nil, ErrNoExecutableFound
	}
	if !foundChecksum {
		return "", nil, fmt.Errorf("checksum file not found")
	}

	// Verify checksum
	if err := u.verifyChecksum(binaryData, checksum); err != nil {
		return "", nil, err
	}

	return checksum, binaryData, nil
}

// readChecksumFromContent extracts checksum from checksum file content
func readChecksumFromContent(content string) string {
	// Checksum file format: "checksum filename"
	content = strings.TrimSpace(content)
	parts := strings.Fields(content)
	if len(parts) > 0 {
		return parts[0] // Only return checksum part
	}
	return content
}

// verifyChecksum verifies binary file checksum
func (u *Updater) verifyChecksum(data []byte, expectedChecksum string) error {
	actualChecksum, err := calculateChecksum(data)
	if err != nil {
		return WrapError("calculate checksum", err)
	}

	logChecksumInfo(u.logger, u.BinaryName, expectedChecksum, actualChecksum)

	if actualChecksum != expectedChecksum {
		return WrapError("checksum verification", fmt.Errorf("%w: expected=%s, actual=%s", errChecksumMismatch, expectedChecksum, actualChecksum))
	}

	return nil
}

// verifyAndSaveBinary verifies the checksum and saves the binary file
func (u *Updater) verifyAndSaveBinary(destPath string, binaryData []byte, checksum string) error {
	// Verify binary file checksum
	if err := u.verifyChecksum(binaryData, checksum); err != nil {
		return err
	}

	// Save binary file
	if err := os.WriteFile(destPath, binaryData, executablePerm); err != nil {
		u.logger.Error("Failed to save binary file", zap.Error(err))
		return WrapError("save binary file", err)
	}

	// Save version and checksum information
	if err := u.writeVersionInfo(checksum); err != nil {
		// If writing version information fails, delete the installed binary file
		os.Remove(destPath)
		return WrapError("save version information", err)
	}

	return nil
}

// writeVersionInfo saves version and checksum information
func (u *Updater) writeVersionInfo(checksum string) error {
	versionFile := filepath.Join(u.InstallDir, "version.txt")
	content := fmt.Sprintf("%s %s\n", u.Version, checksum)
	return os.WriteFile(versionFile, []byte(content), versionPerm)
}

// isTargetBinary checks if filename is the target binary file
func (u *Updater) isTargetBinary(filename string) bool {
	baseName := filepath.Base(filename)
	u.logger.Debug("Checking binary file",
		zap.String("filename", baseName),
		zap.String("target name", u.BinaryName))

	// Special handling for Windows platform
	if runtime.GOOS == "windows" {
		// Ensure file is .exe
		if !strings.HasSuffix(strings.ToLower(baseName), ".exe") {
			return false
		}
	}

	// Compare without extension
	fileNameWithoutExt := strings.TrimSuffix(strings.ToLower(baseName), filepath.Ext(baseName))
	targetNameWithoutExt := strings.TrimSuffix(strings.ToLower(u.BinaryName), filepath.Ext(u.BinaryName))

	// Check if it matches the target name (case-insensitive)
	// Supports the following formats:
	// - Exact match: speedtest-cdn-windows-amd64.exe
	// - With version number: speedtest-cdn-windows-amd64_v1.0.0.exe
	// - Without architecture: speedtest-cdn-windows.exe
	baseParts := strings.Split(targetNameWithoutExt, "-")
	if len(baseParts) > 0 {
		prefix := strings.Join(baseParts[:len(baseParts)-1], "-")
		if strings.HasPrefix(fileNameWithoutExt, prefix) {
			u.logger.Debug("Found matching binary file", zap.String("filename", baseName))
			return true
		}
	}

	return false
}

// calculateChecksum calculates the SHA1 checksum of data
func calculateChecksum(data []byte) (string, error) {
	hash := sha1.New()
	if _, err := hash.Write(data); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// logChecksumInfo logs the checksum information
func logChecksumInfo(logger *zap.Logger, filename, expected, actual string) {
	logger.Debug("Checksum information",
		zap.String("filename", filename),
		zap.String("expected checksum", expected),
		zap.String("actual checksum", actual))
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetBinaryPath gets the full path of the binary file
func (u *Updater) GetBinaryPath() string {
	return filepath.Join(u.InstallDir, "bin", u.BinaryName)
}

// ChecksumInfo represents a file checksum entry
type ChecksumInfo struct {
	Filename string
	Hash     string
}

// ChecksumList represents parsed checksums file
type ChecksumList struct {
	Entries map[string]string // filename -> hash
}

// ParseChecksumContent parses checksums.txt content
func ParseChecksumContent(content string) (*ChecksumList, error) {
	list := &ChecksumList{
		Entries: make(map[string]string),
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		hash := parts[0]
		filename := parts[1]

		// Remove file extension for comparison
		baseFilename := strings.TrimSuffix(filename, filepath.Ext(filename))
		list.Entries[baseFilename] = hash
	}

	return list, nil
}
