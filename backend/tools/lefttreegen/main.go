// Command lefttreegen generates the Go left-tree structure from an SND package
// webui/template/left-tree.json (LT-01). Group hierarchy and zh-cn/en-us labels
// are preserved; each leaf (node with an xpath like "/huawei-vlan") additionally
// carries its module's top-level data-container names, resolved at build time
// with goyang — the same source-module→root-container mapping tasknamegen and
// blacklistgen use, because the runtime schema tree keys modules by root
// container and carries no namespace. Output is committed; the runtime image
// ships no snd files. Parse failures of individual modules leave the leaf's
// container set empty (the leaf renders as 未接入) and never abort generation
// (R08); a malformed left-tree.json aborts with an explicit error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"

	"github.com/leezesi/usmp/backend/tools/internal/rpcrisk"
)

// jsonNode mirrors one left-tree.json node.
type jsonNode struct {
	Type     string     `json:"type"`
	Zh       string     `json:"zh-cn"`
	En       string     `json:"en-us"`
	XPath    string     `json:"xpath"`
	Children []jsonNode `json:"children"`
}

type jsonRoot struct {
	LeftTree []jsonNode `json:"left-tree"`
}

// TreeNode is the generated tree node shape (mirrored in the output file).
type TreeNode struct {
	Zh             string
	En             string
	SourceModule   string
	RootContainers []string
	Nodes          []ModuleNode
	Children       []TreeNode
}

// ModuleNode is one module-level child of a leaf: a top-level data container or
// an rpc, flattened as siblings（模块顶层同级平铺，LT-01）. Labels are baked from
// the snd res files at build time (missing keys fall back to the raw name, R08).
type ModuleNode struct {
	Kind     string // "container" | "rpc"
	Name     string
	Zh       string
	En       string
	HighRisk bool // rpc only（rpcrisk 共享口径）
}

// buildTree parses left-tree.json and resolves each leaf's root containers
// and module-level nodes (containers + rpcs) from the YANG sources in yangDir;
// zh/en labels are baked from resDir/{zh-cn,en-us}/<module>-res.json.
func buildTree(treePath, yangDir, resDir string) ([]TreeNode, error) {
	raw, err := os.ReadFile(treePath)
	if err != nil {
		return nil, fmt.Errorf("read left-tree: %w", err)
	}
	var root jsonRoot
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse left-tree: %w", err)
	}
	if len(root.LeftTree) == 0 {
		return nil, fmt.Errorf("left-tree is empty: %s", treePath)
	}

	// 收集全部叶子模块，单次 goyang Process。
	modules := map[string]bool{}
	var collect func(ns []jsonNode)
	collect = func(ns []jsonNode) {
		for _, n := range ns {
			if m := leafModule(n); m != "" {
				modules[m] = true
			}
			collect(n.Children)
		}
	}
	collect(root.LeftTree)

	ms := yang.NewModules()
	ms.AddPath(yangDir)
	for m := range modules {
		if err := ms.Read(filepath.Join(yangDir, m+".yang")); err != nil {
			log.Printf("lefttreegen: read %s: %v (leaf will be unavailable)", m, err)
		}
	}
	if errs := ms.Process(); len(errs) > 0 {
		for _, e := range errs {
			log.Printf("lefttreegen: yang process warning: %v", e)
		}
	}
	roots := map[string][]string{}
	rpcs := map[string][]string{}
	for m := range modules {
		mod, ok := ms.Modules[m]
		if !ok || mod == nil {
			continue
		}
		e := yang.ToEntry(mod)
		if e == nil {
			continue
		}
		var cs, rs []string
		for name, child := range e.Dir {
			switch {
			case child == nil || len(child.Errors) > 0:
			case child.RPC != nil:
				rs = append(rs, name)
			case child.Dir != nil:
				cs = append(cs, name)
			}
		}
		sort.Strings(cs)
		sort.Strings(rs)
		roots[m] = cs
		rpcs[m] = rs
	}

	labels := newResLabels(resDir)
	var convert func(ns []jsonNode) []TreeNode
	convert = func(ns []jsonNode) []TreeNode {
		out := make([]TreeNode, 0, len(ns))
		for _, n := range ns {
			t := TreeNode{Zh: n.Zh, En: n.En}
			if m := leafModule(n); m != "" {
				t.SourceModule = m
				t.RootContainers = roots[m]
				t.Nodes = moduleNodes(m, roots[m], rpcs[m], labels)
			}
			t.Children = convert(n.Children)
			out = append(out, t)
		}
		return out
	}
	return convert(root.LeftTree), nil
}

// moduleNodes flattens a leaf module's top-level containers and rpcs as ordered
// siblings（container 前、rpc 后，各自字典序）with res-baked bilingual labels.
func moduleNodes(module string, roots, rpcNames []string, labels *resLabels) []ModuleNode {
	out := make([]ModuleNode, 0, len(roots)+len(rpcNames))
	for _, c := range roots {
		zh, en := labels.lookup(module, c)
		out = append(out, ModuleNode{Kind: "container", Name: c, Zh: zh, En: en})
	}
	for _, r := range rpcNames {
		zh, en := labels.lookup(module, r)
		out = append(out, ModuleNode{Kind: "rpc", Name: r, Zh: zh, En: en, HighRisk: rpcrisk.IsHighRisk(r)})
	}
	return out
}

