package utils

var (
	// AppVersion holds the application version, set by main package
	AppVersion = "unknown"
)

// SetAppVersion sets the global application version
func SetAppVersion(version string) {
	AppVersion = version
}

// GetUserAgent returns a formatted user agent string for the given component
func GetUserAgent(component string) string {
	return component + "/" + AppVersion
}
