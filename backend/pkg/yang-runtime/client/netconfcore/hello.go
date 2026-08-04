// hello.go — RFC 6241 会话建立：客户端 hello 构造、服务端 hello 解析、封帧协商。
//
// 关键规则（RFC 6242 §4.1）：hello 消息恒用 EOM 封帧；双方都宣告 base:1.1
// 才切换 chunked，否则回落 EOM（netconfsim 只报 1.0，真机华为 CE 通常 1.1）。
package netconfcore

import (
	"encoding/xml"
	"errors"
	"fmt"
)

const (
	capBase10 = "urn:ietf:params:netconf:base:1.0"
	capBase11 = "urn:ietf:params:netconf:base:1.1"
)

// FramingVersion 会话协商出的封帧格式。
type FramingVersion int

const (
	framingUnknown FramingVersion = iota
	// FramingEOM NETCONF 1.0，]]>]]> 定界。
	FramingEOM
	// FramingChunked NETCONF 1.1，chunked 封帧。
	FramingChunked
)

// ServerHello 服务端 hello 的解析结果。
type ServerHello struct {
	SessionID    uint64
	Capabilities []string
}

// BuildClientHello 构造客户端 hello（同时宣告 1.0/1.1，不带 session-id）。
func BuildClientHello() []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">` +
		`<capabilities>` +
		`<capability>` + capBase10 + `</capability>` +
		`<capability>` + capBase11 + `</capability>` +
		`</capabilities></hello>`)
}

// ParseServerHello 解析服务端 hello。encoding/xml 的元素匹配忽略 ns 前缀差异，
// 华为带 nc: 前缀与 netconfsim 无前缀两种风格都可解析。
func ParseServerHello(raw []byte) (*ServerHello, error) {
	var doc struct {
		XMLName      xml.Name `xml:"hello"`
		Capabilities []string `xml:"capabilities>capability"`
		SessionID    *uint64  `xml:"session-id"`
	}
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("netconfcore: 解析 server hello: %w", err)
	}
	if doc.SessionID == nil {
		return nil, errors.New("netconfcore: server hello 缺 session-id（RFC6241 必含）")
	}
	if len(doc.Capabilities) == 0 {
		return nil, errors.New("netconfcore: server hello 能力清单为空")
	}
	return &ServerHello{SessionID: *doc.SessionID, Capabilities: doc.Capabilities}, nil
}

// NegotiateFraming 依据服务端能力选封帧：1.1 优先，回落 1.0，都没有则报错。
func NegotiateFraming(serverCaps []string) (FramingVersion, error) {
	has10, has11 := false, false
	for _, c := range serverCaps {
		switch c {
		case capBase10:
			has10 = true
		case capBase11:
			has11 = true
		}
	}
	switch {
	case has11:
		return FramingChunked, nil
	case has10:
		return FramingEOM, nil
	default:
		return framingUnknown, errors.New("netconfcore: 服务端未宣告任何 base 能力，无法协商封帧")
	}
}
