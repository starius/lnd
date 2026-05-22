//go:build dev

package graphdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// MakeTestGraph creates a new instance of the ChannelGraph for testing
// purposes. The backing Store implementation depends on the version of
// NewTestDB included in the current build.
//
// NOTE: this is currently unused, but is left here for future use to show how
// NewTestDB can be used. As the SQL implementation of the Store is
// implemented, unit tests will be switched to use this function instead of
// the existing MakeTestGraph helper. Once only this function is used, the
// existing MakeTestGraph function will be removed and this one will be renamed.
func MakeTestGraph(t testing.TB,
	opts ...ChanGraphOption) *ChannelGraph {

	t.Helper()

	store := NewTestDB(t)

	// Default to synchronous cache population in tests so that the
	// cache is fully loaded before the test proceeds.
	allOpts := append(
		[]ChanGraphOption{WithSyncGraphCachePopulation()}, opts...,
	)

	graph, err := NewChannelGraph(store, allOpts...)
	require.NoError(t, err)
	require.NoError(t, graph.Start())

	t.Cleanup(func() {
		require.NoError(t, graph.Stop())
	})

	return graph
}
