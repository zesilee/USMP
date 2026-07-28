// Command rpcgen extracts YANG rpc definitions from an SND package's YANG sources
// (RPC-01) and generates a committed Go map (module → []RPCDef) for the runtime.
// The ygot-generated runtime schema is config-tree only and carries no rpc, so
// rpc metadata must be lifted at build time with goyang — the same source-parse
// approach lefttreegen/tasknamegen/blacklistgen use. Output is committed; the
// runtime image ships no snd files. Per-module parse failures are logged and skip
// that module (R08), never aborting generation.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// RPCInputLeaf is one input leaf of an rpc, in a shape the runtime maps to a
// front-end FieldDef (RPC-02).
type RPCInputLeaf struct {
	Name      string
	Type      string // string / number / boolean / enum / leafref (FieldDef type family)
	LeafRef   string // leafref target path (when Type == "leafref")
	Mandatory bool
	Units     string
	Pattern   string
}

// RPCDef is one rpc: its name, high-risk flag (RPC-04) and input leaves.
type RPCDef struct {
	Name     string
	HighRisk bool
	Input    []RPCInputLeaf
}

// highRiskWords flag rpcs with major, often irreversible device impact — restart/
// power/delete/rollback/upgrade/etc. Deliberately excludes the mild, common
// reset-/clear-counters family (those get the base confirm, not the escalated
// warning). Heuristic by design (D5); can later switch to a model annotation.
var highRiskWords = []string{
	"restart", "reboot", "reload", "warm", "cold", "power",
	"delete", "batch", "erase", "format", "rollback", "switch",
	"upgrade", "undo",
}

// isHighRisk reports whether an rpc name matches a high-risk verb (word-boundary
// on '-', so "reset" never matches "restart" and "clear" never matches).
func isHighRisk(name string) bool {
	segs := strings.Split(name, "-")
	for _, s := range segs {
		for _, w := range highRiskWords {
			if s == w {
				return true
			}
		}
	}
	return false
}

// buildRPCs parses the given modules under yangDir and returns, per module, its
// rpc definitions. Modules that fail to parse are skipped (logged), never fatal.
func buildRPCs(yangDir string, modules []string) (map[string][]RPCDef, error) {
	ms := yang.NewModules()
	ms.AddPath(yangDir)
	for _, m := range modules {
		if err := ms.Read(filepath.Join(yangDir, m+".yang")); err != nil {
			log.Printf("rpcgen: read %s: %v (module skipped)", m, err)
		}
	}
	if errs := ms.Process(); len(errs) > 0 {
		for _, e := range errs {
			log.Printf("rpcgen: yang process warning: %v", e)
		}
	}

	out := make(map[string][]RPCDef, len(modules))
	for _, m := range modules {
		mod, ok := ms.Modules[m]
		if !ok || mod == nil {
			continue
		}
		e := yang.ToEntry(mod)
		if e == nil {
			continue
		}
		var defs []RPCDef
		var roots []string
		for name, child := range e.Dir {
			switch {
			case child == nil || len(child.Errors) > 0:
				continue
			case child.RPC != nil:
				defs = append(defs, RPCDef{
					Name:     name,
					HighRisk: isHighRisk(name),
					Input:    inputLeaves(child.RPC.Input),
				})
			case child.Dir != nil:
				// config root container — the runtime schema keys modules by this name.
				roots = append(roots, name)
			}
		}
		if len(defs) == 0 {
			continue
		}
		// Deterministic order (e.Dir is a map) — regen-and-diff depends on it.
		sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })

		// Key by root container name (same key the runtime schema / schema API use),
		// so ModuleRPCs[<container>] serves rpcs at /yang/schema/<container>. rpcs are
		// module-level; a module with several roots exposes its rpcs under each. A
		// module with rpcs but no config container falls back to its module name
		// (not servable via container route, but never dropped).
		if len(roots) == 0 {
			out[m] = defs
			continue
		}
		for _, r := range roots {
			out[r] = defs
		}
	}
	return out, nil
}

// inputLeaves flattens an rpc's <input> container into ordered input leaves.
func inputLeaves(input *yang.Entry) []RPCInputLeaf {
	if input == nil || input.Dir == nil {
		return nil
	}
	leaves := make([]RPCInputLeaf, 0, len(input.Dir))
	for name, leaf := range input.Dir {
		if leaf == nil || leaf.Type == nil {
			continue
		}
		l := RPCInputLeaf{
			Name:      name,
			Type:      mapType(leaf.Type),
			Mandatory: leaf.Mandatory == yang.TSTrue,
			Units:     leaf.Units,
		}
		if leaf.Type.Kind == yang.Yleafref && leaf.Type.Path != "" {
			l.LeafRef = leaf.Type.Path
		}
		if len(leaf.Type.Pattern) > 0 {
			l.Pattern = leaf.Type.Pattern[0]
		}
		leaves = append(leaves, l)
	}
	sort.Slice(leaves, func(i, j int) bool { return leaves[i].Name < leaves[j].Name })
	return leaves
}

