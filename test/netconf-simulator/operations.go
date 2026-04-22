package netsim

import (
	"encoding/xml"
	"fmt"
)

// GetConfigRequest represents a get-config request
type GetConfigRequest struct {
	XMLName  xml.Name `xml:"get-config"`
	Source   struct {
		Running   *struct{} `xml:"running"`
		Candidate *struct{} `xml:"candidate"`
	} `xml:"source"`
	Filter struct {
		Type    string `xml:"type,attr"`
		Content []byte `xml:",innerxml"`
	} `xml:"filter"`
}

// GetConfigResponse is the response to get-config
type GetConfigResponse struct {
	XMLName xml.Name `xml:"rpc-reply"`
	Config  []byte   `xml:"data>config,omitempty"`
}

// handleGetConfig handles a get-config RPC
func (h *RPCHandler) handleGetConfig(content []byte, msgID string, session *Session) error {
	var req GetConfigRequest
	if err := xml.Unmarshal(content, &req); err != nil {
		return fmt.Errorf("invalid get-config request: %w", err)
	}

	// Determine which datastore to use
	var config *Device
	if req.Source.Candidate != nil {
		config = h.server.GetDatastore().GetCandidate()
	} else {
		// default to running
		config = h.server.GetDatastore().GetRunning()
	}

	// Generate config XML
	// For now, just render the whole config
	// Filter processing will be done in later iteration if needed
	configXML, err := h.server.GetDatastore().RenderConfigXML(config, req.Filter.Content)
	if err != nil {
		return h.sendError(msgID, fmt.Sprintf("failed to render config: %v", err), session)
	}

	// Build response
	resp := fmt.Sprintf(`<rpc-reply message-id="%s"><data>%s</data></rpc-reply>`, msgID, configXML)
	return session.WriteMessage([]byte(resp))
}

// EditConfigRequest represents an edit-config request
type EditConfigRequest struct {
	XMLName     xml.Name `xml:"edit-config"`
	Target      struct {
		Candidate *struct{} `xml:"candidate"`
		Running   *struct{} `xml:"running"`
	} `xml:"target"`
	DefaultOperation string `xml:"default-operation"`
	Config           struct {
		Content []byte `xml:",innerxml"`
	} `xml:"config"`
}

// handleEditConfig handles edit-config request
func (h *RPCHandler) handleEditConfig(content []byte, msgID string, session *Session) error {
	var req EditConfigRequest
	if err := xml.Unmarshal(content, &req); err != nil {
		return h.sendError(msgID, fmt.Sprintf("invalid edit-config: %v", err), session)
	}

	// Get the target datastore
	var datastore *Datastore
	var candidate *Device
	datastore = h.server.GetDatastore()
	if req.Target.Running != nil {
		// Edit directly in running - copy to candidate first
		candidate = &Device{}
		if running := datastore.GetRunning(); running != nil && running.Vlans != nil {
			candidate.Vlans = running.Vlans
		}
		datastore.SetCandidate(candidate)
	} else {
		// default to candidate
		candidate = datastore.GetCandidate()
	}

	// Parse the config XML and merge into candidate
	if err := datastore.ParseConfigXML(req.Config.Content, candidate); err != nil {
		return h.sendError(msgID, err.Error(), session)
	}

	return h.sendOK(msgID, session)
}

// handleCommit handles commit request
func (h *RPCHandler) handleCommit(msgID string, session *Session) error {
	sc := h.server.GetScenario()
	if err, ok := sc.GetErrorForRPC("commit"); ok {
		return h.sendError(msgID, err.Error(), session)
	}

	datastore := h.server.GetDatastore()
	if err := datastore.Commit(); err != nil {
		return h.sendError(msgID, err.Error(), session)
	}

	return h.sendOK(msgID, session)
}
