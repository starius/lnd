package commands

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

// Static strings for the table.
const (
	channelReportForwardingLabel = "Forwarding cashflow"
	channelReportFeeLabel        = "Earned Mon Fee"
	channelReportMaxEvents       = 50000

	channelReportHeaderNum      = "Num"
	channelReportHeaderChanID   = "Channel ID"
	channelReportHeaderPubKey   = "Public Key"
	channelReportHeaderCapacity = "Capacity"
	channelReportHeaderLocal    = "Local"
	channelReportHeaderRemote   = "Remote"
	channelReportHeaderRatio    = "Ratio"
	channelReportHeaderDayIn    = "Day In"
	channelReportHeaderMonthIn  = "Month In"
	channelReportHeaderTotalIn  = "Total In"
	channelReportHeaderOutAmt   = "Out Amount"
	channelReportHeaderFeeIn    = "In"
	channelReportHeaderFeeOut   = "Out"
	channelReportHeaderEffcy    = "Effcy"
)

// channelReportPeerHexLen is the hex length of a compressed pubkey.
const channelReportPeerHexLen = 66

// The column ordering for header helpers.
const (
	channelReportColNum = iota
	channelReportColChanID
	channelReportColPubKey
	channelReportColCapacity
	channelReportColLocal
	channelReportColRemote
	channelReportColRatio
	channelReportColDayIn
	channelReportColDayOut
	channelReportColMonthIn
	channelReportColMonthOut
	channelReportColFeeIn
	channelReportColFeeOut
	channelReportColTotalIn
	channelReportColTotalOut
	channelReportColEffcy
)

// channelReportForwarding holds forwarding totals for a single channel.
type channelReportForwarding struct {
	// DayInSat is the incoming forwarding amount in satoshis over 24h.
	DayInSat int64

	// DayOutSat is the outgoing forwarding amount in satoshis over 24h.
	DayOutSat int64

	// MonthInSat is the incoming forwarding amount in satoshis over 30d.
	MonthInSat int64

	// MonthOutSat is the outgoing forwarding amount in satoshis over 30d.
	MonthOutSat int64

	// MonthFeeInMsat is the incoming forwarding fees in msat over 30d.
	MonthFeeInMsat int64

	// MonthFeeOutMsat is the outgoing forwarding fees in msat over 30d.
	MonthFeeOutMsat int64
}

// channelReportTotals aggregates totals across all channels.
type channelReportTotals struct {
	// Capacity is the total channel capacity in satoshis.
	Capacity int64

	// LocalBalance is the total local balance in satoshis.
	LocalBalance int64

	// RemoteBalance is the total remote balance in satoshis.
	RemoteBalance int64

	// AmountIn is the total received amount in satoshis.
	AmountIn int64

	// AmountOut is the total sent amount in satoshis.
	AmountOut int64

	// DayInSat is the incoming forwarding amount in satoshis over 24h.
	DayInSat int64

	// DayOutSat is the outgoing forwarding amount in satoshis over 24h.
	DayOutSat int64

	// MonthInSat is the incoming forwarding amount in satoshis over 30d.
	MonthInSat int64

	// MonthOutSat is the outgoing forwarding amount in satoshis over 30d.
	MonthOutSat int64

	// MonthFeeInMsat is the incoming forwarding fees in msat over 30d.
	MonthFeeInMsat int64

	// MonthFeeOutMsat is the outgoing forwarding fees in msat over 30d.
	MonthFeeOutMsat int64

	// Ratio is the local balance ratio in percent.
	Ratio float64

	// Efficiency is the flow efficiency in percent.
	Efficiency float64
}

