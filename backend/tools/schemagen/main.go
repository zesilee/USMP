package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	output := flag.String("output", "", "IR blob 输出路径（必填）")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "schemagen: -output is required")
		os.Exit(2)
	}
	if err := Run(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
