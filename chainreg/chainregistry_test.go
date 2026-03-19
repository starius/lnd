package chainreg

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/btcsuite/btcd/rpcclient"

	"github.com/stretchr/testify/require"
)

// TestParseConnectionsOut tests parsing the outbound connection count from a
// getnetworkinfo response.
func TestParseConnectionsOut(t *testing.T) {
	tests := []struct {
		name        string
		resp        string
		expected    int
		expectedErr string
	}{
		{
			name:     "valid response",
			resp:     `{"connections_out": 8}`,
			expected: 8,
		},
		{
			name:        "missing field",
			resp:        `{}`,
			expectedErr: "connections_out is missing",
		},
		{
			name:        "wrong field type",
			resp:        `{"connections_out": "8"}`,
			expectedErr: "cannot unmarshal string",
		},
		{
			name:        "invalid json",
			resp:        `{"connections_out":`,
			expectedErr: "unexpected end of JSON input",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connectionsOut, err := parseConnectionsOut(
				[]byte(test.resp),
			)

			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, connectionsOut)
			}
		})
	}
}

// TestBitcoindPeerCheckTimeout tests that an outbound peer check returns once
// the context timeout elapses.
func TestBitcoindPeerCheckTimeout(t *testing.T) {
	t.Parallel()

	resp := `{"result":{"connections_out":8},"error":null,"id":"1"}`
	handle := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)

		_, err := w.Write([]byte(resp))
		require.NoError(t, err)
	}

	server := httptest.NewServer(http.HandlerFunc(handle))
	t.Cleanup(server.Close)

	host := strings.TrimPrefix(server.URL, "http://")
	cfg := &rpcclient.ConnConfig{
		Host:       host,
		DisableTLS: true,
	}
	check, err := newBitcoindPeerCheck(cfg, 50*time.Millisecond)
	require.NoError(t, err)

	err = check.checkOutboundPeers()
	require.ErrorContains(t, err, "context deadline exceeded")
}

// TestBitcoindPeerCheckSingleFlight tests that concurrent checks don't issue
// concurrent RPC calls.
func TestBitcoindPeerCheckSingleFlight(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	allowResponse := make(chan struct{})

	resp := `{"result":{"connections_out":8},"error":null,"id":"1"}`
	handle := func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		<-allowResponse

		_, err := w.Write([]byte(resp))
		require.NoError(t, err)
	}

	server := httptest.NewServer(http.HandlerFunc(handle))
	t.Cleanup(server.Close)

	host := strings.TrimPrefix(server.URL, "http://")
	cfg := &rpcclient.ConnConfig{
		Host:       host,
		DisableTLS: true,
	}
	check, err := newBitcoindPeerCheck(cfg, 5*time.Second)
	require.NoError(t, err)

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- check.checkOutboundPeers()
	}()

	require.Eventually(t, func() bool {
		return requests.Load() == 1
	}, time.Second, 10*time.Millisecond)

	start := time.Now()
	err = check.checkOutboundPeers()
	require.NoError(t, err)
	require.Less(t, time.Since(start), 100*time.Millisecond)
	require.Equal(t, int32(1), requests.Load())

	close(allowResponse)

	select {
	case err := <-firstDone:
		require.NoError(t, err)

	case <-time.After(time.Second):
		t.Fatal("first check did not complete")
	}
}

// TestBitcoindPeerCheckRPCError tests handling of non-OK HTTP responses.
func TestBitcoindPeerCheckRPCError(t *testing.T) {
	t.Parallel()

	handle := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}
	server := httptest.NewServer(http.HandlerFunc(handle))
	t.Cleanup(server.Close)

	host := strings.TrimPrefix(server.URL, "http://")
	cfg := &rpcclient.ConnConfig{
		Host:       host,
		DisableTLS: true,
	}
	check, err := newBitcoindPeerCheck(cfg, time.Second)
	require.NoError(t, err)

	err = check.checkOutboundPeers()
	require.ErrorContains(t, err, fmt.Sprintf("http status %d",
		http.StatusForbidden))
}
