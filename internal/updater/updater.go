package updater

import (
	"aqua-speed-tools/internal/config"
	"aqua-speed-tools/internal/utils"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/schollz/progressbar/v3"
	"go.uber.org/zap"
)

// Updater handles program update related operations.
type Updater struct {
	Version        semver.Version
	InstallDir     string
	BinaryName     string
	CompressedName string
	logger         *zap.Logger
	client         *http.Client
	githubClient   GitHubClient
}

// New creates a new Updater instance.
func New(currentVersion string, urls *utils.GitHubURLs) (*Updater, error) {
	logger := InitLogger()

	parsedVersion, err := ParseVersion(currentVersion)
	if err != nil {
		return nil, WrapError("parse current version", err)
	}

	arch := NormalizeArch(runtime.GOARCH)
	binaryName := FormatBinaryName("aqua-speed", runtime.GOOS, arch)
	compressedName := FormatCompressedName("aqua-speed", runtime.GOOS, arch, currentVersion)

	// 如果没有提供 URLs，使用默认值
	if urls == nil {
		urls = utils.NewGitHubURLs(
			config.ConfigReader.GithubRawBaseURL,
			config.ConfigReader.GithubAPIBaseURL,
			config.ConfigReader.GithubRawJsdelivrSet,
		)
	}

	return &Updater{
		Version:        parsedVersion,
		InstallDir:     GetInstallDir(),
		BinaryName:     binaryName,
		CompressedName: compressedName,
		logger:         logger,
		client:         &http.Client{Timeout: time.Duration(config.ConfigReader.DownloadTimeout) * time.Second},
		githubClient:   NewDefaultGitHubClient(&http.Client{Timeout: time.Duration(config.ConfigReader.DownloadTimeout) * time.Second}, logger, currentVersion, urls),
	}, nil
}

// NewWithLocalVersionAndURLs creates a new Updater instance with the local version and custom GitHub URLs.
func NewWithLocalVersionAndURLs(defaultVersion string, urls *utils.GitHubURLs) (*Updater, error) {
	versionFile := filepath.Join(GetInstallDir(), "version.txt")
	content, err := ReadFileContent(versionFile)
	if err != nil {
		// If read failed, use default version
		return New(defaultVersion, urls)
	}

	parts := strings.Fields(content)
	if len(parts) > 0 {
		return New(parts[0], urls)
	}

	return New(defaultVersion, urls)
}

// NewWithLocalVersion creates a new Updater instance with the local version.
// If reading the local version fails, it falls back to the default version.
func NewWithLocalVersion(defaultVersion string) (*Updater, error) {
	return NewWithLocalVersionAndURLs(defaultVersion, nil)
}

// GetLatestVersion fetches the latest version and its download URL from GitHub.
func (u *Updater) GetLatestVersion() (semver.Version, string, string, error) {
	if u.githubClient == nil {
		return semver.Version{}, "", "", fmt.Errorf("github client is nil")
	}

	// 确保 GithubRepo 不为空并且格式正确
	repo := strings.Trim(config.DefaultGithubRepo, "/")
	if !strings.Contains(repo, "/") {
		return semver.Version{}, "", "", fmt.Errorf("invalid repository format: %s", repo)
	}

	owner, repoName := splitRepo(repo)
	baseURL := strings.TrimSuffix(config.ConfigReader.GithubAPIMagicURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimSuffix(config.ConfigReader.GithubAPIBaseURL, "/")
	}
	apiURL := fmt.Sprintf("%s/repos/%s/%s/releases/latest", baseURL, owner, repoName)

	u.logger.Debug("Fetching latest release",
		zap.String("apiURL", apiURL),
		zap.String("repo", repo),
		zap.String("currentVersion", u.Version.String()),
		zap.String("baseURL", baseURL),
		zap.String("magicURL", config.ConfigReader.GithubAPIMagicURL),
		zap.String("baseAPIURL", config.ConfigReader.GithubAPIBaseURL))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	release, err := u.githubClient.GetLatestRelease(ctx, apiURL)
	if err != nil {
		u.logger.Error("Failed to fetch latest release",
			zap.String("apiURL", apiURL),
			zap.Error(err))
		return semver.Version{}, "", "", fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// Parse and validate version
	latestVersion, err := ParseVersion(release.TagName)
	if err != nil {
		u.logger.Error("Failed to parse version",
			zap.String("tagName", release.TagName),
			zap.Error(err))
		return semver.Version{}, "", "", WrapError("parse latest version", err)
	}

	// Determine the appropriate asset name
	arch := NormalizeArch(runtime.GOARCH)
	expectedPrefix := fmt.Sprintf("aqua-speed-%s-%s", runtime.GOOS, arch)
	u.logger.Debug("Looking for asset",
		zap.String("expectedPrefix", expectedPrefix),
		zap.String("version", latestVersion.String()),
		zap.Int("totalAssets", len(release.Assets)),
		zap.Any("assets", release.Assets))

	var downloadURL string
	var matchedAssetName string
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" {
			continue
		}
		if strings.HasPrefix(asset.Name, expectedPrefix) {
			downloadURL = asset.BrowserDownloadURL
			matchedAssetName = asset.Name
			break
		}
	}

	if downloadURL == "" {
		u.logger.Error("No matching asset found",
			zap.String("expectedPrefix", expectedPrefix),
			zap.Int("totalAssets", len(release.Assets)),
			zap.Any("availableAssets", release.Assets))
		return semver.Version{}, "", "", fmt.Errorf("no matching asset found for %s (available assets: %d)", expectedPrefix, len(release.Assets))
	}

	u.logger.Debug("Found matching asset",
		zap.String("assetName", matchedAssetName),
		zap.String("downloadURL", downloadURL),
		zap.String("version", latestVersion.String()))

	// Try to convert GitHub release URL to mirror if available
	originalDownloadURL := downloadURL
	if u.githubClient != nil {
		if defaultClient, ok := u.githubClient.(*DefaultGitHubClient); ok && defaultClient.urls != nil && defaultClient.urls.FastestMirror != "" {
			if mirrorURL, err := utils.ConvertReleaseURLToMirror(downloadURL, defaultClient.urls.FastestMirror); err == nil && mirrorURL != downloadURL {
				downloadURL = mirrorURL
				u.logger.Info("Using mirror for download",
					zap.String("original", originalDownloadURL),
					zap.String("mirror", downloadURL),
					zap.String("mirrorBase", defaultClient.urls.FastestMirror))
			} else {
				u.logger.Debug("Could not convert to mirror URL",
					zap.String("original", originalDownloadURL),
					zap.String("mirrorBase", defaultClient.urls.FastestMirror),
					zap.Error(err))
			}
		}
	}

	// Validate download URL
	if _, err := url.Parse(downloadURL); err != nil {
		u.logger.Error("Invalid download URL",
			zap.String("downloadURL", downloadURL),
			zap.Error(err))
		return semver.Version{}, "", "", fmt.Errorf("invalid download URL %q: %w", downloadURL, err)
	}

	return latestVersion, downloadURL, matchedAssetName, nil
}

