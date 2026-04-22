package netsim

import (
	"encoding/xml"
	"fmt"
)

// RPCMessage represents an incoming RPC request
type RPCMessage struct {
	XMLName   xml.Name  `xml:"rpc"`
	MessageID string    `xml:"message-id,attr"`
	Operations []Operation `xml:",any"`
}

// Operation represents a specific operation in the RPC request
type Operation struct {
	XMLName xml.Name
	Content []byte `xml:",innerxml"`
}

// RPCResponse represents an outgoing RPC response
type RPCResponse struct {
	XMLName   xml.Name `xml:"rpc-reply"`
	MessageID string   `xml:"message-id,attr"`
	Content   []byte   `xml:",innerxml"`
}

// RPCHandler handles incoming RPC requests
type RPCHandler struct {
	server *Server
}

// NewRPCHandler creates a new RPC handler
func NewRPCHandler(server *Server) *RPCHandler {
	return &RPCHandler{
		server: server,
	}
}

// HandleRPC processes an RPC request
func (h *RPCHandler) HandleRPC(msg []byte, session *Session) error {
	var rpc RPCMessage
	if err := xml.Unmarshal(msg, &rpc); err != nil {
		return h.sendError(rpc.MessageID, fmt.Sprintf("malformed request: %v", err), session)
	}

	// Route to appropriate handler
	if len(rpc.Operations) == 0 {
		return h.sendError(rpc.MessageID, "no operation found", session)
	}

	// Check scenario error injection
	opName := rpc.Operations[0].XMLName.Local
	sc := h.server.GetScenario()
	if err, ok := sc.GetErrorForRPC(opName); ok {
		return h.sendError(rpc.MessageID, err.Error(), session)
	}
	if sc.IsTimeoutForRPC(opName) {
		// Never respond
		select {}
	}

	// Handle operation
	err := h.handleOperation(opName, rpc.Operations[0].Content, rpc.MessageID, session)
	if err != nil {
		return h.sendError(rpc.MessageID, err.Error(), session)
	}

	return nil
}

func (h *RPCHandler) handleOperation(opName string, content []byte, msgID string, session *Session) error {
	switch opName {
	case "get-config":
		return h.handleGetConfig(content, msgID, session)
	case "edit-config":
		return h.handleEditConfig(content, msgID, session)
	case "commit":
		return h.handleCommit(msgID, session)
	case "lock":
		return h.sendOK(msgID, session)
	case "unlock":
		return h.sendOK(msgID, session)
	case "close-session":
		return h.handleCloseSession(msgID, session)
	default:
		return fmt.Errorf("unsupported operation: %s", opName)
	}
}

func (h *RPCHandler) sendOK(msgID string, session *Session) error {
	resp := fmt.Sprintf(`<rpc-reply message-id="%s"><ok/></rpc-reply>`, msgID)
	return session.WriteMessage([]byte(resp))
}

func (h *RPCHandler) sendError(msgID, errMsg string, session *Session) error {
	// Escape the error message properly
	var buf []byte
	for _, c := range []byte(errMsg) {
		switch c {
		case '<':
			buf = append(buf, []byte("&lt;")...)
		case '>':
			buf = append(buf, []byte("&gt;")...)
		case '&':
			buf = append(buf, []byte("&amp;")...)
		default:
			buf = append(buf, c)
		}
	}
	resp := fmt.Sprintf(`<rpc-reply message-id="%s"><rpc-error><error-type>protocol</error-type><error-tag>operation-failed</error-tag><error-message>%s</error-message></rpc-error></rpc-reply>`, msgID, string(buf))
	return session.WriteMessage([]byte(resp))
}

// handleCloseSession handles close-session request
func (h *RPCHandler) handleCloseSession(msgID string, session *Session) error {
	err := h.sendOK(msgID, session)
	session.Close()
	return err
}
