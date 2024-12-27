package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// Wrap error with operation context
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &UpdateError{Op: op, Err: err}
}

var (
	// Define base errors
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

	name := fmt.Sprintf("%s-%s-%s-v%s", config.ConfigReader.Binary.Prefix, runtime.GOOS, arch, version)

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

// getLatestVersion fetches the latest version from GitHub API
func (u *Updater) getLatestVersion() (string, string, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", config.ConfigReader.GithubApiBaseUrl, config.ConfigReader.GithubRepo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("Aqua-Speed-Updater/%s", u.Version))

	resp, err := u.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	version := strings.TrimPrefix(release.TagName, "v")

	arch := normalizeArch(runtime.GOARCH)
	assetName := formatCompressedName(arch, version)

	u.logger.Debug("Looking for asset",
		zap.String("assetName", assetName),
		zap.String("version", version))

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", "", fmt.Errorf("no matching asset found for %s", assetName)
	}

	return version, downloadURL, nil
}

// needsUpdate checks if an update is needed
func (u *Updater) needsUpdate() (bool, string) {
	latestVersion, downloadURL, err := u.getLatestVersion()
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
	needsUpdate, downloadURL := u.needsUpdate()
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

func (u *Updater) performUpdate(tempDir, downloadURL string) error {
	binDir := filepath.Join(u.InstallDir, "bin")
	compressedPath := filepath.Join(tempDir, u.CompressedName)

	// Download file
	if err := u.downloadFile(compressedPath, downloadURL); err != nil {
		return WrapError("download file", err)
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

func (u *Updater) verifyAndSaveBinary(destPath string, binaryData []byte, checksum string) error {
	// Verify binary file checksum
	if err := u.verifyChecksum(binaryData, checksum); err != nil {
		return WrapError("checksum verification", err)
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

// downloadFile downloads a file from the given URL
func (u *Updater) downloadFile(destPath, downloadURL string) error {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("Aqua-Speed-Updater/%s", u.Version))

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading update",
	)

	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// readArchiveContents reads checksum and binary data from archive
func (u *Updater) readArchiveContents(archivePath string) (checksum string, binaryData []byte, err error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return u.readZipContents(archivePath)
	}
	return u.readTarXzContents(archivePath)
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

// readZipContents reads contents from ZIP file
func (u *Updater) readZipContents(archivePath string) (string, []byte, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	var checksum string
	var binaryData []byte
	var foundBinary, foundChecksum bool
	var binaryName string

	// First, look for checksum file
	for _, file := range reader.File {
		u.logger.Debug("Scanning archive file", zap.String("filename", file.Name))

		if strings.HasSuffix(file.Name, "checksum.txt") {
			rc, err := file.Open()
			if err != nil {
				return "", nil, fmt.Errorf("failed to open checksum file: %w", err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", nil, fmt.Errorf("failed to read checksum file: %w", err)
			}
			checksum = readChecksumFromContent(string(content))
			foundChecksum = true
			u.logger.Debug("Found checksum file",
				zap.String("filename", file.Name),
				zap.String("raw content", string(content)),
				zap.String("extracted checksum", checksum))
			continue
		}

		if u.isTargetBinary(file.Name) {
			binaryName = file.Name
			rc, err := file.Open()
			if err != nil {
				return "", nil, fmt.Errorf("failed to open binary file: %w", err)
			}
			binaryData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", nil, fmt.Errorf("failed to read binary file: %w", err)
			}
			foundBinary = true
			u.logger.Debug("Found binary file",
				zap.String("filename", file.Name),
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

	// Verify read data
	actualChecksum, err := calculateChecksum(binaryData)
	if err != nil {
		return "", nil, err
	}
	logChecksumInfo(u.logger, binaryName, checksum, actualChecksum)

	return checksum, binaryData, nil
}

// readTarXzContents reads contents from TAR.XZ file
func (u *Updater) readTarXzContents(archivePath string) (string, []byte, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	xzReader, err := xz.NewReader(f)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create XZ decompressor: %w", err)
	}

	tarReader := tar.NewReader(xzReader)
	var checksum string
	var binaryData []byte
	var foundBinary, foundChecksum bool
	var binaryName string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("failed to read TAR file: %w", err)
		}

		u.logger.Debug("Scanning archive file", zap.String("filename", header.Name))

		if strings.HasSuffix(header.Name, "checksum.txt") {
			content, err := io.ReadAll(tarReader)
			if err != nil {
				return "", nil, fmt.Errorf("failed to read checksum file: %w", err)
			}
			checksum = readChecksumFromContent(string(content))
			foundChecksum = true
			u.logger.Debug("Found checksum file",
				zap.String("filename", header.Name),
				zap.String("raw content", string(content)),
				zap.String("extracted checksum", checksum))
			continue
		}

		if u.isTargetBinary(header.Name) {
			binaryName = header.Name
			binaryData, err = io.ReadAll(tarReader)
			if err != nil {
				return "", nil, fmt.Errorf("failed to read binary file: %w", err)
			}
			foundBinary = true
			u.logger.Debug("Found binary file",
				zap.String("filename", header.Name),
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

	// Verify read data
	actualChecksum, err := calculateChecksum(binaryData)
	if err != nil {
		return "", nil, err
	}
	logChecksumInfo(u.logger, binaryName, checksum, actualChecksum)

	return checksum, binaryData, nil
}

// verifyChecksum verifies binary file checksum
func (u *Updater) verifyChecksum(data []byte, expectedChecksum string) error {
	actualChecksum, err := calculateChecksum(data)
	if err != nil {
		return err
	}

	logChecksumInfo(u.logger, u.BinaryName, expectedChecksum, actualChecksum)

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("%w: expected checksum=%s, actual checksum=%s", ErrChecksumMismatch, expectedChecksum, actualChecksum)
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

// fileExists checks if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetBinaryPath gets the full path of the binary file
func (u *Updater) GetBinaryPath() string {
	return filepath.Join(u.InstallDir, "bin", u.BinaryName)
}

// Utility functions
func calculateChecksum(data []byte) (string, error) {
	hash := sha1.New()
	if _, err := io.Copy(hash, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func logChecksumInfo(logger *zap.Logger, filename, expected, actual string) {
	logger.Debug("Checksum information",
		zap.String("filename", filename),
		zap.String("expected checksum", expected),
		zap.String("actual checksum", actual))
}
