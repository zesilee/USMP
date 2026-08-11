package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	output := flag.String("output", "", "IR blob 输出路径（必填）")
	repoRoot := flag.String("repo_root", "..", "仓库根相对路径（gen.conf 与 yang_path 解析基准）")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "schemagen: -output is required")
		os.Exit(2)
	}
	if err := Run(*repoRoot, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
