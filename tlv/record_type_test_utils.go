//go:build dev

package tlv

import "testing"

// UnwrapOrFailV is used to extract a value from an option within a test
// context. If the option is None, then the test fails. This gives the
// underlying value of the record, rather then the record itself.
func (o *OptionalRecordT[T, V]) UnwrapOrFailV(t *testing.T) V {
	inner := o.Option.UnwrapOrFail(t)

	return inner.Val
}
