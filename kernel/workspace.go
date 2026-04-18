package kernel

import (
	"context"
)

// ============================================================================
// Workspace Interface
// ============================================================================

// Workspace is the main interface consumed by the UI layer.
// It combines an Agent with shared Infrastructure to provide
// a complete environment for the UI to interact with.
//
// The UI only depends on this interface, allowing different
// agent implementations to be swapped in and out.
type Workspace interface {
	// ------------------------------------------------------------------------
	// Identity
	// ------------------------------------------------------------------------

	// Name returns the name of the current agent.
	Name() string

	// Description returns a description of the current agent.
	Description() string

	// ------------------------------------------------------------------------
	// Agent Execution
	// ------------------------------------------------------------------------

	// RunPrompt sends a prompt to the agent for processing.
	// Returns immediately; execution happens asynchronously.
	// Results are delivered via the message subscription.
	RunPrompt(ctx context.Context, sessionID, prompt string, attachments ...Attachment) error

	// Cancel stops the current agent execution for the session.
	Cancel(sessionID string)

	// CancelAll stops all agent executions.
	CancelAll()

	// IsBusy returns true if the agent is processing any request.
	IsBusy() bool

	// IsSessionBusy returns true if the agent is processing a request for this session.
	IsSessionBusy(sessionID string) bool

	// QueuedPrompts returns the number of queued prompts for a session.
	QueuedPrompts(sessionID string) int

	// ClearQueue clears the prompt queue for a session.
	ClearQueue(sessionID string)

	// Summarize generates a summary for the session.
	Summarize(ctx context.Context, sessionID string) error

	// ------------------------------------------------------------------------
	// Messages
	// ------------------------------------------------------------------------

	// ListMessages returns all messages for a session.
	ListMessages(ctx context.Context, sessionID string) ([]Message, error)

	// SubscribeMessages returns a channel for message events.
	SubscribeMessages(ctx context.Context) <-chan Event[Message]

	// ------------------------------------------------------------------------
	// Sessions
	// ------------------------------------------------------------------------

	// CreateSession creates a new chat session.
	CreateSession(ctx context.Context, title string) (Session, error)

	// GetSession returns a session by ID.
	GetSession(ctx context.Context, sessionID string) (Session, error)

	// ListSessions returns all sessions.
	ListSessions(ctx context.Context) ([]Session, error)

	// SaveSession saves a session (used for updates).
	SaveSession(ctx context.Context, session Session) (Session, error)

	// DeleteSession deletes a session.
	DeleteSession(ctx context.Context, sessionID string) error

	// SubscribeSessions returns a channel for session events.
	SubscribeSessions(ctx context.Context) <-chan Event[Session]

	// ------------------------------------------------------------------------
	// Permissions
	// ------------------------------------------------------------------------

	// PermissionGrant grants a permission request.
	PermissionGrant(req PermissionRequest)

	// PermissionGrantPersistent grants a permission persistently.
	PermissionGrantPersistent(req PermissionRequest)

	// PermissionDeny denies a permission request.
	PermissionDeny(req PermissionRequest)

	// PermissionSkipRequests returns whether permission requests are skipped.
	PermissionSkipRequests() bool

	// PermissionSetSkipRequests sets whether to skip permission requests.
	PermissionSetSkipRequests(skip bool)

	// SubscribePermissions returns a channel for permission requests.
	SubscribePermissions(ctx context.Context) <-chan Event[PermissionRequest]

	// SubscribePermissionNotifications returns a channel for permission results.
	SubscribePermissionNotifications(ctx context.Context) <-chan Event[PermissionNotification]

	// ------------------------------------------------------------------------
	// Notifications
	// ------------------------------------------------------------------------

	// SubscribeNotifications returns a channel for agent notifications.
	SubscribeNotifications(ctx context.Context) <-chan Event[Notification]

	// ------------------------------------------------------------------------
	// MCP Management
	// ------------------------------------------------------------------------

	// GetMCPStates returns the current state of all MCP clients.
	GetMCPStates() map[string]MCPClientInfo

	// SubscribeMCP returns a channel for MCP events.
	SubscribeMCP(ctx context.Context) <-chan Event[MCPEvent]

	// RefreshMCPTools refreshes the tools for an MCP client.
	RefreshMCPTools(ctx context.Context, name string)

	// RefreshMCPPrompts refreshes prompts for an MCP client.
	RefreshMCPPrompts(ctx context.Context, name string)

	// RefreshMCPResources refreshes resources for an MCP client.
	RefreshMCPResources(ctx context.Context, name string)

	// GetMCPPrompt gets an MCP prompt with arguments.
	GetMCPPrompt(clientID, promptID string, args map[string]string) (string, error)

	// ReadMCPResource reads an MCP resource.
	ReadMCPResource(ctx context.Context, clientID, uri string) ([]MCPResourceContents, error)

	// EnableDockerMCP enables Docker MCP.
	EnableDockerMCP(ctx context.Context) error

	// DisableDockerMCP disables Docker MCP.
	DisableDockerMCP() error

	// ------------------------------------------------------------------------
	// LSP Events
	// ------------------------------------------------------------------------

	// LSPStart starts an LSP server for the given path.
	LSPStart(ctx context.Context, path string)

	// LSPStopAll stops all LSP servers.
	LSPStopAll(ctx context.Context)

	// GetLSPStates returns the current state of all LSP clients.
	GetLSPStates() map[string]LSPClientInfo

	// SubscribeLSP returns a channel for LSP events.
	SubscribeLSP(ctx context.Context) <-chan Event[LSPEvent]

	// ------------------------------------------------------------------------
	// File History Events
	// ------------------------------------------------------------------------

	// SubscribeHistory returns a channel for file history events.
	SubscribeHistory(ctx context.Context) <-chan Event[HistoryFile]

	// RecordFileRead records that a file was read.
	RecordFileRead(ctx context.Context, sessionID, path string)

	// LastReadTime returns the last time a file was read.
	LastReadTime(ctx context.Context, sessionID, path string) int64

	// ListReadFiles returns all files read in a session.
	ListReadFiles(ctx context.Context, sessionID string) ([]string, error)

	// ------------------------------------------------------------------------
	// History
	// ------------------------------------------------------------------------

	// ListSessionHistory returns the file history for a session.
	ListSessionHistory(ctx context.Context, sessionID string) ([]HistoryFile, error)

	// ------------------------------------------------------------------------
	// Session ID Helpers (for agent tools that spawn sub-sessions)
	// ------------------------------------------------------------------------

	// CreateAgentToolSessionID creates a composite session ID for agent tool sessions.
	CreateAgentToolSessionID(messageID, toolCallID string) string

	// ParseAgentToolSessionID parses a composite session ID.
	ParseAgentToolSessionID(sessionID string) (messageID string, toolCallID string, ok bool)

	// ------------------------------------------------------------------------
	// Model Selection
	// ------------------------------------------------------------------------

	// GetModel returns the current model configuration.
	GetModel() ModelConfig

	// SetModel updates the model configuration.
	SetModel(ctx context.Context, model ModelConfig) error

	// GetDefaultSmallModel returns a suitable small/fast model for quick tasks.
	GetDefaultSmallModel(providerID string) ModelConfig

	// InitAgent initializes the agent (called once at startup).
	InitAgent(ctx context.Context) error

	// ------------------------------------------------------------------------
	// Configuration (read-only)
	// ------------------------------------------------------------------------

	// Config returns the current configuration.
	Config() ConfigProvider

	// WorkingDir returns the current working directory.
	WorkingDir() string

	// Resolver returns the variable resolver.
	Resolver() VariableResolver

	// ------------------------------------------------------------------------
	// Config Mutations (proxied in client mode)
	// ------------------------------------------------------------------------

	// UpdatePreferredModel updates the preferred model for a scope.
	UpdatePreferredModel(scope Scope, modelType ModelType, model ModelConfig) error

	// SetCompactMode sets compact mode.
	SetCompactMode(scope Scope, enabled bool) error

	// SetProviderAPIKey sets a provider API key.
	SetProviderAPIKey(scope Scope, providerID string, apiKey any) error

	// SetConfigField sets a configuration field.
	SetConfigField(scope Scope, key string, value any) error

	// RemoveConfigField removes a configuration field.
	RemoveConfigField(scope Scope, key string) error

	// ImportCopilot imports Copilot credentials.
	ImportCopilot() (*OAuthToken, bool)

	// RefreshOAuthToken refreshes an OAuth token.
	RefreshOAuthToken(ctx context.Context, scope Scope, providerID string) error

	// ------------------------------------------------------------------------
	// Project Lifecycle
	// ------------------------------------------------------------------------

	// ProjectNeedsInitialization returns whether the project needs initialization.
	ProjectNeedsInitialization() (bool, error)

	// MarkProjectInitialized marks the project as initialized.
	MarkProjectInitialized() error

	// InitializePrompt returns the initialization prompt.
	InitializePrompt() (string, error)

	// ------------------------------------------------------------------------
	// Events (Bubble Tea integration)
	// ------------------------------------------------------------------------

	// Subscribe subscribes to workspace events.
	Subscribe(program Program)

	// KernelSubscribe subscribes to kernel-level events.
	KernelSubscribe(program Program)

	// ------------------------------------------------------------------------
	// Lifecycle
	// ------------------------------------------------------------------------

	// Shutdown gracefully shuts down the workspace.
	Shutdown()
}

// ============================================================================
// Supporting Types
// ============================================================================

// Program represents the Bubble Tea program for event subscription.
// This is a minimal interface to avoid importing bubbletea into kernel.
type Program interface {
	Send(msg any)
}

// Scope represents a configuration scope.
type Scope string

const (
	ScopeGlobal   Scope = "global"
	ScopeProject  Scope = "project"
	ScopeSession  Scope = "session"
)

// ModelType represents the type of model.
type ModelType string

const (
	ModelTypeMain  ModelType = "main"
	ModelTypeSmall ModelType = "small"
)

// OAuthToken represents an OAuth token.
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	TokenType    string
}
