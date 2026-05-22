//go:build kvdb_sqlite && !(windows && (arm || 386)) && !(linux && (ppc64 || mips || mipsle || mips64))

package kvdb

const (
	// SqliteBackend is conditionally set to true when the kvdb_sqlite build
	// tag is defined. This will allow testing of other database backends.
	SqliteBackend = true
)
