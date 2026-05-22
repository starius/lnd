//go:build dev

package fn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// UnwrapOrFail is used to extract a value from an option within a test
// context. If the option is None, then the test fails.
func (o Option[A]) UnwrapOrFail(t *testing.T) A {
	t.Helper()

	require.True(t, o.isSome, "Option[%T] was None()", o.some)

	return o.some
}

// UnwrapOrFail returns the success value or fails the test if it's an error.
func (r Result[T]) UnwrapOrFail(t *testing.T) T {
	t.Helper()

	require.True(
		t, r.IsOk(), "Result[%T] contained error: %v", r.left, r.right,
	)

	return r.left
}
