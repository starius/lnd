package commands

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Output column ordering after removing pair separators.
const (
	outputColNum = iota
	outputColChanID
	outputColPubKey
	outputColCapacity
	outputColLocal
	outputColRemote
	outputColRatio
	outputColDay
	outputColMonth
	outputColFee
	outputColTotal
	outputColEffcy
)

// mockChannelReportClient records requests and returns preset responses.
type mockChannelReportClient struct {
	// listReqs captures ListChannels requests.
	listReqs []*lnrpc.ListChannelsRequest

	// fwdReqs captures ForwardingHistory requests.
	fwdReqs []*lnrpc.ForwardingHistoryRequest

	// listResp is the response returned for ListChannels.
	listResp *lnrpc.ListChannelsResponse

	// fwdResps are queued responses for ForwardingHistory.
	fwdResps []*lnrpc.ForwardingHistoryResponse
}

// ListChannels records the request and returns the configured response.
func (m *mockChannelReportClient) ListChannels(ctx context.Context,
	req *lnrpc.ListChannelsRequest,
	_ ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {

	m.listReqs = append(m.listReqs, req)

	return m.listResp, nil
}

// ForwardingHistory records the request and returns queued responses.
func (m *mockChannelReportClient) ForwardingHistory(ctx context.Context,
	req *lnrpc.ForwardingHistoryRequest,
	_ ...grpc.CallOption) (*lnrpc.ForwardingHistoryResponse, error) {

	m.fwdReqs = append(m.fwdReqs, req)
	if len(m.fwdResps) == 0 {
		return &lnrpc.ForwardingHistoryResponse{}, nil
	}

	resp := m.fwdResps[0]
	m.fwdResps = m.fwdResps[1:]

	return resp, nil
}

// TestBuildChannelReportTable verifies table layout and values.
func TestBuildChannelReportTable(t *testing.T) {
	now := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

	chanID1 := lnwire.ShortChannelID{
		BlockHeight: 114,
		TxIndex:     3,
		TxPosition:  0,
	}.ToUint64()
	chanID2 := lnwire.ShortChannelID{
		BlockHeight: 113,
		TxIndex:     3,
		TxPosition:  0,
	}.ToUint64()

	pubKey1 := makePubKeyHex([]byte{0x02, 0x71, 0xd6, 0xe2})
	pubKey2 := makePubKeyHex([]byte{0x03, 0xd7, 0xdc, 0x31})

	channels := []*lnrpc.Channel{
		{
			Active:                true,
			RemotePubkey:          pubKey2,
			ChanId:                chanID2,
			Capacity:              15000000,
			LocalBalance:          6991599,
			RemoteBalance:         8004930,
			TotalSatoshisReceived: 0,
			TotalSatoshisSent:     1004930,
			Private:               true,
		},
		{
			Active:                true,
			RemotePubkey:          pubKey1,
			ChanId:                chanID1,
			Capacity:              15000000,
			LocalBalance:          1165739,
			RemoteBalance:         13830791,
			TotalSatoshisReceived: 498176,
			TotalSatoshisSent:     7328967,
		},
	}

	events := []*lnrpc.ForwardingEvent{
		{
			TimestampNs: uint64(now.Add(-2 * time.Hour).UnixNano()),
			ChanIdIn:    chanID1,
			ChanIdOut:   chanID2,
			AmtInMsat:   111000,
			AmtOutMsat:  100000,
			FeeMsat:     12345,
		},
		{
			TimestampNs: uint64(
				now.Add(-10 * 24 * time.Hour).UnixNano(),
			),
			ChanIdIn:   chanID2,
			ChanIdOut:  chanID1,
			AmtInMsat:  500000,
			AmtOutMsat: 480000,
			FeeMsat:    2000,
		},
		{
			TimestampNs: uint64(
				now.Add(-40 * 24 * time.Hour).UnixNano(),
			),
			ChanIdIn:   chanID1,
			ChanIdOut:  chanID2,
			AmtInMsat:  900000,
			AmtOutMsat: 890000,
			FeeMsat:    1000,
		},
	}

	got := buildChannelReportTable(channels, events, now)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 7)

	headerLineIdx := findLineIndexContaining(
		lines, channelReportHeaderChanID,
	)
	require.NotEqual(t, -1, headerLineIdx)

	headerDividerIdx := findDividerLineAfter(lines, headerLineIdx+1)
	require.NotEqual(t, -1, headerDividerIdx)

	totalsDividerIdx := findDividerLineBefore(lines, len(lines)-1)
	require.Greater(t, totalsDividerIdx, headerDividerIdx)
	require.Less(t, totalsDividerIdx+1, len(lines))

	headerSection := strings.Join(lines[:headerDividerIdx], "\n")
	require.Contains(t, headerSection, "Forwarding cashflow")
	require.Contains(t, headerSection, "Earned Mon Fee")

	headerLine := lines[headerLineIdx]
	dividerLine := lines[headerDividerIdx]
	require.Equal(t, strings.Repeat("-", len(headerLine)), dividerLine)

	pipePositions := findPipePositions(headerLine)
	require.Equal(t, 11, len(pipePositions))

	dataLines := lines[headerDividerIdx+1 : totalsDividerIdx]
	require.Len(t, dataLines, 2)

	for _, line := range []string{
		dataLines[0], dataLines[1], lines[totalsDividerIdx+1],
	} {
		require.Equal(t, len(headerLine), len(line))
		for _, pos := range pipePositions {
			require.Equal(t, '|', rune(line[pos]))
		}
	}

	row1 := splitRowByPipes(dataLines[0], pipePositions)
	row2 := splitRowByPipes(dataLines[1], pipePositions)
	totalRow := splitRowByPipes(lines[totalsDividerIdx+1], pipePositions)

	row1Trim := trimCells(row1)
	row2Trim := trimCells(row2)
	totalTrim := trimCells(totalRow)

	require.Equal(t, rowCells(
		"1|%s|0271d6e2|15000000|1165739|13830791|8%%|111 0|111 480|"+
			"12.345 2.000|498176 7328967|52%%",
		strings.TrimSpace(formatShortChanID(chanID1)),
	), row1Trim)

	require.Equal(t, rowCells(
		"*2|%s|03d7dc31|15000000|6991599|8004930|47%%|0 100|500 100|"+
			"2.000 12.345|0 1004930|7%%",
		strings.TrimSpace(formatShortChanID(chanID2)),
	), row2Trim)

	require.Equal(t, rowCells(
		"2|||30000000|8157338|21835721|27%%|111 100|611 580|14 14|"+
			"498176 8333897|29%%",
	), totalTrim)

	for _, col := range []int{
		outputColDay,
		outputColMonth,
		outputColFee,
		outputColTotal,
	} {
		require.Equal(t, row1Trim[col],
			strings.TrimSpace(row1[col]))
		require.Equal(t, row2Trim[col],
			strings.TrimSpace(row2[col]))
		require.Equal(t, totalTrim[col],
			strings.TrimSpace(totalRow[col]))
	}
}

