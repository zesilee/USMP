package netconf

// Constant XML templates for common NETCONF messages
const (
	// GetConfigTemplate is the base template for get-config
	GetConfigTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="%d" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source>
      <running/>
    </source>
    %s
  </get-config>
</rpc>`

	// EditConfigTemplate is the base template for edit-config
	EditConfigTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="%d" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target>
      <running/>
    </target>
    <config>
      %s
    </config>
  </edit-config>
</rpc>`

	// CommitTemplate is the commit RPC
	CommitTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="%d" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <commit/>
</rpc>`

	// LockTemplate locks the running configuration
	LockTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="%d" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <lock>
    <target>
      <running/>
    </target>
  </lock>
</rpc>`

	// UnlockTemplate unlocks the running configuration
	UnlockTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="%d" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <unlock>
    <target>
      <running/>
    </target>
  </unlock>
</rpc>`
)