// resLabels lazily loads per-module snd res files（键 `/<module>:<node>` → name）.
// Any missing file/key falls back to the raw node name — logged, never fatal
// (R08: 缺标签不阻断生成).
type resLabels struct {
	dir   string
	cache map[string]map[string]string // locale/module → path → name
}

func newResLabels(dir string) *resLabels {
	return &resLabels{dir: dir, cache: map[string]map[string]string{}}
}

func (r *resLabels) lookup(module, node string) (zh, en string) {
	key := "/" + module + ":" + node
	zh = r.name("zh-cn", module, key, node)
	en = r.name("en-us", module, key, node)
	return zh, en
}

func (r *resLabels) name(locale, module, key, fallback string) string {
	ck := locale + "/" + module
	m, ok := r.cache[ck]
	if !ok {
		m = map[string]string{}
		raw, err := os.ReadFile(filepath.Join(r.dir, locale, module+"-res.json"))
		if err != nil {
			log.Printf("lefttreegen: res %s/%s: %v (labels fall back to raw names)", locale, module, err)
		} else {
			var entries map[string]struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(raw, &entries); err != nil {
				log.Printf("lefttreegen: res %s/%s: %v (labels fall back to raw names)", locale, module, err)
			} else {
				for k, v := range entries {
					m[k] = v.Name
				}
			}
		}
		r.cache[ck] = m
	}
	if n := m[key]; n != "" {
		return n
	}
	return fallback
}

// leafModule returns the source module name of a leaf node ("" for groups).
func leafModule(n jsonNode) string {
	if n.XPath == "" {
		return ""
	}
	return strings.TrimPrefix(n.XPath, "/")
}

func main() {
	treePath := flag.String("tree", "", "path to left-tree.json")
	yangDir := flag.String("path", "", "YANG source directory")
	resDir := flag.String("res", "", "snd i18n resources directory (zh-cn/en-us res.json)")
	output := flag.String("output", "", "output .go file")
	pkg := flag.String("package", "yangschema", "output package name")
	flag.Parse()
	if *treePath == "" || *yangDir == "" || *resDir == "" || *output == "" {
		log.Fatal("lefttreegen: -tree, -path, -res and -output are required")
	}

	nodes, err := buildTree(*treePath, *yangDir, *resDir)
	if err != nil {
		log.Fatalf("lefttreegen: %v", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, `// Code generated by tools/lefttreegen. DO NOT EDIT.
//
// SND left-tree（LT-01）：分组层级 + 双语名 + 叶子源模块与顶层数据容器映射。
// 运行期零 snd 文件依赖；升级 snd 包后重跑 go:generate。

package %s

// LeftTreeNode 是左树节点：分组（Children 非空）或叶子（SourceModule 非空）。
// 叶子的 Nodes 是模块顶层 container 与 rpc 的平铺同级子节点（LT-01）。
type LeftTreeNode struct {
	Zh             string
	En             string
	SourceModule   string
	RootContainers []string
	Nodes          []LeftTreeModuleNode
	Children       []LeftTreeNode
}

// LeftTreeModuleNode 是叶子的模块级子节点：顶层数据容器或 rpc，双语标签
// 构建期自 snd res 烘焙（缺键回退原名）；HighRisk 与 rpcgen 同口径。
type LeftTreeModuleNode struct {
	Kind     string
	Name     string
	Zh       string
	En       string
	HighRisk bool
}

// LeftTree 是完整 SND 左树。
var LeftTree = %s
`, *pkg, renderNodes(nodes, 0))

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		log.Fatalf("lefttreegen: format: %v", err)
	}
	if err := os.WriteFile(*output, src, 0o644); err != nil {
		log.Fatalf("lefttreegen: write: %v", err)
	}
	log.Printf("lefttreegen: wrote %s (%d top groups, %d leaves)", *output, len(nodes), countLeaves(nodes))
}

func countLeaves(ns []TreeNode) int {
	c := 0
	for _, n := range ns {
		if n.SourceModule != "" {
			c++
		}
		c += countLeaves(n.Children)
	}
	return c
}

// renderNodes emits a deterministic Go literal for the node slice.
func renderNodes(ns []TreeNode, depth int) string {
	ind := strings.Repeat("\t", depth+1)
	var b strings.Builder
	b.WriteString("[]LeftTreeNode{\n")
	for _, n := range ns {
		fmt.Fprintf(&b, "%s{Zh: %q, En: %q", ind, n.Zh, n.En)
		if n.SourceModule != "" {
			fmt.Fprintf(&b, ", SourceModule: %q", n.SourceModule)
			if len(n.RootContainers) > 0 {
				fmt.Fprintf(&b, ", RootContainers: []string{")
				for i, c := range n.RootContainers {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q", c)
				}
				b.WriteString("}")
			}
			if len(n.Nodes) > 0 {
				b.WriteString(", Nodes: []LeftTreeModuleNode{\n")
				for _, mn := range n.Nodes {
					fmt.Fprintf(&b, "%s\t{Kind: %q, Name: %q, Zh: %q, En: %q", ind, mn.Kind, mn.Name, mn.Zh, mn.En)
					if mn.HighRisk {
						b.WriteString(", HighRisk: true")
					}
					b.WriteString("},\n")
				}
				b.WriteString(ind + "}")
			}
		}
		if len(n.Children) > 0 {
			fmt.Fprintf(&b, ", Children: %s", renderNodes(n.Children, depth+1))
		}
		b.WriteString("},\n")
	}
	b.WriteString(strings.Repeat("\t", depth) + "}")
	return b.String()
}
