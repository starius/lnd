//go:build kvdb_etcd && !dev
// +build kvdb_etcd,!dev

package kvdb

import (
	"errors"

	"github.com/lightningnetwork/lnd/kvdb/etcd"
)

// StartEtcdTestBackend is a production stub for the test-only embedded etcd
// backend.
func StartEtcdTestBackend(path string, clientPort, peerPort uint16,
	logFile string) (*etcd.Config, func(), error) {

	return nil, func() {}, errors.New(
		"embedded etcd test backend not available in production builds",
	)
}
