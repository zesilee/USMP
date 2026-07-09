package client

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/scrapli/scrapligo/util"
)

// isTransportError 判定「会话已不可用、须重拨」——误判 false 会退回死连接
// 永久 EOF（本次生产事故），误判 true 会把业务级 rpc-error 当断连重拨。
func TestIsTransportError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"raw io.EOF", io.EOF, true},
		{"wrapped io.EOF", fmt.Errorf("get failed: %w", io.EOF), true},
		{"scrapligo timeout", util.ErrTimeoutError, true},
		{"wrapped scrapligo timeout", fmt.Errorf("op: %w", util.ErrTimeoutError), true},
		{"scrapligo connection error", util.ErrConnectionError, true},
		{"EOF in message text", errors.New("read EOF from channel"), true},
		{"connection reset", errors.New("read tcp: connection reset by peer"), true},
		{"broken pipe", errors.New("write tcp: broken pipe"), true},
		{"closed network conn", errors.New("use of closed network connection"), true},
		{"ssh session closed", errors.New("ssh: session closed"), true},
		{"rpc-error is not transport", errors.New("operation-failed: vlan 10 not found"), false},
		{"auth failure is not transport", errors.New("ssh: unable to authenticate"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTransportError(tt.err); got != tt.want {
				t.Errorf("isTransportError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
