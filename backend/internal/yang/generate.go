//go:build generate
// +build generate

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openconfig/ygot/generator"
	"github.com/openconfig/ygot/ygot"
)

func main() {
	// Get the directory where YANG files are located
	yangDir := "models"
	outputFile := "generated/yang_models.go"
	packageName := "generated"

	// Find all .yang files
	yangFiles, err := filepath.Glob(filepath.Join(yangDir, "*.yang"))
	if err != nil {
		fmt.Printf("Failed to find YANG files: %v\n", err)
		os.Exit(1)
	}

	if len(yangFiles) == 0 {
		fmt.Println("No YANG files found in models directory")
		os.Exit(1)
	}

	// Create generator config
	config := &generator.GeneratorConfig{
		YANGFiles:            yangFiles,
		OutputFile:          outputFile,
		PackageName:         packageName,
		GenerateFakeRoot:    true,
		FakeRootName:        "Device",
		IncludeDescriptions: true,
		GenerateJSONSchema:  false,
	}

	// Generate code
	if err := generator.Generate(config); err != nil {
		fmt.Printf("Failed to generate code: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d YANG modules\n", outputFile, len(yangFiles))
}