// mapType maps a goyang leaf type to the FieldDef type family the front-end renders.
func mapType(t *yang.YangType) string {
	switch t.Kind {
	case yang.Yleafref:
		return "leafref"
	case yang.Ybool:
		return "boolean"
	case yang.Yenum:
		return "enum"
	case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64,
		yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64, yang.Ydecimal64:
		return "number"
	default:
		return "string"
	}
}

func main() {
	yangDir := flag.String("path", "", "YANG source directory")
	modulesCSV := flag.String("modules", "", "comma-separated module names")
	output := flag.String("output", "", "output .go file")
	pkg := flag.String("package", "yangschema", "output package name")
	flag.Parse()
	if *yangDir == "" || *modulesCSV == "" || *output == "" {
		log.Fatal("rpcgen: -path, -modules and -output are required")
	}

	n, total, err := run(*yangDir, strings.Split(*modulesCSV, ","), *output, *pkg)
	if err != nil {
		log.Fatalf("rpcgen: %v", err)
	}
	log.Printf("rpcgen: wrote %s (%d modules, %d rpcs)", *output, n, total)
}

// run extracts, renders and writes the rpc definitions, returning the module and
// rpc counts. Extracted from main so the extract→render→write pipeline is
// unit-testable without spawning a process.
func run(yangDir string, modules []string, output, pkg string) (nModules, nRPCs int, err error) {
	rpcs, err := buildRPCs(yangDir, modules)
	if err != nil {
		return 0, 0, err
	}
	src, err := render(rpcs, pkg)
	if err != nil {
		return 0, 0, err
	}
	if err := os.WriteFile(output, src, 0o644); err != nil {
		return 0, 0, fmt.Errorf("write %s: %w", output, err)
	}
	for _, v := range rpcs {
		nRPCs += len(v)
	}
	return len(rpcs), nRPCs, nil
}

// render emits the deterministic ModuleRPCs literal.
func render(rpcs map[string][]RPCDef, pkg string) ([]byte, error) {
	mods := make([]string, 0, len(rpcs))
	for m := range rpcs {
		mods = append(mods, m)
	}
	sort.Strings(mods)

	var b strings.Builder
	fmt.Fprintf(&b, `// Code generated by tools/rpcgen. DO NOT EDIT.
//
// YANG rpc 定义（RPC-01）：模块 → rpc（名称/高危/输入叶）。ygot 运行期 schema
// 不含 rpc，构建期以 goyang 提取；运行期零 snd 文件依赖，升级 snd 后重跑 go:generate。

package %s

// RPCInputLeaf 是一个 rpc 输入叶（运行期映射为前端 FieldDef，RPC-02）。
type RPCInputLeaf struct {
	Name      string
	Type      string
	LeafRef   string
	Mandatory bool
	Units     string
	Pattern   string
}

// RPCDef 是一个 rpc：名称、高危标记（RPC-04）、输入叶。
type RPCDef struct {
	Name     string
	HighRisk bool
	Input    []RPCInputLeaf
}

// ModuleRPCs 是各模块的 rpc 定义（键为模块根容器名，与 schema 一致）。
var ModuleRPCs = map[string][]RPCDef{
`, pkg)
	for _, m := range mods {
		fmt.Fprintf(&b, "\t%q: {\n", m)
		for _, r := range rpcs[m] {
			fmt.Fprintf(&b, "\t\t{Name: %q, HighRisk: %t", r.Name, r.HighRisk)
			if len(r.Input) == 0 {
				b.WriteString("},\n")
				continue
			}
			b.WriteString(", Input: []RPCInputLeaf{\n")
			for _, in := range r.Input {
				fmt.Fprintf(&b, "\t\t\t{Name: %q, Type: %q", in.Name, in.Type)
				if in.LeafRef != "" {
					fmt.Fprintf(&b, ", LeafRef: %q", in.LeafRef)
				}
				if in.Mandatory {
					b.WriteString(", Mandatory: true")
				}
				if in.Units != "" {
					fmt.Fprintf(&b, ", Units: %q", in.Units)
				}
				if in.Pattern != "" {
					fmt.Fprintf(&b, ", Pattern: %q", in.Pattern)
				}
				b.WriteString("},\n")
			}
			b.WriteString("\t\t}},\n")
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n")

	return format.Source([]byte(b.String()))
}