// channelReportRow holds formatted values for a single table row.
type channelReportRow struct {
	// Num is the row number label.
	Num string

	// Active is the active marker (space or dash).
	Active string

	// ChanID is the formatted short channel ID.
	ChanID string

	// PubKey is the shortened peer pubkey.
	PubKey string

	// Capacity is the channel capacity value.
	Capacity string

	// Local is the local balance value.
	Local string

	// Remote is the remote balance value.
	Remote string

	// Ratio is the local balance ratio percent string.
	Ratio string

	// DayIn is the day forwarding in amount.
	DayIn string

	// DayOut is the day forwarding out amount.
	DayOut string

	// MonthIn is the month forwarding in amount.
	MonthIn string

	// MonthOut is the month forwarding out amount.
	MonthOut string

	// FeeIn is the incoming forwarding fee amount.
	FeeIn string

	// FeeOut is the outgoing forwarding fee amount.
	FeeOut string

	// TotalIn is the total received amount.
	TotalIn string

	// TotalOut is the total sent amount.
	TotalOut string

	// Effcy is the efficiency percent string.
	Effcy string
}

// channelReportClient is the subset of RPCs needed to build the report.
type channelReportClient interface {
	// ListChannels returns open channels that match the request.
	ListChannels(context.Context, *lnrpc.ListChannelsRequest,
		...grpc.CallOption) (*lnrpc.ListChannelsResponse, error)

	// ForwardingHistory returns forwarding events for a time window.
	ForwardingHistory(context.Context, *lnrpc.ForwardingHistoryRequest,
		...grpc.CallOption) (*lnrpc.ForwardingHistoryResponse, error)
}

var channelReportCommand = cli.Command{
	Name:     "channelreport",
	Category: "Channels",
	Usage:    "Show channel balances and forwarding cashflow summary.",
	Description: `
	Markers in the Num column:
	  - "-" after the number indicates an inactive channel.
	  - "*" before the number indicates a private channel.
	`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "active_only",
			Usage: "only list channels which are currently active",
		},
		cli.BoolFlag{
			Name: "inactive_only",
			Usage: "only list channels which are currently " +
				"inactive",
		},
		cli.BoolFlag{
			Name:  "public_only",
			Usage: "only list channels which are currently public",
		},
		cli.BoolFlag{
			Name:  "private_only",
			Usage: "only list channels which are currently private",
		},
		cli.StringFlag{
			Name: "peer",
			Usage: "(optional) only display channels with a " +
				"particular peer, accepts hex-encoded " +
				"pubkey prefixes or full pubkeys",
		},
	},
	Action: actionDecorator(channelReport),
}

// channelReport runs the command using CLI flags and prints the report.
func channelReport(ctx *cli.Context) error {
	ctxc := getContext()

	client, cleanUp := getClient(ctx)
	defer cleanUp()

	report, err := channelReportWithClient(
		ctxc, client, time.Now().UTC(), ctx.String("peer"),
		ctx.Bool("active_only"), ctx.Bool("inactive_only"),
		ctx.Bool("public_only"), ctx.Bool("private_only"),
	)
	if err != nil {
		return err
	}

	fmt.Print(report)

	return nil
}

// channelReportWithClient builds the report using explicit filter inputs.
func channelReportWithClient(ctx context.Context, client channelReportClient,
	now time.Time, peer string, activeOnly, inactiveOnly, publicOnly,
	privateOnly bool) (string, error) {

	now = now.UTC()

	peerPrefix, err := parsePeerPrefix(peer)
	if err != nil {
		return "", err
	}

	req := &lnrpc.ListChannelsRequest{
		ActiveOnly:      activeOnly,
		InactiveOnly:    inactiveOnly,
		PublicOnly:      publicOnly,
		PrivateOnly:     privateOnly,
		PeerAliasLookup: true,
	}
	resp, err := client.ListChannels(ctx, req)
	if err != nil {
		return "", err
	}

	filteredChannels := filterChannelsByPeerPrefix(
		resp.Channels, peerPrefix,
	)

	events, err := fetchForwardingEvents(ctx, client, now)
	if err != nil {
		return "", err
	}

	return buildChannelReportTable(filteredChannels, events, now), nil
}