// TestBuildChannelReportTableExpandsWidths ensures wide values stay aligned.
func TestBuildChannelReportTableExpandsWidths(t *testing.T) {
	now := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

	chanID := lnwire.ShortChannelID{
		BlockHeight: 700000,
		TxIndex:     12,
		TxPosition:  0,
	}.ToUint64()

	pubKey := makePubKeyHex([]byte{0x02, 0xaa, 0xbb, 0xcc})
	capacity := int64(1234567890123)
	local := int64(123456789012)
	remote := int64(987654321098)
	totalIn := int64(123456789012)
	totalOut := int64(987654321098)

	channels := []*lnrpc.Channel{
		{
			Active:                true,
			RemotePubkey:          pubKey,
			ChanId:                chanID,
			Capacity:              capacity,
			LocalBalance:          local,
			RemoteBalance:         remote,
			TotalSatoshisReceived: totalIn,
			TotalSatoshisSent:     totalOut,
		},
	}

	dayInMsat := uint64(1234567890000)
	dayOutMsat := uint64(2345678900000)
	feeInMsat := uint64(987654321000)
	feeOutMsat := uint64(111222333444)

	events := []*lnrpc.ForwardingEvent{
		{
			TimestampNs: uint64(now.Add(-1 * time.Hour).UnixNano()),
			ChanIdIn:    chanID,
			AmtInMsat:   dayInMsat,
			FeeMsat:     feeInMsat,
		},
		{
			TimestampNs: uint64(now.Add(-2 * time.Hour).UnixNano()),
			ChanIdOut:   chanID,
			AmtOutMsat:  dayOutMsat,
			FeeMsat:     feeOutMsat,
		},
	}

	got := buildChannelReportTable(channels, events, now)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 6)

	headerLineIdx := findLineIndexContaining(
		lines, channelReportHeaderChanID,
	)
	require.NotEqual(t, -1, headerLineIdx)

	headerDividerIdx := findDividerLineAfter(lines, headerLineIdx+1)
	require.NotEqual(t, -1, headerDividerIdx)

	totalsDividerIdx := findDividerLineBefore(lines, len(lines)-1)
	require.Greater(t, totalsDividerIdx, headerDividerIdx)
	require.Less(t, totalsDividerIdx+1, len(lines))

	headerLine := lines[headerLineIdx]
	pipePositions := findPipePositions(headerLine)

	dataLines := lines[headerDividerIdx+1 : totalsDividerIdx]
	require.Len(t, dataLines, 1)

	rowLine := dataLines[0]
	totalLine := lines[totalsDividerIdx+1]

	for _, line := range []string{rowLine, totalLine} {
		require.Equal(t, len(headerLine), len(line))
		for _, pos := range pipePositions {
			require.Equal(t, '|', rune(line[pos]))
		}
	}

	require.Contains(t, rowLine, fmt.Sprintf("%d", capacity))
	require.Contains(t, rowLine, fmt.Sprintf("%d", local))
	require.Contains(t, rowLine, fmt.Sprintf("%d", remote))
	require.Contains(t, rowLine, fmt.Sprintf("%d", totalIn))
	require.Contains(t, rowLine, fmt.Sprintf("%d", totalOut))

	require.Contains(t, rowLine,
		fmt.Sprintf("%d", dayInMsat/1000))
	require.Contains(t, rowLine,
		fmt.Sprintf("%d", dayOutMsat/1000))
	require.Contains(t, rowLine, formatFeeMsat(int64(feeInMsat)))
	require.Contains(t, rowLine, formatFeeMsat(int64(feeOutMsat)))
}

