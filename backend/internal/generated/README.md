# YANG Generated Code

This directory contains Go structs generated from Huawei YANG modules using ygot.

## Structure

```
huawei/
├── doc.go          - Package documentation and usage guide
├── huawei.go       - go:generate configuration for ygot
└── all.gen.go      - Combined generated code for all YANG modules
```

## Generated Modules

Currently generated:
1. **huawei-vlan**       - VLAN configuration (5 structs)
2. **huawei-ifm**        - Interface Management (29 structs)
3. **huawei-system**     - System configuration (10 structs)
4. **huawei-pub-type**   - Common public types
5. **huawei-extension**  - Extension types

## Adding a New YANG Module

### Option 1: Add to combined generation (Recommended)

Update `huawei/huawei.go` and add the new module name to the existing `go:generate` line, then regenerate:

```bash
cd internal/generated/huawei
go generate -tags=generate
```

### Option 2: Standalone file generation (For future large modules)

For larger new modules, you can generate a standalone `.gen.go` file:

1. Add a new `//go:generate` line in `huawei.go`:
```go
//go:generate go run github.com/openconfig/ygot/generator -path=../../../../yang-models/... -output_file=./aaa.gen.go -package_name=huawei -generate_fakeroot=false -compress_paths=false huawei-aaa huawei-pub-type huawei-extension
```

2. Include common dependencies: `huawei-pub-type` and `huawei-extension`
3. Important: Use `-generate_fakeroot=false` to avoid duplicate Device struct

## Important Notes

1. **Same package constraint**: All generated files MUST be in the same `huawei` package due to cross-module type references.

2. **Device root struct**: There is only one `Device` root struct per package. Subsequent modules with `-generate_fakeroot=true` will cause duplicate definition errors.

3. **Enum naming issue**: ygot uses `|` in enum names which is invalid Go syntax. The sed commands in `huawei.go` fix this.

4. **Do not edit generated files**: `.gen.go` files are auto-generated. Make changes to the YANG models or generation configuration.

## Regeneration

```bash
cd internal/generated/huawei
go generate -tags=generate
```

## Background on Why Single File?

ygot (YANG Go Tools) generates all code in a single file by design because:
- Cross-module type references require everything in one compilation unit
- The `Device` root struct needs to reference all top-level containers
- Enum mappings are global across all modules
- The gzipped schema is a combined representation of all modules

This approach is standard practice in the ygot ecosystem (see: openconfig/public).
