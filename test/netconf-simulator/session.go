package netsim

import (
	"encoding/xml"
	"log"

	"golang.org/x/crypto/ssh"
)

// HelloMessage represents a NETCONF hello message
type HelloMessage struct {
	XMLName xml.Name `xml:"hello"`
	Capabilities []Capability `xml:"capabilities>capability"`
}

// Capability represents a NETCONF capability
type Capability struct {
	URI string `xml:",chardata"`
}

// Session represents a NETCONF session
type Session struct {
	server  *Server
	channel ssh.Channel
	framer  *Framer
}

// NewSession creates a new NETCONF session
func NewSession(server *Server, channel ssh.Channel) *Session {
	return &Session{
		server:  server,
		channel: channel,
		framer:  NewFramer(channel, Base10), // default to base:1.0, will negotiate
	}
}

// Serve processes the NETCONF session
func (s *Session) Serve() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("session panic recovered: %v", r)
		}
	}()

	// 1. Send server hello
	if err := s.sendHello(); err != nil {
		log.Printf("failed to send hello: %v", err)
		return
	}

	// 2. Read client hello
	clientHello, err := s.readHello()
	if err != nil {
		log.Printf("failed to read hello: %v", err)
		return
	}

	// 3. Negotiate framing version
	s.negotiateFraming(clientHello)

	// 4. Process RPC requests
	s.processRPCs()
}

// Close closes the session
func (s *Session) Close() error {
	return s.channel.Close()
}

// ReadMessage reads a complete message from the session
func (s *Session) ReadMessage() ([]byte, error) {
	return s.framer.ReadMessage()
}

// WriteMessage writes a complete message to the session
func (s *Session) WriteMessage(msg []byte) error {
	return s.framer.WriteMessage(msg)
}

func (s *Session) sendHello() error {
	hello := HelloMessage{
		Capabilities: []Capability{
			{URI: "urn:ietf:params:netconf:base:1.0"},
			{URI: "urn:ietf:params:netconf:base:1.1"},
			{URI: "urn:ietf:params:netconf:capability:writable-running:1.0"},
			{URI: "urn:ietf:params:netconf:capability:candidate:1.0"},
			{URI: "urn:ietf:params:netconf:capability:commit:1.0"},
		},
	}

	output, err := xml.MarshalIndent(hello, "", "  ")
	if err != nil {
		return err
	}

	fullMsg := append([]byte(xml.Header), output...)
	return s.WriteMessage(fullMsg)
}

func (s *Session) readHello() (*HelloMessage, error) {
	msg, err := s.ReadMessage()
	if err != nil {
		return nil, err
	}

	var hello HelloMessage
	if err := xml.Unmarshal(msg, &hello); err != nil {
		return nil, err
	}

	return &hello, nil
}

func (s *Session) negotiateFraming(clientHello *HelloMessage) {
	hasBase11 := false
	for _, cap := range clientHello.Capabilities {
		if cap.URI == "urn:ietf:params:netconf:base:1.1" {
			hasBase11 = true
			break
		}
	}

	if hasBase11 {
		s.framer.SetVersion(Base11)
	} else {
		s.framer.SetVersion(Base10)
	}
}

func (s *Session) processRPCs() {
	handler := NewRPCHandler(s.server)
	for {
		sc := s.server.GetScenario()
		if sc.IsTimeoutForRPC("*") {
			// Hang forever
			select {}
		}

		msg, err := s.ReadMessage()
		if err != nil {
			return
		}

		delay := sc.GetResponseDelay()
		if delay > 0 {
			// TODO: implement delay
		}

		if err := handler.HandleRPC(msg, s); err != nil {
			log.Printf("RPC handling error: %v", err)
			continue
		}
	}
}
