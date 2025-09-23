//go:build !linux && !darwin

package fn

import (
	"fmt"
	"runtime"
)

// This stub implementation is used on platforms where the real
// immutable-flag manipulation (FS_IOC_* on Linux or chflags on macOS)
// is not available or not implemented. It lives next to the OS-specific
// implementations so imports are the same across builds.

// makeImmutable uses chmod because immutable is not available.
func makeImmutable(path string) error {
	return os.Chmod(path, 0444)
}

// clearImmutable does nothing.
func clearImmutable(path string) error {
	return nil
}
