package netconfsim

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshServer struct {
	config    *ssh.ServerConfig
	store     *treeDatastore
	scenario  *ScenarioConfig
	extraCaps []string // extra YANG-module capabilities advertised in hello
	done      chan struct{}
}

func generateSigner() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

func (s *sshServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Discard out-of-band requests
	go ssh.DiscardRequests(reqs)

	// Handle channels (only accept session channels)
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		ch, reqs, err := newChan.Accept()
		if err != nil {
			return
		}

		// Handle requests - accept subsystem netconf request
		// subsystem request payload is 4-byte big-endian length followed by name
		go func() {
			for req := range reqs {
				if req.Type == "subsystem" && len(req.Payload) >= 4 {
					// extract length, check if it's "netconf"
					name := string(req.Payload[4:])
					if name == "netconf" {
						_ = req.Reply(true, nil)
						continue
					}
				}
				_ = req.Reply(false, nil)
			}
		}()

		s.handleSession(ch)
		return
	}
}

func (s *sshServer) handleSession(ch ssh.Channel) {
	defer ch.Close()

	reader := bufio.NewReader(ch)

	// Send server hello
	hello := buildHello(1, s.extraCaps, !s.scenario.DisableConfirmedCommit)

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(hello); err != nil {
		log.Printf("encode hello failed: %v", err)
		return
	}
	// Flush the encoder to make sure all data is written to buffer
	if err := encoder.Flush(); err != nil {
		log.Printf("flush hello failed: %v", err)
		return
	}
	// Add EOM marker
	buf.WriteString("]]>]]>")

	if _, err := ch.Write(buf.Bytes()); err != nil {
		log.Printf("write hello failed: %v", err)
		return
	}
	log.Printf("Sent server hello, %d bytes", buf.Len())

	// Read client hello
	clientHello, err := readMessage(reader)
	if err != nil {
		log.Printf("read client hello failed: %v", err)
		return
	}
	log.Printf("Received client hello, %d bytes: %.100s", len(clientHello), clientHello)

	// Handle requests until connection close
	for {
		select {
		case <-s.done:
			return
		default:
		}

		msg, err := readMessage(reader)
		if err == io.EOF {
			log.Printf("readMessage got EOF, exiting")
			return
		}
		if err != nil {
			log.Printf("readMessage failed: %v", err)
			return
		}

		log.Printf("Got request: %.500s", msg)
		response := s.handleRequest(msg)
		if response != "" {
			response += "]]>]]>"
			if _, err := ch.Write([]byte(response)); err != nil {
				log.Printf("write response failed: %v", err)
				return
			}
			log.Printf("Sent response, %d bytes", len(response))
		}
	}
}

func (s *sshServer) handleRequest(msg string) string {
	// Extract message-id (kept as a raw-string scan so it works even when the
	// RPC body is malformed and classifyRPC falls back to rpcUnknown).
	msgID := extractMessageID(msg)

	// Dispatch by structurally decoding the RPC envelope rather than substring
	// matching, so an RPC keyword appearing in element content cannot mislead.
	switch classifyRPC(msg) {
	case rpcGetConfig:
		return s.handleGetConfig(msg, msgID)
	case rpcEditConfig:
		return s.handleEditConfig(msg, msgID)
	case rpcCommit:
		return s.handleCommit(msg, msgID)
	case rpcDiscardChanges:
		return s.handleDiscardChanges(msg, msgID)
	default:
		// Return ok for unsupported/unknown RPCs (lock/unlock/get/…).
		return okReply(msgID)
	}
}

// rpcKind identifies a decoded NETCONF RPC for dispatch.
type rpcKind int

const (
	rpcUnknown rpcKind = iota
	rpcGetConfig
	rpcEditConfig
	rpcCommit
	rpcDiscardChanges
)

// rpcEnvelope decodes just enough of an <rpc> element to identify the operation.
// Each operation is a pointer field so a nil check reports element presence; the
// bodies are still parsed from the raw string by the individual handlers until
// the datastore switch (T5).
type rpcEnvelope struct {
	XMLName        xml.Name  `xml:"rpc"`
	GetConfig      *struct{} `xml:"get-config"`
	EditConfig     *struct{} `xml:"edit-config"`
	Commit         *struct{} `xml:"commit"`
	DiscardChanges *struct{} `xml:"discard-changes"`
}

// classifyRPC returns the kind of the RPC by structurally decoding the envelope.
// Malformed XML or a non-rpc root yields rpcUnknown (graceful fallback, R08).
func classifyRPC(msg string) rpcKind {
	var env rpcEnvelope
	if err := xml.Unmarshal([]byte(msg), &env); err != nil {
		return rpcUnknown
	}
	switch {
	case env.GetConfig != nil:
		return rpcGetConfig
	case env.EditConfig != nil:
		return rpcEditConfig
	case env.Commit != nil:
		return rpcCommit
	case env.DiscardChanges != nil:
		return rpcDiscardChanges
	default:
		return rpcUnknown
	}
}