// parsePeerPrefix validates a peer pubkey prefix for local filtering.
func parsePeerPrefix(peer string) (string, error) {
	if peer == "" {
		return "", nil
	}

	if len(peer) > channelReportPeerHexLen {
		return "", fmt.Errorf("invalid --peer pubkey prefix "+
			"length %d exceeds %d",
			len(peer), channelReportPeerHexLen)
	}
	if !isHexString(peer) {
		return "", fmt.Errorf("invalid --peer pubkey prefix: "+
			"%q", peer)
	}

	return strings.ToLower(peer), nil
}

// isHexString reports whether s contains only hexadecimal characters.
func isHexString(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}

	return true
}

// filterChannelsByPeerPrefix filters channels by remote pubkey prefix.
func filterChannelsByPeerPrefix(channels []*lnrpc.Channel,
	peerPrefix string) []*lnrpc.Channel {

	if peerPrefix == "" {
		return channels
	}

	filtered := make([]*lnrpc.Channel, 0, len(channels))
	for _, channel := range channels {
		if strings.HasPrefix(
			strings.ToLower(channel.RemotePubkey), peerPrefix,
		) {
			filtered = append(filtered, channel)
		}
	}

	return filtered
}

// fetchForwardingEvents loads forwarding events over the last 30 days.
func fetchForwardingEvents(ctx context.Context, client channelReportClient,
	now time.Time) ([]*lnrpc.ForwardingEvent, error) {

	start := now.Add(-time.Hour * 24 * 30)
	var (
		offset uint32
		events []*lnrpc.ForwardingEvent
	)

	for {
		req := &lnrpc.ForwardingHistoryRequest{
			StartTime:    uint64(start.Unix()),
			EndTime:      uint64(now.Unix()),
			IndexOffset:  offset,
			NumMaxEvents: channelReportMaxEvents,
		}
		resp, err := client.ForwardingHistory(ctx, req)
		if err != nil {
			return nil, err
		}
		if len(resp.ForwardingEvents) == 0 {
			return events, nil
		}

		events = append(events, resp.ForwardingEvents...)

		if uint32(len(resp.ForwardingEvents)) < channelReportMaxEvents {
			return events, nil
		}
		if resp.LastOffsetIndex == offset {
			return events, nil
		}

		offset = resp.LastOffsetIndex
	}
}

// buildChannelReportTable renders the channel report table as a string.
func buildChannelReportTable(channels []*lnrpc.Channel,
	events []*lnrpc.ForwardingEvent, now time.Time) string {

	forwarding := computeForwardingTotals(events, now)

	sorted := append([]*lnrpc.Channel(nil), channels...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].ChanId > sorted[j].ChanId
	})

	var (
		rows   []channelReportRow
		totals channelReportTotals
	)

	for i, c := range sorted {
		row, fwdTotals := buildChannelReportRow(
			i+1, c, forwarding[c.ChanId],
		)
		rows = append(rows, row)

		totals.Capacity += c.Capacity
		totals.LocalBalance += c.LocalBalance
		totals.RemoteBalance += c.RemoteBalance
		totals.AmountIn += c.TotalSatoshisReceived
		totals.AmountOut += c.TotalSatoshisSent
		totals.DayInSat += fwdTotals.DayInSat
		totals.DayOutSat += fwdTotals.DayOutSat
		totals.MonthInSat += fwdTotals.MonthInSat
		totals.MonthOutSat += fwdTotals.MonthOutSat
		totals.MonthFeeInMsat += fwdTotals.MonthFeeInMsat
		totals.MonthFeeOutMsat += fwdTotals.MonthFeeOutMsat
	}

	if totals.LocalBalance > 0 {
		totals.Ratio = float64(totals.LocalBalance) /
			float64(totals.LocalBalance+totals.RemoteBalance) * 100
		if totals.Capacity > 0 {
			totals.Efficiency = (float64(totals.AmountIn) +
				float64(totals.AmountOut)) /
				float64(totals.Capacity) * 100
		}
	}

	totalsRow := buildChannelReportTotalsRow(len(sorted), totals)

	t := table.NewWriter()
	t.SetStyle(channelReportTableStyle())
	t.SetColumnConfigs(channelReportColumnConfigs())
	t.AppendHeader(table.Row{
		channelReportHeaderNum,
		channelReportHeaderChanID,
		channelReportHeaderPubKey,
		channelReportHeaderCapacity,
		channelReportHeaderLocal,
		channelReportHeaderRemote,
		channelReportHeaderRatio,
		channelReportHeaderDayIn,
		channelReportHeaderOutAmt,
		channelReportHeaderMonthIn,
		channelReportHeaderOutAmt,
		channelReportHeaderFeeIn,
		channelReportHeaderFeeOut,
		channelReportHeaderTotalIn,
		channelReportHeaderOutAmt,
		channelReportHeaderEffcy,
	})

	for _, row := range rows {
		t.AppendRow(row.toTableRow())
	}

	t.AppendSeparator()
	t.AppendRow(totalsRow.toTableRow())

	output := t.Render()
	output = insertChannelReportGroupHeader(output)
	output = removeChannelReportPairSeparators(output)

	return output + "\n"
}

