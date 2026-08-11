// load.go — gen.conf 解析与 goyang 模型装载（codegen-conventions.md §7）。
// gen.conf 语义与 scripts/gen-yang.sh 完全一致；装载套路对齐仓库既有构建期
// 工具（rpcgen/tasknamegen）：AddPath + Read + Process + ToEntry。
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// GenConf mirrors the gen.conf manifest consumed by make gen-yang.
type GenConf struct {
	YangPaths  []string // 逗号拆分后的目录列表（相对仓库根）
	Modules    []string // 空格分隔的模块名
	FakeRoot   bool
	SplitCount int // 0 = 单文件模式
}

// ParseGenConf reads a gen.conf manifest（未知键报错——与 gen-yang.sh 同语义）。
func ParseGenConf(path string) (*GenConf, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("yanggen: open %s: %w", path, err)
	}
	defer f.Close()

	conf := &GenConf{FakeRoot: true}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("yanggen: %s 行格式非法: %q", path, line)
		}
		switch key {
		case "yang_path":
			for _, d := range strings.Split(val, ",") {
				if d = strings.TrimSpace(d); d != "" {
					conf.YangPaths = append(conf.YangPaths, d)
				}
			}
		case "modules":
			conf.Modules = strings.Fields(val)
		case "generate_fakeroot":
			conf.FakeRoot = val == "true"
		case "compress_paths":
			if val != "false" {
				return nil, fmt.Errorf("yanggen: compress_paths=true 不支持（约定冻结为非压缩路径）")
			}
		case "split_count":
			n, err := strconv.Atoi(val)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("yanggen: split_count 须为正整数: %q", val)
			}
			conf.SplitCount = n
		default:
			return nil, fmt.Errorf("yanggen: %s 含未知键: %s", path, key)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("yanggen: read %s: %w", path, err)
	}
	if len(conf.YangPaths) == 0 || len(conf.Modules) == 0 {
		return nil, fmt.Errorf("yanggen: %s 缺少 yang_path 或 modules", path)
	}
	return conf, nil
}

// LoadEntries parses the requested modules (searching every dir in paths,
// recursively) and returns module name → resolved root entry, plus the module
// set for namespace/prefix metadata. Deviation 模块列在 modules 中由 goyang
// 自动应用（CG-04）。任一请求模块缺失即报错（不静默缺模块）。
func LoadEntries(paths, modules []string) (map[string]*yang.Entry, map[string]*yang.Module, error) {
	ms := yang.NewModules()
	for _, p := range paths {
		if err := filepath.Walk(p, func(sub string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				ms.AddPath(sub)
			}
			return nil
		}); err != nil {
			return nil, nil, fmt.Errorf("yanggen: 模型目录不可用 %s: %w", p, err)
		}
	}
	for _, m := range modules {
		if err := readModule(ms, paths, m); err != nil {
			return nil, nil, err
		}
	}
	if errs := ms.Process(); len(errs) > 0 {
		// 与 -ignore_unsupported 精神一致：Process 的告警不 fatal（跨模块引用等
		// 已由 deviation 剪除的形态仍可能告警），但逐条落 stderr 供排查。
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "yanggen: yang process warning: %v\n", e)
		}
	}
	entries := make(map[string]*yang.Entry, len(modules))
	mods := make(map[string]*yang.Module, len(modules))
	for _, m := range modules {
		mod, ok := ms.Modules[m]
		if !ok || mod == nil {
			return nil, nil, fmt.Errorf("yanggen: 模块 %s 解析后缺失", m)
		}
		e := yang.ToEntry(mod)
		if e == nil {
			return nil, nil, fmt.Errorf("yanggen: 模块 %s 无法转换为 entry", m)
		}
		entries[m] = e
		mods[m] = mod
	}
	return entries, mods, nil
}

func readModule(ms *yang.Modules, paths []string, name string) error {
	for _, p := range paths {
		fn := filepath.Join(p, name+".yang")
		if _, err := os.Stat(fn); err == nil {
			return ms.Read(fn)
		}
	}
	// 交给 goyang 的搜索路径兜底（子目录中的模块）。
	if err := ms.Read(name + ".yang"); err != nil {
		return fmt.Errorf("yanggen: 模块 %s 在 %v 下未找到: %w", name, paths, err)
	}
	return nil
}

// sortedNames returns map keys sorted（确定性输出的通用小工具）。
func sortedNames[M ~map[string]V, V any](m M) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
