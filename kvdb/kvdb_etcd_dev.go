//go:build kvdb_etcd && dev
// +build kvdb_etcd,dev

package kvdb

import "github.com/lightningnetwork/lnd/kvdb/etcd"

// StartEtcdTestBackend creates an embedded etcd backend for testing, storing
// the database at the passed path.
func StartEtcdTestBackend(path string, clientPort, peerPort uint16,
	logFile string) (*etcd.Config, func(), error) {

	return etcd.NewEmbeddedEtcdInstance(
		path, clientPort, peerPort, logFile,
	)
}
