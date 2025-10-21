package models

// Host activity action types (T038: migrated from HostActivityLog)
// These constants define the actions that can be logged for host management
const (
	// Terminal actions
	ActivityTerminalCreated = "terminal_created"
	ActivityTerminalClosed  = "terminal_closed"
	ActivityTerminalReplay  = "terminal_replay"

	// Agent actions
	ActivityAgentConnected    = "agent_connected"
	ActivityAgentDisconnected = "agent_disconnected"
	ActivityAgentReconnected  = "agent_reconnected"

	// Node management actions
	ActivityNodeCreated = "node_created"
	ActivityNodeUpdated = "node_updated"
	ActivityNodeDeleted = "node_deleted"

	// System actions
	ActivitySystemAlert = "system_alert"
	ActivitySystemError = "system_error"
)

// Host activity type categories (T038: migrated from HostActivityLog)
// These constants categorize the type of activity
const (
	ActivityTypeTerminal = "terminal"
	ActivityTypeAgent    = "agent"
	ActivityTypeSystem   = "system"
	ActivityTypeUser     = "user"
)
