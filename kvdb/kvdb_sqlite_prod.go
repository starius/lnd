//go:build kvdb_sqlite && !dev && !(windows && (arm || 386)) && !(linux && (ppc64 || mips || mipsle || mips64))

package kvdb

import (
	"errors"

	"github.com/btcsuite/btcwallet/walletdb"
)

// StartSqliteTestBackend is a production stub for the test-only sqlite
// backend starter.
func StartSqliteTestBackend(path, name, table string) (walletdb.DB, error) {
	return nil, errors.New(
		"sqlite test backend not available in production builds",
	)
}
