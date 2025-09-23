//go:build linux

package fn

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// makeImmutable sets the filesystem "immutable" flag on a path while
// preserving all other flags. Subsequent writes (truncate, unlink, rename into,
// etc.) should fail with EPERM until cleared.
func makeImmutable(path string) error {
	const (
		// From <linux/fs.h>
		FS_IOC_GETFLAGS = 0x80086601
		FS_IOC_SETFLAGS = 0x40086602
		FS_IMMUTABLE_FL = 0x00000010
	)

	f, err := os.Open(path) // O_RDONLY is fine for ioctls
	if err != nil {
		return err
	}
	defer f.Close()

	// Read current flags
	flags, err := unix.IoctlGetInt(int(f.Fd()), FS_IOC_GETFLAGS)
	if err != nil {
		return wrapMaybeENOTTY(err, "FS_IOC_GETFLAGS")
	}

	// Set immutable bit, preserve others
	newFlags := flags | FS_IMMUTABLE_FL
	if newFlags == flags {
		return nil // already immutable
	}

	if err := unix.IoctlSetInt(int(f.Fd()), FS_IOC_SETFLAGS, newFlags); err != nil {
		return wrapMaybeENOTTY(err, "FS_IOC_SETFLAGS")
	}
	return nil
}

// clearImmutable clears the filesystem "immutable" flag on a path while
// preserving all other flags.
func clearImmutable(path string) error {
	const (
		FS_IOC_GETFLAGS = 0x80086601
		FS_IOC_SETFLAGS = 0x40086602
		FS_IMMUTABLE_FL = 0x00000010
	)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	flags, err := unix.IoctlGetInt(int(f.Fd()), FS_IOC_GETFLAGS)
	if err != nil {
		return wrapMaybeENOTTY(err, "FS_IOC_GETFLAGS")
	}

	newFlags := flags &^ FS_IMMUTABLE_FL
	if newFlags == flags {
		return nil // already cleared
	}

	if err := unix.IoctlSetInt(int(f.Fd()), FS_IOC_SETFLAGS, newFlags); err != nil {
		return wrapMaybeENOTTY(err, "FS_IOC_SETFLAGS")
	}
	return nil
}

func wrapMaybeENOTTY(err error, op string) error {
	// Many filesystems that don't support FS ioctls return ENOTTY (Inappropriate ioctl for device).
	// Surface a helpful hint in that case.
	if errno, ok := err.(unix.Errno); ok && errno == unix.ENOTTY {
		return fmt.Errorf("%s unsupported on this filesystem (ENOTTY). Try a different FS or a different failure method: %w", op, err)
	}
	return err
}