// NeedsUpdate determines if an update is needed by comparing the current version with the latest version.
func (u *Updater) NeedsUpdate() (bool, semver.Version, string, string) {
	latestVersion, downloadURL, assetName, err := u.GetLatestVersion()
	if err != nil {
		u.logger.Error("Failed to get latest version", zap.Error(err))
		return false, semver.Version{}, "", ""
	}

	// Compare versions using semantic versioning
	if latestVersion.LTE(u.Version) {
		return false, semver.Version{}, "", ""
	}

	return true, latestVersion, downloadURL, assetName
}

// CheckAndUpdate checks for updates and performs the update if needed.
func (u *Updater) CheckAndUpdate() error {
	u.logger.Info("Starting update check", zap.String("current version", u.Version.String()))

	// Create installation directory
	binDir := filepath.Join(u.InstallDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		u.logger.Error("Failed to create installation directory", zap.Error(err))
		return WrapError("create installation directory", err)
	}

	// Check if update is needed
	needsUpdate, latestVersion, downloadURL, assetName := u.NeedsUpdate()
	if !needsUpdate {
		u.logger.Info("Current version is already the latest")
		return nil
	}

	u.logger.Info("Update available", zap.String("latest version", latestVersion.String()))

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "aqua-speed-update")
	if err != nil {
		u.logger.Error("Failed to create temporary directory", zap.Error(err))
		return WrapError("create temporary directory", err)
	}
	defer os.RemoveAll(tempDir)

	// Perform the update
	if err := u.performUpdate(tempDir, downloadURL, latestVersion, assetName); err != nil {
		u.logger.Error("Update failed", zap.Error(err))
		return err
	}

	u.logger.Info("Update completed successfully", zap.String("new version", latestVersion.String()))
	return nil
}

// performUpdate handles the download, extraction, verification, and installation of the update.
func (u *Updater) performUpdate(tempDir, downloadURL string, latestVersion semver.Version, assetName string) error {
	binDir := filepath.Join(u.InstallDir, "bin")
	compressedPath := filepath.Join(tempDir, assetName)

	// Download the archive
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

	// Verify and save the binary file
	destPath := filepath.Join(binDir, u.BinaryName)
	if err := u.verifyAndSaveBinary(destPath, binaryData, latestVersion, checksum); err != nil {
		return err
	}

	return nil
}

// downloadWithProgress downloads a file from the given URL and displays a progress bar.
func (u *Updater) downloadWithProgress(downloadURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, WrapError("create download request", err)
	}

	// Set proper User-Agent header
	userAgent := "Aqua-Speed-Updater/" + u.Version.String()
	req.Header.Set("User-Agent", userAgent)

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, WrapError("download", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, WrapError("download", fmt.Errorf("failed with status: %s", resp.Status))
	}

	u.logger.Info("Downloading from", zap.String("url", downloadURL))
	fmt.Printf("Downloading from '%s' ...\n", downloadURL)

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading update",
	)

	buf := new(bytes.Buffer)
	_, err = io.Copy(io.MultiWriter(buf, bar), resp.Body)
	if err != nil {
		return nil, WrapError("download", err)
	}

	// Ensure progress bar completes and add a newline
	bar.Finish()
	fmt.Println() // Add newline for clean output

	return buf.Bytes(), nil
}