// TestChannelReportWithClient verifies request construction and output.
func TestChannelReportWithClient(t *testing.T) {
	now := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

	chanID1 := lnwire.ShortChannelID{
		BlockHeight: 120,
		TxIndex:     1,
		TxPosition:  0,
	}.ToUint64()
	chanID2 := lnwire.ShortChannelID{
		BlockHeight: 121,
		TxIndex:     2,
		TxPosition:  0,
	}.ToUint64()

	pubKey1 := makePubKeyHex([]byte{0x02, 0xaa, 0xbb, 0xcc})
	pubKey2 := makePubKeyHex([]byte{0x03, 0xdd, 0xee, 0xff})
	channels := []*lnrpc.Channel{
		{
			Active:                true,
			RemotePubkey:          pubKey1,
			ChanId:                chanID1,
			Capacity:              1000,
			LocalBalance:          600,
			RemoteBalance:         400,
			TotalSatoshisReceived: 10,
			TotalSatoshisSent:     20,
		},
		{
			Active:                true,
			RemotePubkey:          pubKey2,
			ChanId:                chanID2,
			Capacity:              2000,
			LocalBalance:          800,
			RemoteBalance:         1200,
			TotalSatoshisReceived: 30,
			TotalSatoshisSent:     40,
		},
	}

	events := []*lnrpc.ForwardingEvent{
		{
			TimestampNs: uint64(now.Add(-2 * time.Hour).UnixNano()),
			ChanIdIn:    chanID1,
			ChanIdOut:   0,
			AmtInMsat:   1000,
			FeeMsat:     500,
		},
	}

	testCases := []struct {
		name             string
		peer             string
		activeOnly       bool
		inactiveOnly     bool
		publicOnly       bool
		privateOnly      bool
		expectedChannels []*lnrpc.Channel
		expectErr        bool
	}{
		{
			name:       "peer prefix active only",
			peer:       strings.ToUpper(pubKey1[:8]),
			activeOnly: true,
			expectedChannels: []*lnrpc.Channel{
				channels[0],
			},
		},
		{
			name:         "peer full inactive public",
			peer:         pubKey1,
			inactiveOnly: true,
			publicOnly:   true,
			expectedChannels: []*lnrpc.Channel{
				channels[0],
			},
		},
		{
			name: "no peer filter",
			expectedChannels: []*lnrpc.Channel{
				channels[0],
				channels[1],
			},
		},
		{
			name:      "invalid peer",
			peer:      "not a hex key",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockChannelReportClient{
				listResp: &lnrpc.ListChannelsResponse{
					Channels: channels,
				},
				fwdResps: []*lnrpc.ForwardingHistoryResponse{
					{
						ForwardingEvents: events,
						LastOffsetIndex:  1,
					},
				},
			}

			got, err := channelReportWithClient(
				context.Background(),
				mockClient,
				now,
				tc.peer,
				tc.activeOnly,
				tc.inactiveOnly,
				tc.publicOnly,
				tc.privateOnly,
			)

			if tc.expectErr {
				require.Error(t, err)
				require.Len(t, mockClient.listReqs, 0)
				require.Len(t, mockClient.fwdReqs, 0)
				return
			}

			require.NoError(t, err)
			require.Len(t, mockClient.listReqs, 1)
			require.Len(t, mockClient.fwdReqs, 1)

			require.Nil(t, mockClient.listReqs[0].Peer)

			require.Equal(t, tc.activeOnly,
				mockClient.listReqs[0].ActiveOnly)
			require.Equal(t, tc.inactiveOnly,
				mockClient.listReqs[0].InactiveOnly)
			require.Equal(t, tc.publicOnly,
				mockClient.listReqs[0].PublicOnly)
			require.Equal(t, tc.privateOnly,
				mockClient.listReqs[0].PrivateOnly)
			require.True(t, mockClient.listReqs[0].PeerAliasLookup)

			expectedStart := uint64(
				now.Add(-30 * 24 * time.Hour).Unix(),
			)
			expectedEnd := uint64(now.Unix())
			require.Equal(t, expectedStart,
				mockClient.fwdReqs[0].StartTime)
			require.Equal(t, expectedEnd,
				mockClient.fwdReqs[0].EndTime)
			require.Equal(t, uint32(0),
				mockClient.fwdReqs[0].IndexOffset)
			require.Equal(t, uint32(channelReportMaxEvents),
				mockClient.fwdReqs[0].NumMaxEvents)

			require.NotNil(t, tc.expectedChannels)
			want := buildChannelReportTable(
				tc.expectedChannels, events, now,
			)
			require.Equal(t, want, got)
		})
	}
}

