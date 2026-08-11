// yanggen — 自研 YANG→Go 生成器（retire-ygot-runtime 阶段2，CG-01 修订版）。
// 结构约定冻结自 ygot 生成物（openspec/changes/retire-ygot-runtime/
// codegen-conventions.md），生成物实现 pkg/yang-runtime/object 接口族、
// 零 ygot/goyang import。CLI 形状对齐 ygot generator，便于 gen-yang.sh 切换。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	path := flag.String("path", "", "YANG 模型目录（逗号分隔，递归搜索）")
	outputDir := flag.String("output_dir", "", "生成物输出目录")
	pkgName := flag.String("package_name", "", "生成包名")
	splitCount := flag.Int("structs_split_files_count", 0, "structs 拆分份数（0=单文件 all.gen.go）")
	flag.Parse()
	modules := flag.Args()

	if *path == "" || *outputDir == "" || *pkgName == "" || len(modules) == 0 {
		fmt.Fprintln(os.Stderr, "用法: yanggen -path=<dir,dir> -output_dir=<dir> -package_name=<pkg> [-structs_split_files_count=N] <module>...")
		os.Exit(2)
	}
	if err := run(strings.Split(*path, ","), *outputDir, *pkgName, *splitCount, modules); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(paths []string, outputDir, pkgName string, splitCount int, modules []string) error {
	entries, mods, err := LoadEntries(paths, modules)
	if err != nil {
		return err
	}
	model, err := BuildModel(pkgName, entries, mods)
	if err != nil {
		return err
	}
	files, err := EmitPackage(model, modules, splitCount)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("yanggen: mkdir %s: %w", outputDir, err)
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(outputDir, name), src, 0o644); err != nil {
			return fmt.Errorf("yanggen: write %s: %w", name, err)
		}
	}
	return nil
}
