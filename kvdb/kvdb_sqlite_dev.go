//go:build kvdb_sqlite && dev && !(windows && (arm || 386)) && !(linux && (ppc64 || mips || mipsle || mips64))

package kvdb

import (
	"context"
	"os"
	"time"

	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/lightningnetwork/lnd/kvdb/sqlbase"
	"github.com/lightningnetwork/lnd/kvdb/sqlite"
)

const testMaxConnections = 50

// StartSqliteTestBackend starts a sqlite backed wallet.DB instance.
func StartSqliteTestBackend(path, name, table string) (walletdb.DB, error) {
	if !fileExists(path) {
		err := os.Mkdir(path, 0700)
		if err != nil {
			return nil, err
		}
	}

	sqlbase.Init(testMaxConnections)
	return sqlite.NewSqliteBackend(
		context.Background(), &sqlite.Config{
			Timeout:     time.Second * 30,
			BusyTimeout: time.Second * 5,
		}, path, name, table,
	)
}
