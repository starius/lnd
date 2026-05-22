//go:build dev && !js

package kvdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

// GetTestBackend opens (or creates if doesn't exist) a bbolt or etcd
// backed database (for testing), and returns a kvdb.Backend and a cleanup
// func. Whether to create/open bbolt or embedded etcd database is based
// on the TestBackend constant which is conditionally compiled with build tag.
// The passed path is used to hold all db files, while the name is only used
// for bbolt.
func GetTestBackend(path, name string) (Backend, func(), error) {
	empty := func() {}

	// Note that for tests, we expect only one db backend build flag
	// (or none) to be set at a time and thus one of the following switch
	// cases should ever be true
	switch {
	case PostgresBackend:
		key := filepath.Join(path, name)
		keyHash := sha256.Sum256([]byte(key))

		f, err := NewPostgresFixture("test_" + hex.EncodeToString(
			keyHash[:]),
		)
		if err != nil {
			return nil, func() {}, err
		}
		return f.DB(), func() {
			_ = f.DB().Close()
		}, nil

	case EtcdBackend:
		etcdConfig, cancel, err := StartEtcdTestBackend(path, 0, 0, "")
		if err != nil {
			return nil, empty, err
		}
		backend, err := Open(
			EtcdBackendName, context.Background(), etcdConfig,
		)
		return backend, cancel, err

	case SqliteBackend:
		dbPath := filepath.Join(path, name)
		keyHash := sha256.Sum256([]byte(dbPath))
		sqliteDb, err := StartSqliteTestBackend(
			path, name, "test_"+hex.EncodeToString(keyHash[:]),
		)
		if err != nil {
			return nil, empty, err
		}

		return sqliteDb, func() {
			_ = sqliteDb.Close()
		}, nil

	default:
		db, err := GetBoltBackend(&BoltBackendConfig{
			DBPath:         path,
			DBFileName:     name,
			NoFreelistSync: true,
			DBTimeout:      DefaultDBTimeout,
			ReadOnly:       false,
		})
		if err != nil {
			return nil, nil, err
		}
		return db, empty, nil
	}
}
