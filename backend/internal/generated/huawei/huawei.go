//go:build generate
// +build generate

package huawei

// Generate all Huawei YANG modules together
// NOTE: ygot does not support per-module file splitting
// To regenerate: cd internal/generated/huawei && go generate -tags=generate
//go:generate go run github.com/openconfig/ygot/generator -path=../../../../yang-models/network-router/8.20.10/ne40e-x8x16 -output_file=./all.gen.go -package_name=huawei -generate_fakeroot=true -compress_paths=false huawei-vlan huawei-ifm huawei-system huawei-pub-type huawei-extension
//go:generate sh -c "sed -i '' 's/HuaweiIfm_PortType_50|100GE/HuaweiIfm_PortType_50_OR_100GE/g' ./all.gen.go"
//go:generate sh -c "sed -i '' 's/HuaweiIfm_PortType_FlexE_50|100G/HuaweiIfm_PortType_FlexE_50_OR_100G/g' ./all.gen.go"
//go:generate gofmt -w ./all.gen.go

// ============================================================
// FOR NEW MODULES (example for future use):
// To generate a standalone file for a new YANG module:
// 1. Add a new //go:generate line below with the new module name
// 2. Set -output_file=./module_name.gen.go
// 3. Note: huawei-pub-type and huawei-extension are common dependencies
// ============================================================
// EXAMPLE (uncomment and modify for new modules):
//go:generate go run github.com/openconfig/ygot/generator -path=../../../../yang-modules/network-router/8.20.10/ne40e-x8x16 -output_file=./aaa.gen.go -package_name=huawei -generate_fakeroot=false -compress_paths=false huawei-aaa huawei-pub-type huawei-extension
//go:generate go run github.com/openconfig/ygot/generator -path=../../../../yang-modules/network-router/8.20.10/ne40e-x8x16 -output_file=./ntp.gen.go -package_name=huawei -generate_fakeroot=false -compress_paths=false huawei-ntp huawei-pub-type huawei-extension