// channelReportGroup describes a grouped header span.
type channelReportGroup struct {
	// Label is the text to center within the span.
	Label string

	// StartCol is the first column index in the span.
	StartCol int

	// EndCol is the last column index in the span.
	EndCol int
}

// channelReportColumnRange describes a half-open column range.
type channelReportColumnRange struct {
	// Start is the first index in the line.
	Start int

	// End is the index after the last character.
	End int
}

// insertChannelReportGroupHeader adds the grouped header line above columns.
func insertChannelReportGroupHeader(output string) string {
	lines := strings.Split(output, "\n")
	headerIdx := findLineIndexContaining(
		lines, channelReportHeaderChanID,
	)
	if headerIdx == -1 {
		return output
	}

	headerLine := lines[headerIdx]
	pipePositions := findPipePositions(headerLine)
	columnRanges := channelReportColumnRanges(headerLine, pipePositions)
	groups := []channelReportGroup{
		{
			Label:    channelReportForwardingLabel,
			StartCol: channelReportColDayIn,
			EndCol:   channelReportColMonthOut,
		},
		{
			Label:    channelReportFeeLabel,
			StartCol: channelReportColFeeIn,
			EndCol:   channelReportColFeeOut,
		},
	}
	groupPipePositions := channelReportGroupPipePositions(
		pipePositions, len(columnRanges), groups,
	)

	groupLine := buildChannelReportGroupLine(
		len(headerLine), groupPipePositions, columnRanges, groups,
	)
	lines = append(
		lines[:headerIdx],
		append([]string{groupLine}, lines[headerIdx:]...)...,
	)

	return strings.Join(lines, "\n")
}

// removeChannelReportPairSeparators removes separators between in/out columns.
func removeChannelReportPairSeparators(output string) string {
	lines := strings.Split(output, "\n")
	headerIdx := findLineIndexContaining(
		lines, channelReportHeaderChanID,
	)
	if headerIdx == -1 {
		return output
	}

	pipePositions := findPipePositions(lines[headerIdx])
	removeAfterCols := []int{
		channelReportColDayIn,
		channelReportColMonthIn,
		channelReportColFeeIn,
		channelReportColTotalIn,
	}

	var removePositions []int
	for _, col := range removeAfterCols {
		if col >= 0 && col < len(pipePositions) {
			removePositions = append(
				removePositions, pipePositions[col],
			)
		}
	}

	for i, line := range lines {
		if !strings.Contains(line, "|") {
			continue
		}

		lineRunes := []rune(line)
		for _, pos := range removePositions {
			if pos >= 0 && pos < len(lineRunes) &&
				lineRunes[pos] == '|' {
				lineRunes[pos] = ' '
			}
		}
		lines[i] = string(lineRunes)
	}

	return strings.Join(lines, "\n")
}