// buildHello constructs the server hello advertising base:1.0 plus the
// :candidate and :writable-running capabilities. base:1.1 is intentionally not
// advertised so scrapligo negotiates 1.0 EOM framing (design D4 / T0.3).
// confirmedCommit gates the :confirmed-commit capability (NS-07 能力开关).
func buildHello(sessionID int, extraCaps []string, confirmedCommit bool) *Hello {
	caps := []Capability{
		{URN: "urn:ietf:params:netconf:base:1.0"},
		{URN: "urn:ietf:params:netconf:capability:candidate:1.0"},
		{URN: "urn:ietf:params:netconf:capability:writable-running:1.0"},
	}
	if confirmedCommit {
		caps = append(caps, Capability{URN: "urn:ietf:params:netconf:capability:confirmed-commit:1.1"})
	}
	for _, c := range extraCaps {
		caps = append(caps, Capability{URN: c})
	}
	return &Hello{
		XMLName:      xml.Name{Space: "urn:ietf:params:xml:ns:netconf:base:1.0 hello"},
		SessionID:    sessionID,
		Capabilities: Capabilities{Capabilities: caps},
	}
}

// Hello represents a NETCONF hello message.
type Hello struct {
	XMLName      xml.Name     `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 hello"`
	Capabilities Capabilities `xml:"capabilities"`
	SessionID    int          `xml:"session-id,omitempty"`
}

// Capabilities contains capability list.
type Capabilities struct {
	Capabilities []Capability `xml:"capability"`
}

// Capability is a single capability URN.
type Capability struct {
	URN string `xml:",chardata"`
}

func readMessage(r *bufio.Reader) (string, error) {
	var builder strings.Builder
	eom := "]]>]]>"
	found := false

	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return "", err
		}

		builder.Write(line)
		if strings.Contains(string(line), eom) {
			found = true
			break
		}
		// Check if EOM is at the end after adding newline
		current := builder.String()
		if len(current) >= 5 && strings.HasSuffix(current, eom) {
			found = true
			break
		}
		builder.WriteByte('\n')
	}

	if !found {
		return "", io.EOF
	}

	content := builder.String()
	content = strings.TrimSuffix(content, "]]>]]>")
	return content, nil
}

func extractMessageID(msg string) string {
	const idAttr = `message-id="`
	start := strings.Index(msg, idAttr)
	if start == -1 {
		return ""
	}
	start += len(idAttr)
	end := strings.Index(msg[start:], `"`)
	if end == -1 {
		return ""
	}
	return msg[start : start+end]
}

func okReply(msgID string) string {
	return fmt.Sprintf(`<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="%s"><ok/></rpc-reply>`, msgID)
}

func (s *sshServer) handleGetConfig(msg, msgID string) string {
	log.Printf("handleGetConfig: msg: %.200s", msg)
	// Extract source (running/candidate)
	source := "running"
	if strings.Contains(msg, `<source><candidate/>`) {
		source = "candidate"
	}

	var config []byte
	if source == "candidate" {
		config = s.store.GetCandidate()
	} else {
		config = s.store.GetRunning()
	}
	response := fmt.Sprintf(`<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="%s"><data>%s</data></rpc-reply>`, msgID, config)
	log.Printf("handleGetConfig: full response:\n%s", response)
	log.Printf("handleGetConfig: response len: %d", len(response))
	return response
}

