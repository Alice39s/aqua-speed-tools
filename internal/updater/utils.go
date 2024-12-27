package updater

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizeArch converts GOARCH to a normalized architecture string.
func NormalizeArch(arch string) string {
	if arch == "amd64" {
		return "x64"
	}
	return arch
}

// FormatBinaryName constructs the binary name based on OS and architecture.
func FormatBinaryName(prefix, osName, arch string) string {
	name := fmt.Sprintf("%s-%s-%s", prefix, osName, arch)
	if osName == "windows" {
		name += ".exe"
	}
	return name
}

// FormatCompressedName constructs the compressed archive name based on OS, architecture, and version.
func FormatCompressedName(prefix, osName, arch, version string) string {
	version = strings.TrimPrefix(version, "v")
	if arch == "amd64" {
		arch = "x64"
	}

	name := fmt.Sprintf("%s-%s-%s_v%s", prefix, osName, arch, version)

	switch osName {
	case "windows", "darwin":
		return name + ".zip"
	default:
		return name + ".tar.xz"
	}
}

// GetInstallDir determines the installation directory based on the OS.
func GetInstallDir(appName string) string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), appName)
	case "linux":
		return "/etc/" + appName
	case "freebsd":
		return "/usr/local/" + appName
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library/Application Support", appName)
	default:
		return "/etc/" + appName
	}
}

// CalculateChecksum computes the SHA1 checksum of the given data.
func CalculateChecksum(data []byte) (string, error) {
	hash := sha1.New()
	if _, err := hash.Write(data); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// FileExists checks if a file exists at the specified path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFileContent reads and returns the content of a file.
func ReadFileContent(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
