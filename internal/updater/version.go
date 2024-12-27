package updater

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
)

// ParseVersion parses a version string and returns a semver.Version instance.
// It supports versions prefixed with 'v', e.g., 'v1.2.3'.
func ParseVersion(versionStr string) (semver.Version, error) {
	// Remove 'v' prefix if present
	versionStr = strings.TrimPrefix(versionStr, "v")

	// Validate version format using regex
	matched, err := regexp.MatchString(`^\d+\.\d+\.\d+`, versionStr)
	if err != nil {
		return semver.Version{}, err
	}
	if !matched {
		return semver.Version{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	// Parse semantic version
	return semver.Parse(versionStr)
}
