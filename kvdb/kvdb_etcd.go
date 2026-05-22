//go:build kvdb_etcd
// +build kvdb_etcd

package kvdb

// EtcdBackend is conditionally set to etcd when the kvdb_etcd build tag is
// defined, allowing testing our database code with etcd backend.
const EtcdBackend = true
