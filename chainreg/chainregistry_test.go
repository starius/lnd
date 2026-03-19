package chainreg

import (
	"testing"

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
