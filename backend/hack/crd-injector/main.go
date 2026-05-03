// CRD Injector: Parses +custom:* annotations from Go source files
// and injects them as x-custom-* extensions into CRD OpenAPI schemas.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FieldAnnotation holds custom annotations for a struct field
type FieldAnnotation struct {
	StructName string
	FieldName  string
	Label      string
	Placeholder string
	Group      string
	Readonly   bool
	Hidden     bool
	Pattern    string
	Minimum    string
	Maximum    string
}

// CRDSpec represents the relevant parts of a CRD for modification
type CRD struct {
	Spec struct {
		Versions []struct {
			Name   string `yaml:"name"`
			Schema struct {
				OpenAPIV3Schema struct {
					Properties map[string]interface{} `yaml:"properties"`
				} `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

var (
	// Regex to parse +custom:* annotations
	customAnnotationRegex = regexp.MustCompile(`^//\s*\+custom:(\w+)\s*=\s*(.+)$`)
	fieldNameRegex        = regexp.MustCompile(`^(\w+)\s+\w+.*json:"([^,"]+).*$`)
)

// parseGoFile parses a Go source file and extracts custom annotations from struct fields
func parseGoFile(filePath string) ([]FieldAnnotation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var annotations []FieldAnnotation
	var currentStruct string
	var pendingCustoms map[string]string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for struct definition
		if strings.HasPrefix(line, "type ") && strings.Contains(line, " struct") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentStruct = parts[1]
			}
			continue
		}

		// Collect custom annotations
		if strings.HasPrefix(line, "// +custom:") {
			if pendingCustoms == nil {
				pendingCustoms = make(map[string]string)
			}
			matches := customAnnotationRegex.FindStringSubmatch(line)
			if len(matches) == 3 {
				pendingCustoms[matches[1]] = matches[2]
			}
			continue
		}

		// Check for field definition with json tag
		if currentStruct != "" && pendingCustoms != nil {
			matches := fieldNameRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				jsonName := matches[2]

				annot := FieldAnnotation{
					StructName: currentStruct,
					FieldName:  jsonName,
				}

				if v, ok := pendingCustoms["label"]; ok {
					annot.Label = strings.Trim(v, `"`)
				}
				if v, ok := pendingCustoms["placeholder"]; ok {
					annot.Placeholder = strings.Trim(v, `"`)
				}
				if v, ok := pendingCustoms["group"]; ok {
					annot.Group = strings.Trim(v, `"`)
				}
				if v, ok := pendingCustoms["readonly"]; ok {
					annot.Readonly = v == "true"
				}
				if v, ok := pendingCustoms["hidden"]; ok {
					annot.Hidden = v == "true"
				}

				annotations = append(annotations, annot)
				pendingCustoms = nil
			}
		}

		// Reset pending customs on empty line or closing brace
		if line == "" || line == "}" {
			pendingCustoms = nil
		}
	}

	return annotations, scanner.Err()
}

// injectToCRD injects custom annotations into a CRD YAML file
func injectToCRD(crdPath string, annotations []FieldAnnotation) error {
	data, err := os.ReadFile(crdPath)
	if err != nil {
		return err
	}

	var crd CRD
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return err
	}

	// Process each version
	for vi := range crd.Spec.Versions {
		specProps := getSchemaProperties(crd.Spec.Versions[vi].Schema.OpenAPIV3Schema.Properties, "spec")
		statusProps := getSchemaProperties(crd.Spec.Versions[vi].Schema.OpenAPIV3Schema.Properties, "status")

		for _, annot := range annotations {
			var props map[string]interface{}
			switch annot.StructName {
			case "BusinessVlanSpec", "BusinessInterfaceSpec", "BusinessRouteSpec", "BusinessSwitchSpec", "NativeDeviceConfigSpec":
				props = specProps
			case "BusinessVlanStatus", "BusinessInterfaceStatus", "BusinessRouteStatus", "BusinessSwitchStatus", "NativeDeviceConfigStatus":
				props = statusProps
			default:
				continue
			}

			if props == nil {
				continue
			}

			// Find the field schema
			fieldYaml, ok := props[annot.FieldName]
			if !ok {
				continue
			}

			// Convert to node to modify
			fieldNode, ok := fieldYaml.(map[string]interface{})
			if !ok {
				continue
			}

			// Add x-custom-* extensions
			if annot.Label != "" {
				fieldNode["x-custom-label"] = annot.Label
			}
			if annot.Placeholder != "" {
				fieldNode["x-custom-placeholder"] = annot.Placeholder
			}
			if annot.Group != "" {
				fieldNode["x-custom-group"] = annot.Group
			}
			if annot.Readonly {
				fieldNode["x-custom-readonly"] = true
			}
			if annot.Hidden {
				fieldNode["x-custom-hidden"] = true
			}
		}
	}

	// Write back
	output, err := yaml.Marshal(&crd)
	if err != nil {
		return err
	}

	return os.WriteFile(crdPath, output, 0644)
}

// getSchemaProperties extracts properties from a nested schema path
func getSchemaProperties(root map[string]interface{}, path string) map[string]interface{} {
	if root == nil {
		return nil
	}

	parts := strings.Split(path, ".")
	current := root

	for _, part := range parts {
		prop, ok := current[part]
		if !ok {
			return nil
		}

		propMap, ok := prop.(map[string]interface{})
		if !ok {
			// Check if it has properties
			withProps, ok := prop.(map[string]interface{})
			if !ok {
				return nil
			}
			if props, ok := withProps["properties"].(map[string]interface{}); ok {
				current = props
			} else {
				current = withProps
			}
		} else {
			if props, ok := propMap["properties"].(map[string]interface{}); ok {
				current = props
			} else {
				current = propMap
			}
		}
	}

	return current
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: crd-injector <go-source-dir> <crd-output-dir>")
		os.Exit(1)
	}

	sourceDir := os.Args[1]
	crdDir := os.Args[2]

	// Parse all Go source files for annotations
	var allAnnotations []FieldAnnotation
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), "_types.go") {
			annotations, err := parseGoFile(path)
			if err != nil {
				fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
				return nil
			}
			allAnnotations = append(allAnnotations, annotations...)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking source dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d custom annotations\n", len(allAnnotations))
	for _, a := range allAnnotations {
		fmt.Printf("  %s.%s: label=%q, group=%q\n", a.StructName, a.FieldName, a.Label, a.Group)
	}

	// Inject into each CRD
	CRDFiles := []string{
		filepath.Join(crdDir, "biz.usmp.io_businessvlans.yaml"),
		filepath.Join(crdDir, "biz.usmp.io_businessinterfaces.yaml"),
		filepath.Join(crdDir, "biz.usmp.io_businessroutes.yaml"),
		filepath.Join(crdDir, "biz.usmp.io_businessswitches.yaml"),
		filepath.Join(crdDir, "core.usmp.io_nativedeviceconfigs.yaml"),
	}

	for _, crdFile := range CRDFiles {
		if _, err := os.Stat(crdFile); os.IsNotExist(err) {
			fmt.Printf("Warning: CRD file not found: %s\n", crdFile)
			continue
		}

		if err := injectToCRD(crdFile, allAnnotations); err != nil {
			fmt.Printf("Error injecting to %s: %v\n", crdFile, err)
			continue
		}
		fmt.Printf("Injected annotations into %s\n", crdFile)
	}

	fmt.Println("Done!")
}