// buildChannelReportGroupLine centers group labels within column ranges.
func buildChannelReportGroupLine(lineLen int, pipePositions []int,
	columnRanges []channelReportColumnRange,
	groups []channelReportGroup) string {

	if lineLen <= 0 {
		return ""
	}

	line := make([]rune, lineLen)
	for i := range line {
		line[i] = ' '
	}

	for _, pos := range pipePositions {
		if pos >= 0 && pos < len(line) {
			line[pos] = '|'
		}
	}

	for _, group := range groups {
		if group.StartCol < 0 ||
			group.EndCol >= len(columnRanges) ||
			group.StartCol > group.EndCol {

			continue
		}

		start := columnRanges[group.StartCol].Start
		end := columnRanges[group.EndCol].End
		if start < 0 || end > len(line) || end <= start {
			continue
		}

		label := group.Label
		if len(label) > end-start {
			label = label[:end-start]
		}

		offset := (end - start - len(label)) / 2
		for idx, ch := range label {
			line[start+offset+idx] = ch
		}
	}

	return string(line)
}

// channelReportColumnRanges returns ranges between pipe separators.
func channelReportColumnRanges(line string,
	pipePositions []int) []channelReportColumnRange {

	ranges := make([]channelReportColumnRange, len(pipePositions)+1)
	start := 0
	for i, pos := range pipePositions {
		ranges[i] = channelReportColumnRange{
			Start: start,
			End:   pos,
		}
		start = pos + 1
	}
	ranges[len(pipePositions)] = channelReportColumnRange{
		Start: start,
		End:   len(line),
	}

	return ranges
}

// channelReportGroupPipePositions selects pipe locations for group headers.
func channelReportGroupPipePositions(pipePositions []int, columnCount int,
	groups []channelReportGroup) []int {

	if len(pipePositions) == 0 || columnCount == 0 {
		return nil
	}

	maxPipeIdx := len(pipePositions) - 1
	positions := make(map[int]struct{})
	for _, group := range groups {
		if group.StartCol > 0 &&
			group.StartCol-1 <= maxPipeIdx {
			positions[pipePositions[group.StartCol-1]] = struct{}{}
		}
		if group.EndCol >= 0 &&
			group.EndCol < columnCount-1 &&
			group.EndCol <= maxPipeIdx {
			positions[pipePositions[group.EndCol]] = struct{}{}
		}
	}

	out := make([]int, 0, len(positions))
	for pos := range positions {
		out = append(out, pos)
	}
	sort.Ints(out)

	return out
}

// findLineIndexContaining returns the index of the first line with substr.
func findLineIndexContaining(lines []string, substr string) int {
	for i, line := range lines {
		if strings.Contains(line, substr) {
			return i
		}
	}

	return -1
}

// findPipePositions returns indexes of pipe separators in a line.
func findPipePositions(line string) []int {
	var positions []int
	for i, ch := range line {
		if ch == '|' {
			positions = append(positions, i)
		}
	}

	return positions
}

// computeForwardingTotals aggregates forwarding events per channel.
func computeForwardingTotals(events []*lnrpc.ForwardingEvent,
	now time.Time) map[uint64]channelReportForwarding {

	now = now.UTC()
	dayStart := now.Add(-time.Hour * 24)
	monthStart := now.Add(-time.Hour * 24 * 30)

	totals := make(map[uint64]channelReportForwarding)
	for _, event := range events {
		eventTime := forwardingEventTime(event)

		if event.ChanIdIn > 0 {
			entry := totals[event.ChanIdIn]
			entry = addForwarding(
				entry, eventTime, dayStart, monthStart,
				lnwire.MilliSatoshi(event.AmtInMsat),
				lnwire.MilliSatoshi(event.FeeMsat),
				true,
			)
			totals[event.ChanIdIn] = entry
		}

		if event.ChanIdOut > 0 {
			entry := totals[event.ChanIdOut]
			entry = addForwarding(
				entry, eventTime, dayStart, monthStart,
				lnwire.MilliSatoshi(event.AmtOutMsat),
				lnwire.MilliSatoshi(event.FeeMsat),
				false,
			)
			totals[event.ChanIdOut] = entry
		}
	}

	return totals
}

