package netconfsim

import (
	"encoding/xml"
	"sync"
)

// RecordedRPC is a custom (module) rpc the simulator received (NS-09), captured
// for end-to-end test assertions.
type RecordedRPC struct {
	Op     string
	Inputs map[string]string
}

// rpcLog is a thread-safe record of custom rpcs the simulator has executed.
type rpcLog struct {
	mu    sync.Mutex
	calls []RecordedRPC
}

func newRPCLog() *rpcLog { return &rpcLog{} }

func (l *rpcLog) record(op string, inputs map[string]string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, RecordedRPC{Op: op, Inputs: inputs})
}

func (l *rpcLog) snapshot() []RecordedRPC {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]RecordedRPC, len(l.calls))
	copy(out, l.calls)
	return out
}

// stdOps are base-NETCONF operations that are NOT module rpcs; they keep their
// existing "return ok" fallback and are never recorded as custom rpcs.
var stdOps = map[string]bool{
	"lock": true, "unlock": true, "close-session": true, "kill-session": true,
	"validate": true, "copy-config": true, "delete-config": true,
	"cancel-commit": true, "create-subscription": true,
}

// customRPC decodes an <rpc> whose operation is a module rpc (not a base op),
// returning the operation local-name and its input leaves. ok is false for base
// ops or malformed envelopes.
func customRPC(msg string) (op string, inputs map[string]string, ok bool) {
	var env struct {
		XMLName xml.Name `xml:"rpc"`
		Op      struct {
			XMLName xml.Name
			Leaves  []struct {
				XMLName xml.Name
				Value   string `xml:",chardata"`
			} `xml:",any"`
		} `xml:",any"`
	}
	if err := xml.Unmarshal([]byte(msg), &env); err != nil {
		return "", nil, false
	}
	name := env.Op.XMLName.Local
	if name == "" || stdOps[name] {
		return "", nil, false
	}
	inputs = make(map[string]string, len(env.Op.Leaves))
	for _, lf := range env.Op.Leaves {
		inputs[lf.XMLName.Local] = lf.Value
	}
	return name, inputs, true
}

// handleCustomRPC records a custom module rpc and replies. A scenario
// ErrorOnRPC[op] injects an <rpc-error> for negative-path integration tests;
// otherwise it returns <ok/> (NS-09). The simulator carries no YANG schema, so
// mandatory/leafref semantics are validated at the API layer (RPC-03), not here.
func (s *sshServer) handleCustomRPC(op string, inputs map[string]string, msgID string) string {
	if s.rpcLog != nil {
		s.rpcLog.record(op, inputs)
	}
	if err, ok := s.scenario.ErrorOnRPC[op]; ok && err != nil {
		return errorReply(msgID, err.Error())
	}
	return okReply(msgID)
}
