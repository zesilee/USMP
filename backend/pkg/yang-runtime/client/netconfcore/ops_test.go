package netconfcore

import (
	"context"
	"strings"
	"testing"
)

// captureServer 记录最近一次请求并回 <ok/>，用于断言操作层生成的报文形态。
func newCaptureSession(t *testing.T) (*Session, *[]string) {
	t.Helper()
	var reqs []string
	s := newTestSession(t, caps10, func(req []byte) []byte {
		reqs = append(reqs, string(req))
		return echoOK(req)
	})
	return s, &reqs
}

func TestOpsEnvelopes(t *testing.T) {
	tests := []struct {
		name     string
		call     func(s *Session) error
		contains []string
		excludes []string
	}{
		{"GetConfig 带过滤器",
			func(s *Session) error {
				_, err := s.GetConfig(context.Background(), "running", "<vlan xmlns=\"urn:x\"/>")
				return err
			},
			[]string{"<get-config>", "<source><running/></source>",
				`<filter type="subtree"><vlan xmlns="urn:x"/></filter>`}, nil},
		{"GetConfig 无过滤器不得有空 filter",
			func(s *Session) error {
				_, err := s.GetConfig(context.Background(), "running", "")
				return err
			},
			[]string{"<get-config>"}, []string{"<filter"}},
		{"Get 状态读",
			func(s *Session) error {
				_, err := s.GetState(context.Background(), "<ifm xmlns=\"urn:y\"/>")
				return err
			},
			[]string{"<get>", `<filter type="subtree"><ifm xmlns="urn:y"/></filter>`}, nil},
		{"EditConfig",
			func(s *Session) error {
				_, err := s.EditConfig(context.Background(), "running", "<vlan><id>5</id></vlan>")
				return err
			},
			[]string{"<edit-config>", "<target><running/></target>",
				"<config>", "<vlan><id>5</id></vlan>"}, nil},
		{"Commit",
			func(s *Session) error { _, err := s.Commit(context.Background()); return err },
			[]string{"<commit/>"}, nil},
		{"CommitConfirmed 带超时",
			func(s *Session) error {
				_, err := s.CommitConfirmed(context.Background(), 60)
				return err
			},
			[]string{"<commit>", "<confirmed/>", "<confirm-timeout>60</confirm-timeout>"}, nil},
		{"DiscardChanges",
			func(s *Session) error { _, err := s.DiscardChanges(context.Background()); return err },
			[]string{"<discard-changes/>"}, nil},
		{"原样 RPC 透传",
			func(s *Session) error {
				_, err := s.Do(context.Background(), []byte(`<reset-counter xmlns="urn:h"/>`))
				return err
			},
			[]string{`<reset-counter xmlns="urn:h"/>`}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, reqs := newCaptureSession(t)
			if err := tt.call(s); err != nil {
				t.Fatalf("call: %v", err)
			}
			last := (*reqs)[len(*reqs)-1]
			// 信封统一约束：rpc 根元素 + base ns + message-id
			for _, want := range append(tt.contains,
				`<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="`) {
				if !strings.Contains(last, want) {
					t.Fatalf("报文缺 %q:\n%s", want, last)
				}
			}
			for _, bad := range tt.excludes {
				if strings.Contains(last, bad) {
					t.Fatalf("报文不应含 %q:\n%s", bad, last)
				}
			}
		})
	}
}

func TestOpsDatastoreValidation(t *testing.T) {
	s, _ := newCaptureSession(t)
	// datastore 进 XML 元素名，必须白名单拦注入
	if _, err := s.GetConfig(context.Background(), "running/><hack/", ""); err == nil {
		t.Fatal("非法 datastore 应拒绝")
	}
	if _, err := s.EditConfig(context.Background(), "", "<x/>"); err == nil {
		t.Fatal("空 datastore 应拒绝")
	}
}

func TestOpsConfirmTimeoutValidation(t *testing.T) {
	s, _ := newCaptureSession(t)
	if _, err := s.CommitConfirmed(context.Background(), 0); err == nil {
		t.Fatal("confirm-timeout ≤0 应拒绝")
	}
}