// buildChannelReportRow formats a single channel into a report row.
func buildChannelReportRow(index int, channel *lnrpc.Channel,
	fwd channelReportForwarding) (channelReportRow,
	channelReportForwarding) {

	totalIn := channel.TotalSatoshisReceived
	totalOut := channel.TotalSatoshisSent

	ratio := 0.0
	efficiency := 0.0
	if channel.LocalBalance > 0 {
		totalBalance := channel.LocalBalance + channel.RemoteBalance
		ratio = float64(channel.LocalBalance) /
			float64(totalBalance) * 100

		if channel.Capacity > 0 {
			efficiency = (float64(totalIn) + float64(totalOut)) /
				float64(channel.Capacity) * 100
		}
	}

	active := "-"
	if channel.Active {
		active = " "
	}

	num := fmt.Sprintf("%d", index)
	if channel.Private {
		num = "*" + num
	}

	row := channelReportRow{
		Num:      num,
		Active:   active,
		ChanID:   formatShortChanID(channel.ChanId),
		PubKey:   shortPubKey(channel.RemotePubkey),
		Capacity: fmt.Sprintf("%d", channel.Capacity),
		Local:    fmt.Sprintf("%d", channel.LocalBalance),
		Remote:   fmt.Sprintf("%d", channel.RemoteBalance),
		Ratio:    fmt.Sprintf("%d%%", int64(math.Round(ratio))),
		DayIn:    fmt.Sprintf("%d", fwd.DayInSat),
		DayOut:   fmt.Sprintf("%d", fwd.DayOutSat),
		MonthIn:  fmt.Sprintf("%d", fwd.MonthInSat),
		MonthOut: fmt.Sprintf("%d", fwd.MonthOutSat),
		FeeIn:    formatFeeMsat(fwd.MonthFeeInMsat),
		FeeOut:   formatFeeMsat(fwd.MonthFeeOutMsat),
		TotalIn:  fmt.Sprintf("%d", totalIn),
		TotalOut: fmt.Sprintf("%d", totalOut),
		Effcy:    fmt.Sprintf("%d%%", int64(math.Round(efficiency))),
	}

	return row, fwd
}

// buildChannelReportTotalsRow formats the totals row for all channels.
func buildChannelReportTotalsRow(channelCount int,
	totals channelReportTotals) channelReportRow {

	return channelReportRow{
		Num:      fmt.Sprintf("%d", channelCount),
		Active:   " ",
		ChanID:   "",
		PubKey:   "",
		Capacity: fmt.Sprintf("%d", totals.Capacity),
		Local:    fmt.Sprintf("%d", totals.LocalBalance),
		Remote:   fmt.Sprintf("%d", totals.RemoteBalance),
		Ratio:    fmt.Sprintf("%d%%", int64(math.Round(totals.Ratio))),
		DayIn:    fmt.Sprintf("%d", totals.DayInSat),
		DayOut:   fmt.Sprintf("%d", totals.DayOutSat),
		MonthIn:  fmt.Sprintf("%d", totals.MonthInSat),
		MonthOut: fmt.Sprintf("%d", totals.MonthOutSat),
		FeeIn:    formatFeeMsatTotal(totals.MonthFeeInMsat),
		FeeOut:   formatFeeMsatTotal(totals.MonthFeeOutMsat),
		TotalIn:  fmt.Sprintf("%d", totals.AmountIn),
		TotalOut: fmt.Sprintf("%d", totals.AmountOut),
		Effcy: fmt.Sprintf("%d%%",
			int64(math.Round(totals.Efficiency))),
	}
}

