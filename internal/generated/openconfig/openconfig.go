//go:build generate
// +build generate

package openconfig

// Only generate VLAN module directly - all dependencies are already downloaded to /tmp/yang-vlan
// Use exclude_modules to exclude openconfig-interfaces which conflicts with ietf-interfaces on the root interfaces container
// openconfig-vlan only uses augment paths from openconfig-interfaces which still works without generating its top-level container
//go:generate go run github.com/openconfig/ygot/generator -path=/tmp/yang-vlan -output_file=./vlan.gen.go -package_name=openconfig -generate_fakeroot=true -exclude_modules=openconfig-interfaces openconfig-vlan openconfig-vlan-types
//go:generate gofmt -w ./vlan.gen.go
