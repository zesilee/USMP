package netconfsim

import (
	"fmt"
	"strings"
)

// 按路径 unknown-element 注入（CN-04 地基）：复刻华为真机对 running 配置 schema
// 不存在节点的 313 形态拒绝。注入路径为局部名链（如 "devm/cards"）；请求的
// filter/config 体中出现完整链才拒绝——仅含父容器的请求不误伤（真机对只到父
// 容器的 filter 返回它有的部分，不报错）。

// matchUnknownElement reports whether the request body (filter inner or config
// content XML) contains any injected path chain. On a hit it returns the bad
// element (last segment) and the full matched chain.
func matchUnknownElement(paths []string, body string) (badElement string, chain []string, hit bool) {
	if len(paths) == 0 || strings.TrimSpace(body) == "" {
		return "", nil, false
	}
	root, err := parseXML([]byte(body))
	if err != nil {
		return "", nil, false // 畸形体不归注入管，交由正常链路报错
	}
	for _, p := range paths {
		segs := strings.Split(strings.Trim(p, "/"), "/")
		if len(segs) == 0 {
			continue
		}
		if chainPresent(root, segs) {
			return segs[len(segs)-1], segs, true
		}
	}
	return "", nil, false
}

// chainPresent walks the synthetic root looking for the local-name chain.
func chainPresent(n *dataNode, segs []string) bool {
	if len(segs) == 0 {
		return true
	}
	for _, c := range n.Children {
		if c.Name.Local == segs[0] && chainPresent(c, segs[1:]) {
			return true
		}
	}
	return false
}

// unknownElementReply builds the Huawei-shaped rejection (error-info-code 313).
func unknownElementReply(msgID, op string, chain []string) string {
	bad := chain[len(chain)-1]
	return fmt.Sprintf(`<rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">`+
		`<rpc-error>`+
		`<error-type>application</error-type>`+
		`<error-tag>unknown-element</error-tag>`+
		`<error-severity>error</error-severity>`+
		`<error-path xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0">/nc:rpc/nc:%s/%s</error-path>`+
		`<error-message xml:lang="en">Unexpected element: %s.</error-message>`+
		`<error-info xmlns:nc-ext="urn:huawei:yang:huawei-ietf-netconf-ext">`+
		`<bad-element>%s</bad-element>`+
		`<nc-ext:error-info-code>313</nc-ext:error-info-code>`+
		`</error-info>`+
		`</rpc-error>`+
		`</rpc-reply>`, msgID, op, strings.Join(chain, "/"), bad, bad)
}
