//go:build kvdb_postgres && !dev

package kvdb

import (
	"errors"

	"github.com/lightningnetwork/lnd/kvdb/postgres"
)

func NewPostgresFixture(dbName string) (postgres.Fixture, error) {
	return nil, errors.New(
		"postgres test fixture not available in production builds",
	)
}

func StartEmbeddedPostgres() (func() error, error) {
	return nil, errors.New(
		"embedded postgres test backend not available in production builds",
	)
}