// toTableRow converts a channelReportRow into a go-pretty row.
func (row channelReportRow) toTableRow() table.Row {

	return table.Row{
		row.Num + row.Active,
		row.ChanID,
		row.PubKey,
		row.Capacity,
		row.Local,
		row.Remote,
		row.Ratio,
		row.DayIn,
		row.DayOut,
		row.MonthIn,
		row.MonthOut,
		row.FeeIn,
		row.FeeOut,
		row.TotalIn,
		row.TotalOut,
		row.Effcy,
	}
}

// channelReportTableStyle configures minimal separators for the table.
func channelReportTableStyle() table.Style {
	style := table.StyleDefault
	style.Options.DrawBorder = false
	style.Options.SeparateColumns = true
	style.Options.SeparateHeader = true
	style.Options.SeparateFooter = false
	style.Options.SeparateRows = false
	style.Format.Header = text.FormatDefault
	style.Format.Footer = text.FormatDefault
	style.Box.Left = ""
	style.Box.Right = ""
	style.Box.LeftSeparator = ""
	style.Box.RightSeparator = ""
	style.Box.TopSeparator = ""
	style.Box.BottomSeparator = ""
	style.Box.MiddleSeparator = "-"
	style.Box.MiddleHorizontal = "-"
	style.Box.MiddleVertical = "|"
	style.Box.PaddingLeft = ""
	style.Box.PaddingRight = ""

	return style
}

// channelReportColumnConfig builds a column config with custom padding.
func channelReportColumnConfig(col int, align text.Align,
	widthMin int) table.ColumnConfig {

	prefix, suffix := channelReportColumnPadding(col)
	transformer := channelReportPadTransformer(prefix, suffix)

	cfg := table.ColumnConfig{
		Number:            col + 1,
		Align:             align,
		AlignHeader:       align,
		Transformer:       transformer,
		TransformerHeader: transformer,
	}
	if widthMin > 0 {
		cfg.WidthMin = widthMin
	}

	return cfg
}

// channelReportColumnPadding returns the prefix/suffix spacing for a column.
func channelReportColumnPadding(col int) (string, string) {
	prefix := ""
	if channelReportHasSeparatorBefore(col) {
		prefix = " "
	}

	suffix := ""
	if channelReportHasSeparatorAfter(col) {
		suffix = " "
	}

	return prefix, suffix
}

// channelReportHasSeparatorBefore reports whether a column has a left pipe.
func channelReportHasSeparatorBefore(col int) bool {
	switch col {
	case channelReportColDayOut,
		channelReportColMonthOut,
		channelReportColFeeOut,
		channelReportColTotalOut:
		return false
	}

	return col != channelReportColNum
}

// channelReportHasSeparatorAfter reports whether a column has a right pipe.
func channelReportHasSeparatorAfter(col int) bool {
	switch col {
	case channelReportColDayIn,
		channelReportColMonthIn,
		channelReportColFeeIn,
		channelReportColTotalIn:
		return false
	}

	return col != channelReportColEffcy
}

// channelReportPadTransformer adds custom padding to a column value.
func channelReportPadTransformer(prefix, suffix string) text.Transformer {
	return func(val interface{}) string {
		return prefix + fmt.Sprint(val) + suffix
	}
}

// channelReportColumnConfigs defines alignments for channel report columns.
func channelReportColumnConfigs() []table.ColumnConfig {
	feeInMin, feeOutMin := feeHeaderMinWidths()

	return []table.ColumnConfig{
		channelReportColumnConfig(
			channelReportColNum, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColChanID, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColPubKey, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColCapacity, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColLocal, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColRemote, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColRatio, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColDayIn, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColDayOut, text.AlignLeft, 0,
		),
		channelReportColumnConfig(
			channelReportColMonthIn, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColMonthOut, text.AlignLeft, 0,
		),
		channelReportColumnConfig(
			channelReportColFeeIn, text.AlignRight, feeInMin,
		),
		channelReportColumnConfig(
			channelReportColFeeOut, text.AlignLeft, feeOutMin,
		),
		channelReportColumnConfig(
			channelReportColTotalIn, text.AlignRight, 0,
		),
		channelReportColumnConfig(
			channelReportColTotalOut, text.AlignLeft, 0,
		),
		channelReportColumnConfig(
			channelReportColEffcy, text.AlignRight, 0,
		),
	}
}

