//go:build dev

package channeldb

import (
	"testing"

	"github.com/lightningnetwork/lnd/invoices"
	"github.com/lightningnetwork/lnd/kvdb"
	"github.com/stretchr/testify/require"
)

// OpenForTesting opens or creates a channeldb to be used for tests. Any
// necessary schemas migrations due to updates will take place as necessary.
func OpenForTesting(t testing.TB, dbPath string,
	modifiers ...OptionModifier) *DB {

	backend, err := kvdb.GetBoltBackend(&kvdb.BoltBackendConfig{
		DBPath:            dbPath,
		DBFileName:        dbName,
		NoFreelistSync:    true,
		AutoCompact:       false,
		AutoCompactMinAge: kvdb.DefaultBoltAutoCompactMinAge,
		DBTimeout:         kvdb.DefaultDBTimeout,
	})
	require.NoError(t, err)

	db, err := CreateWithBackend(backend, modifiers...)
	require.NoError(t, err)

	db.dbPath = dbPath

	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	return db
}

// MakeTestInvoiceDB is used to create a test invoice database for testing
// purposes. It simply calls into MakeTestDB so the same modifiers can be used.
func MakeTestInvoiceDB(t *testing.T, modifiers ...OptionModifier) (
	invoices.InvoiceDB, error) {

	return MakeTestDB(t, modifiers...)
}

// MakeTestDB creates a new instance of the ChannelDB for testing purposes.
// A callback which cleans up the created temporary directories is also
// returned and intended to be executed after the test completes.
func MakeTestDB(t *testing.T, modifiers ...OptionModifier) (*DB, error) {
	// First, create a temporary directory to be used for the duration of
	// this test.
	tempDirName := t.TempDir()

	// Next, create channeldb for the first time.
	backend, backendCleanup, err := kvdb.GetTestBackend(tempDirName, "cdb")
	if err != nil {
		backendCleanup()
		return nil, err
	}

	cdb, err := CreateWithBackend(backend, modifiers...)
	if err != nil {
		backendCleanup()
		return nil, err
	}

	t.Cleanup(func() {
		cdb.Close()
		backendCleanup()
	})

	return cdb, nil
}