// verifyAndSaveBinary verifies the checksum and saves the binary file.
func (u *Updater) verifyAndSaveBinary(destPath string, binaryData []byte, latestVersion semver.Version, checksum string) error {
	// Verify binary file checksum
	actualChecksum, err := CalculateChecksum(binaryData)
	if err != nil {
		return WrapError("calculate checksum", err)
	}

	u.logger.Debug("Checksum information",
		zap.String("filename", u.BinaryName),
		zap.String("expected checksum", checksum),
		zap.String("actual checksum", actualChecksum))

	if actualChecksum != checksum {
		return WrapError("checksum verification", fmt.Errorf("%w: expected=%s, actual=%s", ErrChecksumMismatch, checksum, actualChecksum))
	}

	// Save binary file
	if err := os.WriteFile(destPath, binaryData, 0755); err != nil {
		u.logger.Error("Failed to save binary file", zap.Error(err))
		return WrapError("save binary file", err)
	}

	// Save version and checksum information
	if err := u.writeVersionInfo(latestVersion.String(), checksum); err != nil {
		// If writing version information fails, delete the installed binary file
		os.Remove(destPath)
		return WrapError("save version information", err)
	}

	return nil
}

// writeVersionInfo saves version and checksum information.
func (u *Updater) writeVersionInfo(latestVersion, checksum string) error {
	versionFile := filepath.Join(u.InstallDir, "version.txt")
	content := fmt.Sprintf("%s %s\n", latestVersion, checksum)
	return os.WriteFile(versionFile, []byte(content), 0644)
}

// readArchiveContents reads checksum and binary data from the archive.
func (u *Updater) readArchiveContents(archivePath string) (string, []byte, error) {
	archiveReader, err := NewArchiveReader(archivePath, u.logger)
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

		switch {
		case strings.HasSuffix(name, "checksum.txt"):
			content, err := io.ReadAll(reader)
			if err != nil {
				return "", nil, WrapError("read checksum file", err)
			}
			checksum = readChecksumFromContent(string(content))
			foundChecksum = true
			u.logger.Debug("Found checksum file", zap.String("checksum", checksum))
		case u.isTargetBinary(name):
			binaryData, err = io.ReadAll(reader)
			if err != nil {
				return "", nil, WrapError("read binary file", err)
			}
			foundBinary = true
			u.logger.Debug("Found binary file", zap.Int("size", len(binaryData)))
		}

		if foundBinary && foundChecksum {
			break
		}
	}

	if !foundBinary {
		return "", nil, ErrNoExecutableFound
	}
	if !foundChecksum {
		return "", nil, WrapError("read archive contents", fmt.Errorf("checksum file not found"))
	}

	// Verify checksum
	if err := u.verifyChecksum(binaryData, checksum); err != nil {
		return "", nil, err
	}

	return checksum, binaryData, nil
}

// verifyChecksum verifies the binary data against the expected checksum.
func (u *Updater) verifyChecksum(data []byte, expectedChecksum string) error {
	actualChecksum, err := CalculateChecksum(data)
	if err != nil {
		return WrapError("calculate checksum", err)
	}

	u.logger.Debug("Checksum verification",
		zap.String("expected", expectedChecksum),
		zap.String("actual", actualChecksum))

	if actualChecksum != expectedChecksum {
		return WrapError("checksum verification", fmt.Errorf("%w: expected=%s, actual=%s", ErrChecksumMismatch, expectedChecksum, actualChecksum))
	}

	return nil
}

// isTargetBinary checks if the filename corresponds to the target binary.
func (u *Updater) isTargetBinary(filename string) bool {
	baseName := filepath.Base(filename)
	u.logger.Debug("Checking binary file", zap.String("filename", baseName), zap.String("target name", u.BinaryName))

	// Ensure correct extension for Windows
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(baseName), ".exe") {
		return false
	}

	// Compare without extension and case-insensitive
	fileNameWithoutExt := strings.TrimSuffix(strings.ToLower(baseName), filepath.Ext(baseName))
	targetNameWithoutExt := strings.TrimSuffix(strings.ToLower(u.BinaryName), filepath.Ext(u.BinaryName))

	// Check for exact or prefixed match
	return strings.HasPrefix(fileNameWithoutExt, targetNameWithoutExt)
}

// readChecksumFromContent extracts the checksum from the checksum file content.
func readChecksumFromContent(content string) string {
	// Assume format: "checksum filename"
	content = strings.TrimSpace(content)
	parts := strings.Fields(content)
	if len(parts) > 0 {
		return parts[0]
	}
	return content
}

// splitRepo splits a repository string into owner and repo parts
func splitRepo(fullRepo string) (owner, repo string) {
	parts := strings.Split(fullRepo, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
