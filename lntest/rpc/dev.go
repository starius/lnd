package rpc

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc/devrpc"
)

// =====================
// DevClient related RPCs.
// =====================

// ToggleNetwork switches networking in brontide on/off.
func (h *HarnessRPC) ToggleNetwork(broken bool) {
	ctxt, cancel := context.WithTimeout(h.runCtx, DefaultTimeout)
	defer cancel()

	_, err := h.Dev.ToggleNetwork(ctxt, &devrpc.ToggleNetworkRequest{
		BrokenNetwork: broken,
	})
	h.NoError(err, "ToggleNetwork")
}
