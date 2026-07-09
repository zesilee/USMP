package xmlcodec

import (
	"strings"
	"testing"
)

// TestCanonicalize 表格驱动覆盖规范化比较器的等价/不等价判定（D3）：
// 同级乱序等价、namespace 前缀不敏感、属性序不敏感、转义形式不敏感、
// 相同同级元素去重、值差异必须可分辨。
func TestCanonicalize(t *testing.T) {
	tests := []struct {
		name  string
		a, b  string
		equal bool
	}{
		{
			name:  "identical",
			a:     `<vlans xmlns="urn:x"><vlan><id>10</id><name>a</name></vlan></vlans>`,
			b:     `<vlans xmlns="urn:x"><vlan><id>10</id><name>a</name></vlan></vlans>`,
			equal: true,
		},
		{
			name:  "list entries reordered",
			a:     `<vlans><vlan><id>10</id></vlan><vlan><id>20</id></vlan></vlans>`,
			b:     `<vlans><vlan><id>20</id></vlan><vlan><id>10</id></vlan></vlans>`,
			equal: true,
		},
		{
			name:  "leaf siblings reordered",
			a:     `<vlan><id>10</id><name>a</name><description>d</description></vlan>`,
			b:     `<vlan><description>d</description><name>a</name><id>10</id></vlan>`,
			equal: true,
		},
		{
			name:  "namespace prefix vs default",
			a:     `<vlans xmlns="urn:x"><vlan><id>1</id></vlan></vlans>`,
			b:     `<h:vlans xmlns:h="urn:x"><h:vlan><h:id>1</h:id></h:vlan></h:vlans>`,
			equal: true,
		},
		{
			name:  "different namespace not equal",
			a:     `<vlans xmlns="urn:x"/>`,
			b:     `<vlans xmlns="urn:y"/>`,
			equal: false,
		},
		{
			name:  "attribute order insensitive",
			a:     `<vlan a="1" b="2"/>`,
			b:     `<vlan b="2" a="1"/>`,
			equal: true,
		},
		{
			name:  "prefixed attribute resolved",
			a:     `<vlan nc:operation="delete" xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"><id>1</id></vlan>`,
			b:     `<vlan x:operation="delete" xmlns:x="urn:ietf:params:xml:ns:netconf:base:1.0"><id>1</id></vlan>`,
			equal: true,
		},
		{
			name:  "escaped vs entity text",
			a:     `<name>a&lt;b&amp;c</name>`,
			b:     `<name>a&#60;b&#38;c</name>`,
			equal: true,
		},
		{
			name:  "duplicate identical siblings deduped",
			a:     `<vlan><suppression><inbound>1</inbound></suppression><suppression><inbound>1</inbound></suppression></vlan>`,
			b:     `<vlan><suppression><inbound>1</inbound></suppression></vlan>`,
			equal: true,
		},
		{
			name:  "duplicate differing siblings kept",
			a:     `<vlan><suppression><inbound>1</inbound></suppression><suppression><inbound>2</inbound></suppression></vlan>`,
			b:     `<vlan><suppression><inbound>1</inbound></suppression></vlan>`,
			equal: false,
		},
		{
			name:  "value difference detected",
			a:     `<vlan><id>10</id></vlan>`,
			b:     `<vlan><id>11</id></vlan>`,
			equal: false,
		},
		{
			name:  "missing element detected",
			a:     `<vlan><id>10</id><name>a</name></vlan>`,
			b:     `<vlan><id>10</id></vlan>`,
			equal: false,
		},
		{
			name:  "self-closing vs empty pair",
			a:     `<vlans xmlns="urn:x"/>`,
			b:     `<vlans xmlns="urn:x"></vlans>`,
			equal: true,
		},
		{
			name:  "inter-element whitespace ignored",
			a:     "<vlan>\n  <id>10</id>\n</vlan>",
			b:     `<vlan><id>10</id></vlan>`,
			equal: true,
		},
		{
			name:  "nested list reorder equal",
			a:     `<vlan><member-ports><member-port><interface-name>ge0</interface-name></member-port><member-port><interface-name>ge1</interface-name></member-port></member-ports></vlan>`,
			b:     `<vlan><member-ports><member-port><interface-name>ge1</interface-name></member-port><member-port><interface-name>ge0</interface-name></member-port></member-ports></vlan>`,
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca, err := Canonicalize([]byte(tt.a))
			if err != nil {
				t.Fatalf("Canonicalize(a): %v", err)
			}
			cb, err := Canonicalize([]byte(tt.b))
			if err != nil {
				t.Fatalf("Canonicalize(b): %v", err)
			}
			if (ca == cb) != tt.equal {
				t.Errorf("equal=%v, want %v\n a: %s\n b: %s", ca == cb, tt.equal, ca, cb)
			}
		})
	}
}

func TestCanonicalizeInvalidXML(t *testing.T) {
	if _, err := Canonicalize([]byte(`<vlans><vlan>`)); err == nil {
		t.Fatal("want error for unclosed XML, got nil")
	}
	if _, err := Canonicalize([]byte(``)); err == nil {
		t.Fatal("want error for empty input, got nil")
	}
}

// 并发调用无共享状态竞态（R09，-race 下验证）。
func TestCanonicalizeConcurrent(t *testing.T) {
	const doc = `<vlans><vlan><id>10</id></vlan><vlan><id>20</id></vlan></vlans>`
	done := make(chan string, 8)
	for i := 0; i < 8; i++ {
		go func() {
			c, err := Canonicalize([]byte(doc))
			if err != nil {
				c = "ERR:" + err.Error()
			}
			done <- c
		}()
	}
	first := <-done
	if strings.HasPrefix(first, "ERR:") {
		t.Fatal(first)
	}
	for i := 1; i < 8; i++ {
		if got := <-done; got != first {
			t.Errorf("nondeterministic canonical form:\n%s\nvs\n%s", got, first)
		}
	}
}
