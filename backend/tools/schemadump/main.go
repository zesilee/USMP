package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/leezesi/usmp/backend/internal/generated/businessdemo"
	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
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

	// 示例北向派生模型（crd-to-yang-codegen C2Y-05）：仅构建期并入 fixture 集，
	// 供前端派生黄金覆盖；不进运行期 Load()（D7，见 yangschema/businessdemo_test.go
	// 的隔离锚点）。schemadump 是独立短命进程，这里的扩注不外溢运行期。
	ds, ok := s.(*schema.DefaultSchema)
	if !ok {
		log.Fatalf("schemadump: unexpected schema impl %T (cannot append demo modules)", s)
	}
	bs, err := businessdemo.Schema()
	if err != nil {
		log.Fatalf("schemadump: load businessdemo schema: %v", err)
	}
	schema.AddYgotSchemaWithVendor(ds, bs, "usmp")

	n, err := run(s, *output)
	if err != nil {
		log.Fatalf("schemadump: %v", err)
	}
	log.Printf("schemadump: wrote %d fixtures to %s", n, *output)
}

// run exports every module fixture from s into outDir and returns the count
// written. The output directory is treated as a generated artifact: it is
// created if absent and existing *.json are cleared first, so the directory is
// a pure function of the schema (a module removed upstream leaves no stale
// fixture behind). Extracted from main so the write/clean pipeline is
// unit-testable without spawning a process.
func run(s schema.Schema, outDir string) (int, error) {
	fixtures, err := exportAll(s)
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	// 仅清 *.json，不触碰目录内其它文件（如 README）。
	stale, err := filepath.Glob(filepath.Join(outDir, "*.json"))
	if err != nil {
		return 0, fmt.Errorf("scan stale fixtures: %w", err)
	}
	for _, p := range stale {
		if err := os.Remove(p); err != nil {
			return 0, fmt.Errorf("remove stale %s: %w", p, err)
		}
	}

	for _, name := range sortedNames(fixtures) {
		p := filepath.Join(outDir, name+".json")
		if err := os.WriteFile(p, fixtures[name], 0o644); err != nil {
			return 0, fmt.Errorf("write %s: %w", p, err)
		}
	}
	return len(fixtures), nil
}
