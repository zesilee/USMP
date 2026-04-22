package netsim

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFramer_Base10(t *testing.T) {
	testMsg := []byte("<hello><capabilities><capability>urn:ietf:params:netconf:base:1.0</capability></capabilities></hello>")
	input := bytes.NewBuffer(append(testMsg, []byte("]]>]]>")...))
	output := bytes.NewBuffer(nil)

	rw := &readWriter{input, output}
	framer := NewFramer(rw, Base10)

	msg, err := framer.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, testMsg, msg)

	err = framer.WriteMessage(testMsg)
	assert.NoError(t, err)
	assert.Equal(t, append(testMsg, []byte("]]>]]>")...), output.Bytes())
}

func TestFramer_Base11(t *testing.T) {
	testMsg := []byte("<hello><capabilities><capability>urn:ietf:params:netconf:base:1.1</capability></capabilities></hello>")

	input := bytes.NewBuffer(nil)
	// Write as base11
	input.WriteString(fmt.Sprintf("#%d\n", len(testMsg)))
	input.Write(testMsg)
	input.WriteString("#0\n")

	output := bytes.NewBuffer(nil)

	rw := &readWriter{input, output}
	framer := NewFramer(rw, Base11)

	msg, err := framer.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, testMsg, msg)

	err = framer.WriteMessage(testMsg)
	assert.NoError(t, err)

	// Output should have one chunk and end with #0
	outputStr := output.String()
	assert.Contains(t, outputStr, "#0\n")
	assert.Contains(t, outputStr, string(testMsg))
}

func TestFramer_SetVersion(t *testing.T) {
	rw := &readWriter{bytes.NewBuffer(nil), bytes.NewBuffer(nil)}
	framer := NewFramer(rw, Base10)
	assert.Equal(t, Base10, framer.version)

	framer.SetVersion(Base11)
	assert.Equal(t, Base11, framer.version)
}

type readWriter struct {
	r *bytes.Buffer
	w *bytes.Buffer
}

func (rw *readWriter) Read(p []byte) (n int, err error) {
	return rw.r.Read(p)
}

func (rw *readWriter) Write(p []byte) (n int, err error) {
	return rw.w.Write(p)
}
