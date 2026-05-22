//go:build kvdb_postgres && dev

package kvdb

import "github.com/lightningnetwork/lnd/kvdb/postgres"

func NewPostgresFixture(dbName string) (postgres.Fixture, error) {
	return postgres.NewFixture(dbName)
}

func StartEmbeddedPostgres() (func() error, error) {
	return postgres.StartEmbeddedPostgres()
}