// feeHeaderMinWidths returns fee column widths honoring the header label.
func feeHeaderMinWidths() (int, int) {
	inWidth := len(channelReportHeaderFeeIn)
	outWidth := len(channelReportHeaderFeeOut)
	pairWidth := inWidth + 1 + outWidth
	if pairWidth >= len(channelReportFeeLabel) {
		return inWidth, outWidth
	}

	extra := len(channelReportFeeLabel) - pairWidth
	leftPad := extra / 2
	rightPad := extra - leftPad

	return inWidth + leftPad, outWidth + rightPad
}

// addForwarding adds a forwarding event into the day and month totals.
func addForwarding(entry channelReportForwarding, eventTime, dayStart,
	monthStart time.Time, amountMsat, feeMsat lnwire.MilliSatoshi,
	isIncoming bool) channelReportForwarding {

	if eventTime.After(dayStart) {
		if isIncoming {
			entry.DayInSat += int64(amountMsat.ToSatoshis())
		} else {
			entry.DayOutSat += int64(amountMsat.ToSatoshis())
		}
	}

	if eventTime.After(monthStart) {
		if isIncoming {
			entry.MonthInSat += int64(amountMsat.ToSatoshis())
			entry.MonthFeeInMsat += int64(feeMsat)
		} else {
			entry.MonthOutSat += int64(amountMsat.ToSatoshis())
			entry.MonthFeeOutMsat += int64(feeMsat)
		}
	}

	return entry
}

// forwardingEventTime returns the event timestamp with ns precision if set.
func forwardingEventTime(event *lnrpc.ForwardingEvent) time.Time {
	if event.GetTimestampNs() > 0 {
		return time.Unix(0, int64(event.GetTimestampNs())).UTC()
	}

	return time.Unix(int64(event.GetTimestamp()), 0).UTC()
}

// formatShortChanID renders a short channel ID with fixed-width fields to keep
// column alignment stable. It zero-pads the tx index to 4 digits (e.g.
// 114:0003:0) instead of using lnwire.ShortChannelID.String(), which does not
// pad.
func formatShortChanID(chanID uint64) string {
	scid := lnwire.NewShortChanIDFromInt(chanID)
	return fmt.Sprintf("%7d:%04d:%1d",
		scid.BlockHeight, scid.TxIndex, scid.TxPosition,
	)
}

// shortPubKey returns a shortened hex pubkey for table display.
func shortPubKey(pubKey string) string {
	pubKeyBytes, err := hex.DecodeString(pubKey)
	if err == nil && len(pubKeyBytes) >= 4 {
		return hex.EncodeToString(pubKeyBytes[:4])
	}

	if len(pubKey) > 8 {
		return pubKey[:8]
	}

	return pubKey
}

// formatFeeMsat renders a fee in msat using sat precision for small values.
func formatFeeMsat(feeMsat int64) string {
	if feeMsat == 0 {
		return "0"
	}

	if feeMsat < 100*1000 {
		sat := feeMsat / 1000
		rem := feeMsat % 1000

		return fmt.Sprintf("%d.%03d", sat, rem)
	}

	return fmt.Sprintf("%d", roundBankersMsatToSat(feeMsat))
}

// formatFeeMsatTotal renders a fee in msat as rounded satoshis.
func formatFeeMsatTotal(feeMsat int64) string {
	return fmt.Sprintf("%d", roundBankersMsatToSat(feeMsat))
}

// roundBankersMsatToSat rounds msat to sat using bankers rounding.
func roundBankersMsatToSat(msat int64) int64 {
	sat := msat / 1000
	rem := msat % 1000

	switch {
	case rem < 500:
		return sat

	case rem > 500:
		return sat + 1

	case sat%2 == 0:
		return sat

	default:
		return sat + 1
	}
}
