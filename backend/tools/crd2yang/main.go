package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	input := flag.String("input", "", "path to the northbound CRD manifest YAML")
	output := flag.String("output", "", "path to write the generated YANG module")
	flag.Parse()
	if *input == "" || *output == "" {
		log.Fatal("crd2yang: -input and -output are required")
	}

	manifest, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalf("crd2yang: read input: %v", err)
	}
	out, err := Generate(manifest)
	if err != nil {
		log.Fatalf("crd2yang: %v", err)
	}
	if err := os.WriteFile(*output, out, 0o644); err != nil {
		log.Fatalf("crd2yang: write output: %v", err)
	}
}
