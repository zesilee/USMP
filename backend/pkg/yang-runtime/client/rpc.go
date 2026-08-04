package client

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RPCInput is one input leaf of an rpc execution (DP-10).
type RPCInput struct {
	Name  string
	Value string
}

// RPCResult is the outcome of an rpc execution (DP-10 / RPC-03).
type RPCResult struct {
	Reply     []byte // raw <rpc-reply> body
	OK        bool   // <ok/> present
	Error     error  // transport error or <rpc-error>
	Timestamp time.Time
}

// ExecuteRPC sends a NETCONF <rpc> carrying rpcName (namespaced by the module)
// with the given input leaves, and returns the parsed reply (DP-10).
//
// Unlike Get/Set, it does NOT auto-retry on a transport error: an rpc is an
// operation with device side effects (restart/clear/…), and re-sending after a
// mid-flight failure could double-execute. On transport error we mark the
// connection dead and surface the failure to the caller.
func (c *NETCONFClient) ExecuteRPC(ctx context.Context, namespace, rpcName string, inputs []RPCInput) (*RPCResult, error) {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	payload := buildRPCPayload(namespace, rpcName, inputs)

	backend, err := c.ensureConnected()
	if err != nil {
		return &RPCResult{Error: err, Timestamp: time.Now()}, err
	}

	resp, err := backend.RPC(ctx, payload)
	if err != nil {
		if isTransportError(err) {
			c.markDisconnected()
		}
		return &RPCResult{Error: err, Timestamp: time.Now()}, err
	}
	if len(resp.Result) == 0 {
		e := fmt.Errorf("empty rpc reply")
		return &RPCResult{Error: e, Timestamp: time.Now()}, e
	}

	res := parseRPCReply(resp.Result)
	res.Timestamp = time.Now()
	return res, res.Error
}

// buildRPCPayload renders the <rpc> body: the operation element namespaced by the
// module, carrying each input leaf. Values are XML-escaped. Input order is
// preserved (caller-supplied) for deterministic payloads.
func buildRPCPayload(namespace, rpcName string, inputs []RPCInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<%s xmlns=%q>", rpcName, namespace)
	for _, in := range inputs {
		fmt.Fprintf(&b, "<%s>%s</%s>", in.Name, xmlEscapeString(in.Value), in.Name)
	}
	fmt.Fprintf(&b, "</%s>", rpcName)
	return b.String()
}

// xmlEscapeString escapes XML metacharacters in an input value.
func xmlEscapeString(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

// parseRPCReply classifies a NETCONF <rpc-reply> body into ok / data / rpc-error.
// scrapligo returns the reply content in resp.Result; we match structurally on
// the well-known reply elements (robust to whitespace and self-closing tags).
func parseRPCReply(reply string) *RPCResult {
	res := &RPCResult{Reply: []byte(reply)}
	if strings.Contains(reply, "<rpc-error") {
		res.Error = fmt.Errorf("rpc-error: %s", extractErrorMessage(reply))
		return res
	}
	if strings.Contains(reply, "<ok/>") || strings.Contains(reply, "<ok>") || strings.Contains(reply, "<ok ") {
		res.OK = true
	}
	return res
}

// extractErrorMessage pulls <error-message> text out of an <rpc-error>, falling
// back to the raw reply when absent.
func extractErrorMessage(reply string) string {
	const open, close = "<error-message", "</error-message>"
	i := strings.Index(reply, open)
	if i < 0 {
		return strings.TrimSpace(reply)
	}
	// skip to end of the opening tag
	gt := strings.Index(reply[i:], ">")
	if gt < 0 {
		return strings.TrimSpace(reply)
	}
	start := i + gt + 1
	j := strings.Index(reply[start:], close)
	if j < 0 {
		return strings.TrimSpace(reply[start:])
	}
	return strings.TrimSpace(reply[start : start+j])
}
