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
	"strings"

	"golang.org/x/crypto/ssh"
)

type sshServer struct {
	config    *ssh.ServerConfig
	datastore *Datastore
	scenario  *ScenarioConfig
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
	hello := &Hello{
		XMLName: xml.Name{Space: "urn:ietf:params:xml:ns:netconf:base:1.0 hello"},
		SessionID: 1,
		Capabilities: Capabilities{
			Capabilities: []Capability{
				{URN: "urn:ietf:params:netconf:base:1.0"},
			},
		},
	}

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

		log.Printf("Got request: %.100s", msg)
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
	// Extract message-id
	msgID := extractMessageID(msg)

	// Check what RPC it is
	switch {
	case strings.Contains(msg, "<get-config"):
		return s.handleGetConfig(msg, msgID)
	case strings.Contains(msg, "<edit-config"):
		return s.handleEditConfig(msg, msgID)
	case strings.Contains(msg, "<commit"):
		return s.handleCommit(msg, msgID)
	default:
		// Return ok for unknown RPC
		return okReply(msgID)
	}
}

// Hello represents a NETCONF hello message.
type Hello struct {
	XMLName      xml.Name       `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 hello"`
	Capabilities Capabilities   `xml:"capabilities"`
	SessionID    int            `xml:"session-id,omitempty"`
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

	config := s.datastore.GetXML(source)
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
	targetIsCandidate := strings.Contains(msg, `<target><candidate/>`)

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

	// Update candidate or running directly
	if targetIsCandidate {
		err := s.datastore.SetCandidate([]byte(configContent))
		if err != nil {
			return errorReply(msgID, err.Error())
		}
	} else {
		// If directly editing running, update immediately
		err := s.datastore.SetCandidate([]byte(configContent))
		if err != nil {
			return errorReply(msgID, err.Error())
		}
		_ = s.datastore.Commit()
	}

	return okReply(msgID)
}

func (s *sshServer) handleCommit(msg, msgID string) string {
	// Check for scenario error
	if err, ok := s.scenario.ErrorOnRPC["commit"]; ok {
		return errorReply(msgID, err.Error())
	}

	err := s.datastore.Commit()
	if err != nil {
		return errorReply(msgID, err.Error())
	}
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