// TestFormatShortChanID verifies the fixed-width channel ID formatting.
func TestFormatShortChanID(t *testing.T) {
	testCases := []struct {
		name   string
		chanID uint64
		expect string
	}{
		{
			name:   "zero",
			expect: "      0:0000:0",
		},
		{
			name: "pads tx index",
			chanID: lnwire.ShortChannelID{
				BlockHeight: 114,
				TxIndex:     3,
				TxPosition:  0,
			}.ToUint64(),
			expect: "    114:0003:0",
		},
		{
			name: "larger values",
			chanID: lnwire.ShortChannelID{
				BlockHeight: 654321,
				TxIndex:     120,
				TxPosition:  1,
			}.ToUint64(),
			expect: " 654321:0120:1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(
				t, tc.expect, formatShortChanID(tc.chanID),
			)
		})
	}
}

// makePubKeyHex returns a 33-byte pubkey hex string with the given prefix.
func makePubKeyHex(prefix []byte) string {
	pubKey := make([]byte, 33)
	copy(pubKey, prefix)

	return hex.EncodeToString(pubKey)
}

// findDividerLineAfter returns the index of the next divider after start.
func findDividerLineAfter(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if isDividerLine(lines[i]) {
			return i
		}
	}

	return -1
}

// findDividerLineBefore returns the index of the last divider at or before end.
func findDividerLineBefore(lines []string, end int) int {
	if end >= len(lines) {
		end = len(lines) - 1
	}
	for i := end; i >= 0; i-- {
		if isDividerLine(lines[i]) {
			return i
		}
	}

	return -1
}

// isDividerLine returns true if the line is made entirely of dashes.
func isDividerLine(line string) bool {
	if line == "" {
		return false
	}

	for _, ch := range line {
		if ch != '-' {
			return false
		}
	}

	return true
}

// splitRowByPipes splits a row into cells based on pipe positions.
func splitRowByPipes(line string, pipes []int) []string {
	cols := make([]string, len(pipes)+1)
	start := 0
	for i, pos := range pipes {
		cols[i] = line[start:pos]
		start = pos + 1
	}
	cols[len(pipes)] = line[start:]

	return cols
}

// rowCells formats a row template into trimmed cell values.
func rowCells(format string, args ...interface{}) []string {
	row := fmt.Sprintf(format, args...)
	cells := strings.Split(row, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}

	return cells
}

// trimCells trims whitespace for each cell in a row.
func trimCells(cells []string) []string {
	trimmed := make([]string, 0, len(cells))
	for _, cell := range cells {
		trimmed = append(
			trimmed,
			strings.Join(strings.Fields(cell), " "),
		)
	}

	return trimmed
}
