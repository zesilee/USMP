package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	path := flag.String("path", "", "directory containing the .yang modules")
	module := flag.String("module", "", "YANG module name")
	group := flag.String("group", "biz.usmp.io", "API group")
	version := flag.String("version", "v1alpha1", "API version")
	kind := flag.String("kind", "", "CR kind, e.g. BusinessVlanService")
	plural := flag.String("plural", "", "lowercase plural resource name")
	output := flag.String("output", "", "output manifest path")
	flag.Parse()
	if *path == "" || *module == "" || *kind == "" || *plural == "" || *output == "" {
		log.Fatal("crdgen: -path, -module, -kind, -plural and -output are required")
	}

	out, err := Generate(Options{
		Path:    *path,
		Module:  *module,
		Group:   *group,
		Version: *version,
		Kind:    *kind,
		Plural:  *plural,
	})
	if err != nil {
		log.Fatalf("crdgen: %v", err)
	}
	if err := os.WriteFile(*output, out, 0o644); err != nil {
		log.Fatalf("crdgen: write %s: %v", *output, err)
	}
	log.Printf("crdgen: wrote %s", *output)
}