func (s *sshServer) handleEditConfig(msg, msgID string) string {
	log.Printf("handleEditConfig: received request msg: %.200s", msg)
	// Check for scenario error
	if err, ok := s.scenario.ErrorOnRPC["edit-config"]; ok {
		return errorReply(msgID, err.Error())
	}

	// Extract target - for most cases it's candidate
	// Handle both <candidate/> and <candidate></candidate> formats
	hasCandidateTarget := strings.Contains(msg, "<candidate")
	targetIsRunning := strings.Contains(msg, "<running")
	targetIsCandidate := hasCandidateTarget && !targetIsRunning

	// Extract the config content
	// Logic:
	// 1. Get everything after closing </target> in <edit-config>
	// 2. Trim whitespace, if it starts with <config, extract the content inside <config>...</config>
	// 3. Otherwise, the entire content after </target> is the config content
	// Handles both wrapped and unwrapped content correctly
	editConfigStart := strings.Index(msg, "<edit-config")
	if editConfigStart == -1 {
		log.Printf("handleEditConfig: no <edit-config> found: %.200s", msg)
		return errorReply(msgID, "invalid edit-config")
	}

	// Find closing </target> after edit-config starts
	endTargetRelative := strings.Index(msg[editConfigStart:], "</target>")
	if endTargetRelative == -1 {
		log.Printf("handleEditConfig: missing </target> in edit-config: %.200s", msg)
		return errorReply(msgID, "missing target")
	}
	contentStart := editConfigStart + endTargetRelative + len("</target>")

	// Extract the entire content after target, trim whitespace
	content := strings.TrimSpace(msg[contentStart:])
	// Remove closing </edit-config> if it's at the end
	if idx := strings.Index(content, "</edit-config>"); idx != -1 {
		content = strings.TrimSpace(content[:idx])
	}

	var configContent string
	if strings.HasPrefix(content, "<config") {
		// Has wrapping <config> tag - extract content inside it
		// Find the closing </config> at the end (simplified - works for our use case)
		start := strings.Index(content, ">") + 1
		end := strings.LastIndex(content, "</config>")
		if end == -1 {
			log.Printf("handleEditConfig: missing closing </config> in content: %.200s", content)
			return errorReply(msgID, "missing closing config tag")
		}
		configContent = content[start:end]
	} else {
		// No wrapping config tag - entire content is config content
		configContent = content
	}
	configContent = strings.TrimSpace(configContent)
	log.Printf("handleEditConfig: extracted config: %.500s", configContent)

	// Per-operation edit-config 语义（RFC 6241 §7.2，treeDatastore.EditConfig）：
	// merge 默认 + 显式 operation（create/delete/remove/replace）。这是 editconfig.go
	// 预留的 "deliberate later adoption"——删除语义落地（DP-07）时接线：整树替换会把
	// 删除报文原样并入并抹掉未提交的兄弟条目/其它模块子树。delete-of-missing 等错误
	// 通过 rpc-error 透出（data-missing）。
	if targetIsCandidate {
		if err := s.store.EditConfig([]byte(configContent)); err != nil {
			return errorReply(msgID, err.Error())
		}
	} else {
		// If directly editing running, update immediately
		if err := s.store.EditConfig([]byte(configContent)); err != nil {
			return errorReply(msgID, err.Error())
		}
		_ = s.store.Commit()
	}

	return okReply(msgID)
}

func (s *sshServer) handleCommit(msg, msgID string) string {
	// Check for scenario error
	if err, ok := s.scenario.ErrorOnRPC["commit"]; ok {
		return errorReply(msgID, err.Error())
	}

	// NS-07: <commit><confirmed/> starts a confirmation window; a plain
	// <commit/> inside the window is the confirming commit.
	if confirmed, timeout := parseConfirmedCommit(msg); confirmed {
		if s.scenario.DisableConfirmedCommit {
			return errorReply(msgID, "confirmed commit not supported")
		}
		if err := s.store.CommitConfirmed(timeout); err != nil {
			return errorReply(msgID, err.Error())
		}
		return okReply(msgID)
	}
	if s.store.ConfirmCommit() {
		return okReply(msgID)
	}

	err := s.store.Commit()
	if err != nil {
		return errorReply(msgID, err.Error())
	}
	return okReply(msgID)
}

// parseConfirmedCommit reports whether the commit RPC carries <confirmed/>,
// and its confirm-timeout (seconds; RFC 6241 default 600).
func parseConfirmedCommit(msg string) (bool, time.Duration) {
	var env struct {
		XMLName xml.Name `xml:"rpc"`
		Commit  *struct {
			Confirmed      *struct{} `xml:"confirmed"`
			ConfirmTimeout string    `xml:"confirm-timeout"`
		} `xml:"commit"`
	}
	if err := xml.Unmarshal([]byte(msg), &env); err != nil || env.Commit == nil || env.Commit.Confirmed == nil {
		return false, 0
	}
	timeout := 600 * time.Second
	if v := strings.TrimSpace(env.Commit.ConfirmTimeout); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}
	return true, timeout
}

func (s *sshServer) handleDiscardChanges(msg, msgID string) string {
	// Check for scenario error
	if err, ok := s.scenario.ErrorOnRPC["discard-changes"]; ok {
		return errorReply(msgID, err.Error())
	}

	s.store.DiscardCandidate()
	return okReply(msgID)
}

func errorReply(msgID, errMsg string) string {
	return fmt.Sprintf(
		"<rpc-reply xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\" message-id=\"%s\">"+
			"<rpc-error>"+
			"<error-type>protocol</error-type>"+
			"<error-tag>operation-failed</error-tag>"+
			"<error-severity>error</error-severity>"+
			"<error-message>%s</error-message>"+
			"</rpc-error>"+
			"</rpc-reply>",
		msgID, xmlEscape(errMsg),
	)
}

func xmlEscape(s string) string {
	var builder strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		case '&':
			builder.WriteString("&amp;")
		case '"':
			builder.WriteString("&quot;")
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
