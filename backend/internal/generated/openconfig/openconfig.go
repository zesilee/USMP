//go:build generate
// +build generate

package openconfig

// Generate all OpenConfig modules together to avoid duplicate declarations
//go:generate go run github.com/openconfig/ygot/generator -path=../../yang/models -output_file=./all.gen.go -package_name=openconfig -generate_fakeroot=true openconfig-vlan openconfig-interfaces
//go:generate gofmt -w ./all.gen.go
