package v1

// SyncState represents the synchronization state of a resource.
type SyncState string

const (
	SyncStateSuccess SyncState = "Success"
	SyncStateFailed  SyncState = "Failed"
	SyncStateSyncing SyncState = "Syncing"
	SyncStateTimeout SyncState = "Timeout"
)
