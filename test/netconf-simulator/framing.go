package netsim

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// FramingVersion represents the NETCONF framing version
type FramingVersion int

const (
	// Base10 is NETCONF base:1.0 framing using ]]>]]> as message delimiter
	Base10 FramingVersion = iota
	// Base11 is NETCONF base:1.1 framing using chunked encoding
	Base11
)

// Framer handles message framing according to NETCONF version
type Framer struct {
	reader  *bufio.Reader
	writer  io.Writer
	version FramingVersion
}

// NewFramer creates a new framer
func NewFramer(rw io.ReadWriter, version FramingVersion) *Framer {
	return &Framer{
		reader:  bufio.NewReader(rw),
		writer:  rw,
		version: version,
	}
}

// SetVersion sets the framing version
func (f *Framer) SetVersion(version FramingVersion) {
	f.version = version
}

// ReadMessage reads a complete message
func (f *Framer) ReadMessage() ([]byte, error) {
	switch f.version {
	case Base10:
		return f.readBase10()
	case Base11:
		return f.readBase11()
	default:
		return nil, fmt.Errorf("unknown framing version: %d", f.version)
	}
}

// WriteMessage writes a complete message
func (f *Framer) WriteMessage(msg []byte) error {
	switch f.version {
	case Base10:
		return f.writeBase10(msg)
	case Base11:
		return f.writeBase11(msg)
	default:
		return fmt.Errorf("unknown framing version: %d", f.version)
	}
}

func (f *Framer) readBase10() ([]byte, error) {
	var buf []byte
	for {
		line, isPrefix, err := f.reader.ReadLine()
		if err != nil {
			return nil, err
		}
		buf = append(buf, line...)

		// Check for ]]>]]> delimiter at the end after adding this line
		if len(buf) >= 6 {
			suffix := string(buf[len(buf)-6:])
			if suffix == "]]>]]>" {
				return buf[:len(buf)-6], nil
			}
		}

		if !isPrefix {
			buf = append(buf, '\n')
			// Check again after adding newline
			if len(buf) >= 6 {
				suffix := string(buf[len(buf)-6:])
				if suffix == "]]>]]>" {
					return buf[:len(buf)-6], nil
				}
			}
		}
	}
}

func (f *Framer) writeBase10(msg []byte) error {
	_, err := f.writer.Write(msg)
	if err != nil {
		return err
	}
	_, err = f.writer.Write([]byte("]]>]]>"))
	return err
}

func (f *Framer) readBase11() ([]byte, error) {
	var total []byte

	for {
		// Read chunk header: #<size>\n
		line, err := f.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			return nil, fmt.Errorf("invalid chunk header: %s", line)
		}

		sizeStr := line[1:]
		size, err := strconv.Atoi(sizeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk size: %s", sizeStr)
		}

		if size == 0 {
			// End of message
			return total, nil
		}

		// Read chunk data
		chunk := make([]byte, size)
		_, err = io.ReadFull(f.reader, chunk)
		if err != nil {
			return nil, err
		}
		total = append(total, chunk...)
	}
}

func (f *Framer) writeBase11(msg []byte) error {
	// Split into chunks of reasonable size
	// For simplicity, send as a single chunk
	chunkSize := len(msg)
	if chunkSize > 0 {
		header := fmt.Sprintf("#%d\n", chunkSize)
		if _, err := f.writer.Write([]byte(header)); err != nil {
			return err
		}
		if _, err := f.writer.Write(msg); err != nil {
			return err
		}
	}
	// End with 0-length chunk
	_, err := f.writer.Write([]byte("#0\n"))
	return err
}
