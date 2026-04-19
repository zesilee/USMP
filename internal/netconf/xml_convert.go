package netconf

import (
	"fmt"

	"github.com/openconfig/ygot/ygot"
)

// YangToJSON converts a ygot Go struct to JSON
func YangToJSON(yangObj ygot.GoStruct) (string, error) {
	output, err := ygot.EmitJSON(yangObj, &ygot.EmitJSONConfig{})
	if err != nil {
		return "", err
	}

	return output, nil
}

// XMLToYang decodes NETCONF XML to a ygot Go struct
func XMLToYang(xml string, yangObj interface{}) error {
	// TODO: Implement full XML→ygot struct decoding
	// This requires proper XML parsing into the generated struct
	return fmt.Errorf("XMLToYang not fully implemented yet")
}

// ConstructGetConfigFilter constructs a get-config filter XML for a specific YANG path
func ConstructGetConfigFilter(yangPath string) string {
	// Simple filter based on path
	// For "/interfaces" we create <filter><interfaces xmlns="..."/></filter>
	switch yangPath {
	case "/interfaces":
		return `<filter>
  <interfaces xmlns="http://openconfig.net/yang/interfaces"/>
</filter>`
	case "/vlans":
		return `<filter>
  <vlans xmlns="http://openconfig.net/yang/vlan"/>
</filter>`
	case "/system":
		return `<filter>
  <system xmlns="http://openconfig.net/yang/system"/>
</filter>`
	default:
		// Return empty filter to get entire configuration
		return `<filter/>`
	}
}

// ConstructEditConfig creates edit-config XML from a ygot object
func ConstructEditConfig(yangPath string, data interface{}) (string, error) {
	// For a given path, construct the proper <config> section
	// This will be properly implemented once we have generated structs
	var xml string
	switch yangPath {
	case "/interfaces":
		xml = `<target><running/></target>
<config>
  <interfaces xmlns="http://openconfig.net/yang/interfaces">
`
		// TODO: Insert the actual data converted from ygot
		xml += `  </interfaces>
</config>`
	default:
		xml = `<target><running/></target>
<config/>`
	}
	return xml, nil
}
