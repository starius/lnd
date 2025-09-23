//go:build darwin

package fn

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

// makeImmutable sets the filesystem "immutable" flag on a path while
// preserving all other flags. Subsequent writes (truncate, unlink, rename into,
// etc.) should fail with EPERM until cleared.
func makeImmutable(path string) error {
	// UF_IMMUTABLE is a user flag bit in st_flags (see <sys/stat.h>)
	const UF_IMMUTABLE = 0x00000002

	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return err
	}

	newFlags := st.Flags | UF_IMMUTABLE
	if newFlags == st.Flags {
		return nil // already immutable
	}
	// chflags sets absolute flags, so we pass the combined mask
	if err := unix.Chflags(path, int(newFlags)); err != nil {
		return err
	}
	return nil
}

// clearImmutable clears the filesystem "immutable" flag on a path while
// preserving all other flags.
func clearImmutable(path string) error {
	const UF_IMMUTABLE = 0x00000002

	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return err
	}

	newFlags := st.Flags &^ UF_IMMUTABLE
	if newFlags == st.Flags {
		return nil // already cleared
	}
	if err := unix.Chflags(path, int(newFlags)); err != nil {
		return err
	}
	return nil
}
