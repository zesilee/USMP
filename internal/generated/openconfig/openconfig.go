//go:build generate
// +build generate

package openconfig

// Only generate VLAN module directly - dependencies are imported via YANG import and should be found on search path
// Search paths:
//  - /tmp/yang-vlan = our direct modules (openconfig-vlan, openconfig-vlan-types)
//  - /Users/leezesi/Documents/code/usmp/yang-models/openconfig-public/release/models = all other openconfig modules
//  - /Users/leezesi/Documents/code/usmp/yang-models/openconfig-public/third_party/ietf = IETF modules
// This avoids duplicate root because ietf-interfaces is only in the third_party search path not in our temp dir
//go:generate go run github.com/openconfig/ygot/generator -path=/tmp/yang-vlan:/Users/leezesi/Documents/code/usmp/yang-models/openconfig-public/release/models:/Users/leezesi/Documents/code/usmp/yang-models/openconfig-public/third_party/ietf -output_file=./vlan.gen.go -package_name=openconfig -generate_fakeroot=true openconfig-vlan openconfig-vlan-types
//go:generate gofmt -w ./vlan.gen.go
