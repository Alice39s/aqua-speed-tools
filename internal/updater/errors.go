package updater

import "fmt"

// UpdateError represents an error with an operation context.
type UpdateError struct {
	Op  string
	Err error
}

func (e *UpdateError) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Err.Error()
}

// WrapError wraps an error with an operation context.
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &UpdateError{Op: op, Err: err}
}

// Predefined errors for common failure scenarios.
var (
	ErrNoExecutableFound = WrapError("archive scan", fmt.Errorf("no executable found in archive"))
	ErrDownloadFailed    = WrapError("download", fmt.Errorf("update download failed"))
	ErrChecksumMismatch  = WrapError("checksum", fmt.Errorf("file checksum mismatch"))
	ErrInvalidVersion    = WrapError("version", fmt.Errorf("invalid version file format"))
)
