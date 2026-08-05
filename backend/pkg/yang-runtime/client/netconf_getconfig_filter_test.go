package client

import (
	"strings"
	"testing"
)

// 真机回归（T07）：get-config 过滤器曾是自造 XPath 形态
// `<filter xmlns="…netconf:base:1.0" select="/ifm:ifm/ifm:interfaces"/>`——
// RFC6241 无 type 缺省按 subtree 解释，空元素=什么都不选，真机（CE 8.20.10）
// 正确回 <data/>，界面接口/VLAN 列表全空、对账以空实际态永久漂移。模拟网元把
// 「filter 存在但为空」当「无 filter」返回全量，全套测试因此从未变红。
// 修后 get-config 与状态读同源：constructSubtreeFilter 生成带命名空间的嵌套
// subtree 体，外裹 <filter type="subtree">。
func TestConstructFilterIsSubtree(t *testing.T) {
	c := &NETCONFClient{}
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "ifm 接口列表",
			path: "/ifm:ifm/ifm:interfaces",
			want: `<filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces/></ifm></filter>`,
		},
		{
			name: "vlan 列表",
			path: "/vlan:vlan/vlan:vlans",
			want: `<filter type="subtree"><vlan xmlns="urn:huawei:yang:huawei-vlan"><vlans/></vlan></filter>`,
		},
		{
			name: "带 list 谓词剥除（整列表读）",
			path: "/ifm:ifm/ifm:interfaces/ifm:interface[name='GE0/0/1']",
			want: `<filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces><interface/></interfaces></ifm></filter>`,
		},
		{
			name: "空路径=全量读不带 filter",
			path: "",
			want: "",
		},
		// 真机回归（wire 抓包实证，2026-08-05）：路径带前导双斜杠时，驱动注册表
		// 前缀匹配（HasPrefix "/ifm:ifm"）落空 → namespace 静默丢失 →
		// <ifm> 无 xmlns → 严格设备 subtree 匹配不到，秒回空 <data/>。
		// 路径规范化后任意数量前导斜杠都必须解析出 namespace。
		{
			name: "前导双斜杠仍须带 namespace",
			path: "//ifm:ifm/ifm:interfaces",
			want: `<filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces/></ifm></filter>`,
		},
		{
			name: "仅斜杠=全量读不带 filter",
			path: "///",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.constructFilter(tt.path)
			if got != tt.want {
				t.Errorf("constructFilter(%q)\n got:  %s\n want: %s", tt.path, got, tt.want)
			}
			if strings.Contains(got, "select=") {
				t.Errorf("禁止 XPath select 形态（真机按 subtree 解释为空过滤器）: %s", got)
			}
		})
	}
}
