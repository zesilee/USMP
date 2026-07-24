package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

func main() {
	output := flag.String("output", "", "directory to write <module>.json fixtures into")
	flag.Parse()
	if *output == "" {
		log.Fatal("schemadump: -output <dir> is required")
	}

	s, err := yangschema.Load()
	if err != nil {
		log.Fatalf("schemadump: load schema: %v", err)
	}

	fixtures, err := exportAll(s)
	if err != nil {
		log.Fatalf("schemadump: export: %v", err)
	}

	if err := os.MkdirAll(*output, 0o755); err != nil {
		log.Fatalf("schemadump: mkdir %s: %v", *output, err)
	}
	// 目录视作生成物：先清掉旧 *.json，让目录成为 schema 的纯函数
	// （上游删模块时不留陈旧 fixture，regen-and-diff 才诚实）。仅删 *.json，
	// 不触碰目录内其它文件（如 README）。
	stale, err := filepath.Glob(filepath.Join(*output, "*.json"))
	if err != nil {
		log.Fatalf("schemadump: scan stale fixtures: %v", err)
	}
	for _, p := range stale {
		if err := os.Remove(p); err != nil {
			log.Fatalf("schemadump: remove stale %s: %v", p, err)
		}
	}

	for _, name := range sortedNames(fixtures) {
		p := filepath.Join(*output, name+".json")
		if err := os.WriteFile(p, fixtures[name], 0o644); err != nil {
			log.Fatalf("schemadump: write %s: %v", p, err)
		}
	}

	log.Printf("schemadump: wrote %d fixtures to %s", len(fixtures), *output)
}
