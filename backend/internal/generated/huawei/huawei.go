//go:build generate
// +build generate

package huawei

// Generate Huawei VLAN module
//go:generate go run github.com/openconfig/ygot/generator -path=../../../../yang-models/network-router/8.20.10/ne40e-x8x16 -output_file=./all.gen.go -package_name=huawei -generate_fakeroot=true huawei-vlan huawei-pub-type huawei-extension
//go:generate sed -i '' 's/HuaweiIfm_PortType_50|100GE/HuaweiIfm_PortType_50_OR_100GE/g' ./all.gen.go
//go:generate sed -i '' 's/HuaweiIfm_PortType_FlexE_50|100G/HuaweiIfm_PortType_FlexE_50_OR_100G/g' ./all.gen.go
//go:generate gofmt -w ./all.gen.go
