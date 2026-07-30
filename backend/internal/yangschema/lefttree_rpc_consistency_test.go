package yangschema

import "testing"

// 左树 rpc 节点 ↔ ModuleRPCs 口径守护（LT-01/RPC-01）：凡可解析叶（RootContainers
// 非空，运行期可 available）在树上展出的 rpc 节点，其根容器必须在 ModuleRPCs 中
// 有同名 rpc 定义——否则前端点树节点会落到「rpc 不存在」错误页（rpcgen 模块清单
// 与 lefttreegen 全叶扫描漂移即在此红灯）。
func TestLeftTreeRPCNodesBackedByModuleRPCs(t *testing.T) {
	var walk func(nodes []LeftTreeNode)
	walk = func(nodes []LeftTreeNode) {
		for _, n := range nodes {
			if n.SourceModule != "" && len(n.RootContainers) > 0 {
				defs := map[string]bool{}
				for _, rc := range n.RootContainers {
					for _, d := range ModuleRPCs[rc] {
						defs[d.Name] = true
					}
				}
				for _, mn := range n.Nodes {
					if mn.Kind == "rpc" && !defs[mn.Name] {
						t.Errorf("叶 %s 的树 rpc 节点 %q 在 ModuleRPCs 无定义（rpcgen 模块清单缺该模块？）",
							n.SourceModule, mn.Name)
					}
				}
			}
			walk(n.Children)
		}
	}
	walk(LeftTree)
}
