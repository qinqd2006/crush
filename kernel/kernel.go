// Package kernel defines the core interface between the UI layer and the
// agent/coordinator engine. This decouples the UI from any specific
// agent implementation, allowing external packages to provide alternative
// engines.
package kernel

import (
	"context"
	"strings"
)

// InfoType represents the type of informational message.
type InfoType int

const (
	InfoTypeInfo InfoType = iota
	InfoTypeSuccess
	InfoTypeWarn
	InfoTypeError
	InfoTypeUpdate
)

// InfoMsg represents an informational message, used for warnings, errors, etc.
type InfoMsg struct {
	Type InfoType
	Msg  string
}

// NewInfoMsg creates an info message.
func NewInfoMsg(info string) InfoMsg {
	return InfoMsg{Type: InfoTypeInfo, Msg: info}
}

// NewWarnMsg creates a warning message.
func NewWarnMsg(warn string) InfoMsg {
	return InfoMsg{Type: InfoTypeWarn, Msg: warn}
}

// NewErrorMsg creates an error message.
func NewErrorMsg(err error) InfoMsg {
	return InfoMsg{Type: InfoTypeError, Msg: err.Error()}
}

// ============================================================================
// Core Types
// ============================================================================

// Message represents a chat message with parts for content, tools, and metadata.
type Message struct {
	ID               string
	Role             MessageRole
	SessionID        string
	Parts            []ContentPart
	Model            string
	Provider         string
	CreatedAt        int64
	UpdatedAt        int64
	IsSummaryMessage bool
}

// MessageRole identifies the sender of a message.
type MessageRole string

const (
	RoleAssistant MessageRole = "assistant"
	RoleUser      MessageRole = "user"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// RoleUser is an alias for convenience.
const User = RoleUser

// RoleAssistant is an alias for convenience.
const Assistant = RoleAssistant

// RoleSystem is an alias for convenience.
const System = RoleSystem

// RoleTool is an alias for convenience.
const Tool = RoleTool

// ContentPart represents a part of a message.
type ContentPart interface {
	isPart()
}

// TextContent represents plain text content.
type TextContent struct {
	Text string `json:"text"`
}

func (TextContent) isPart() {}

// ReasoningContent represents thinking/reasoning content.
type ReasoningContent struct {
	Thinking   string `json:"thinking"`
	Signature  string `json:"signature,omitempty"`
	StartedAt  int64  `json:"started_at,omitempty"`
	FinishedAt int64  `json:"finished_at,omitempty"`
}

func (ReasoningContent) isPart() {}

// ToolCallContent represents a tool invocation.
type ToolCallContent struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Input            string `json:"input"`
	ProviderExecuted bool   `json:"provider_executed"`
	Finished         bool   `json:"finished"`
}

func (ToolCallContent) isPart() {}

// ToolResultContent represents the result of a tool execution.
type ToolResultContent struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Data       []byte `json:"data,omitempty"`
	MIMEType   string `json:"mime_type,omitempty"`
	Metadata   string `json:"metadata,omitempty"`
	IsError    bool   `json:"is_error"`
}

func (ToolResultContent) isPart() {}

// ImageURLContent represents an image from a URL.
type ImageURLContent struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

func (ImageURLContent) isPart() {}

// BinaryContent represents binary data like images.
type BinaryContent struct {
	Path     string
	MIMEType string
	Data     []byte
}

func (BinaryContent) isPart() {}

// FinishContent represents the end of a message.
type FinishContent struct {
	Reason  FinishReason `json:"reason"`
	Time    int64        `json:"time,omitempty"`
	Message string       `json:"message,omitempty"`
	Details string       `json:"details,omitempty"`
}

func (FinishContent) isPart() {}

// FinishReason indicates why the assistant stopped generating.
type FinishReason string

const (
	FinishReasonEndTurn          FinishReason = "end_turn"
	FinishReasonMaxTokens        FinishReason = "max_tokens"
	FinishReasonToolUse          FinishReason = "tool_use"
	FinishReasonCanceled         FinishReason = "canceled"
	FinishReasonError            FinishReason = "error"
	FinishReasonPermissionDenied FinishReason = "permission_denied"
)

// Attachment represents a file attachment.
type Attachment struct {
	FilePath string
	FileName string
	MimeType string
	Content  []byte
}

// IsImage returns true if the attachment is an image.
func (a Attachment) IsImage() bool {
	return strings.HasPrefix(a.MimeType, "image/")
}

// IsText returns true if the attachment is a text file.
func (a Attachment) IsText() bool {
	return strings.HasPrefix(a.MimeType, "text/")
}

// Session represents a chat session.
type Session struct {
	ID               string
	ParentSessionID  string
	Title            string
	MessageCount     int64
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
	Todos            []Todo
	CreatedAt        int64
	UpdatedAt        int64
}

// Todo represents a task item.
type Todo struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"active_form"`
}

// ============================================================================
// Notification Types
// ============================================================================

// Notification represents a domain event from the agent.
type Notification struct {
	SessionID    string `json:"session_id"`
	SessionTitle string `json:"session_title,omitempty"`
	Type         NotificationType
	ProviderID   string `json:"provider_id,omitempty"`
}

// NotificationType identifies the kind of notification.
type NotificationType string

const (
	NotificationAgentFinished  NotificationType = "agent_finished"
	NotificationReAuthenticate NotificationType = "re_authenticate"
)

// PermissionRequest represents a request for user permission.
type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolCallID  string `json:"tool_call_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Path        string `json:"path,omitempty"`
}

// PermissionNotification represents a permission grant/deny event.
type PermissionNotification struct {
	ToolCallID string `json:"tool_call_id"`
	Granted    bool   `json:"granted"`
	Denied     bool   `json:"denied"`
}

// ============================================================================
// MCP Types
// ============================================================================

// MCPClientInfo holds information about an MCP client.
type MCPClientInfo struct {
	Name           string
	State          MCPState
	Error          error
	ToolCount      int
	PromptsCount   int
	ResourcesCount int
}

// MCPState represents the current state of an MCP client.
type MCPState int

const (
	MCPStateDisabled MCPState = iota
	MCPStateStarting
	MCPStateConnected
	MCPStateError
)

// MCPEvent represents an event in the MCP system.
type MCPEvent struct {
	Type           MCPEventType
	Name           string
	State          MCPState
	Error          error
	ToolCount      int
	PromptsCount   int
	ResourcesCount int
}

// MCPEventType represents the type of MCP event.
type MCPEventType string

const (
	MCPEventStateChanged         MCPEventType = "state_changed"
	MCPEventToolsListChanged     MCPEventType = "tools_list_changed"
	MCPEventPromptsListChanged   MCPEventType = "prompts_list_changed"
	MCPEventResourcesListChanged MCPEventType = "resources_list_changed"
)

// ============================================================================
// LSP Types
// ============================================================================

// LSPClientInfo holds information about an LSP client.
type LSPClientInfo struct {
	Name            string
	State           LSPState
	Error           error
	DiagnosticCount int
}

// LSPState represents the state of an LSP server.
type LSPState int

const (
	LSPStateUnstarted LSPState = iota
	LSPStateStarting
	LSPStateRunning
	LSPStateError
)

// LSPEvent represents an LSP event.
type LSPEvent struct {
	Type            LSPEventType
	Name            string
	DiagnosticCount int
	Error           error
}

// LSPEventType represents the type of LSP event.
type LSPEventType string

const (
	LSPEventStateChanged       LSPEventType = "state_changed"
	LSPEventDiagnosticsChanged LSPEventType = "diagnostics_changed"
)

// ============================================================================
// History Types
// ============================================================================

// HistoryFile represents a file in session history.
type HistoryFile struct {
	Path     string        `json:"path"`
	Versions []FileVersion `json:"versions,omitempty"`
}

// FileVersion represents a version of a file.
type FileVersion struct {
	Path      string `json:"path"`
	Hash      string `json:"hash"`
	Timestamp int64  `json:"timestamp"`
}

// ============================================================================
// Event Types
// ============================================================================

// EventType identifies the type of event.
type EventType string

const (
	EventCreated EventType = "created"
	EventUpdated EventType = "updated"
	EventDeleted EventType = "deleted"
)

// Event wraps a payload with an event type.
type Event[T any] struct {
	Type    EventType
	Payload T
}

// ============================================================================
// Message Helpers
// ============================================================================

// Content returns the text content of the message.
func (m *Message) Content() TextContent {
	for _, part := range m.Parts {
		if tc, ok := part.(TextContent); ok {
			return tc
		}
	}
	return TextContent{}
}

// ToolCalls returns all tool calls in the message.
func (m *Message) ToolCalls() []ToolCallContent {
	var calls []ToolCallContent
	for _, part := range m.Parts {
		if tc, ok := part.(ToolCallContent); ok {
			calls = append(calls, tc)
		}
	}
	return calls
}

// ToolResults returns all tool results in the message.
func (m *Message) ToolResults() []ToolResultContent {
	var results []ToolResultContent
	for _, part := range m.Parts {
		if tr, ok := part.(ToolResultContent); ok {
			results = append(results, tr)
		}
	}
	return results
}

// FinishPart returns the finish part if present.
func (m *Message) FinishPart() *FinishContent {
	for _, part := range m.Parts {
		if f, ok := part.(FinishContent); ok {
			return &f
		}
	}
	return nil
}

// FinishReason returns the finish reason.
func (m *Message) FinishReason() FinishReason {
	if f := m.FinishPart(); f != nil {
		return f.Reason
	}
	return ""
}

// IsThinking returns true if the message has thinking content.
func (m *Message) IsThinking() bool {
	for _, part := range m.Parts {
		if _, ok := part.(ReasoningContent); ok {
			return true
		}
	}
	return false
}

// ReasoningContent returns the reasoning content of the message.
func (m *Message) ReasoningContent() ReasoningContent {
	for _, part := range m.Parts {
		if rc, ok := part.(ReasoningContent); ok {
			return rc
		}
	}
	return ReasoningContent{}
}

// ImageURLContents returns all image URL contents in the message.
func (m *Message) ImageURLContents() []ImageURLContent {
	var contents []ImageURLContent
	for _, part := range m.Parts {
		if c, ok := part.(ImageURLContent); ok {
			contents = append(contents, c)
		}
	}
	return contents
}

// BinaryContents returns all binary contents in the message.
func (m *Message) BinaryContents() []BinaryContent {
	var contents []BinaryContent
	for _, part := range m.Parts {
		if c, ok := part.(BinaryContent); ok {
			contents = append(contents, c)
		}
	}
	return contents
}

// IsImage returns true if the message has image content.
func (m *Message) IsImage() bool {
	return false
}

// IsText returns true if the message has text content.
func (m *Message) IsText() bool {
	for _, part := range m.Parts {
		if _, ok := part.(TextContent); ok {
			return true
		}
	}
	return false
}

// ============================================================================
// Kernel Interface
// ============================================================================

// Kernel is the core interface between the UI and the agent/coordinator engine.
// It provides message operations, session management, agent control, and
// subscriptions to events.
type Kernel interface {
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

	// SubscribePermissions returns a channel for permission requests.
	SubscribePermissions(ctx context.Context) <-chan Event[PermissionRequest]

	// SubscribePermissionNotifications returns a channel for permission results.
	SubscribePermissionNotifications(ctx context.Context) <-chan Event[PermissionNotification]

	// ------------------------------------------------------------------------
	// Agent Notifications
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
	// Session ID Helpers (for agent tools that spawn sub-sessions)
	// ------------------------------------------------------------------------

	// CreateAgentToolSessionID creates a composite session ID for agent tool sessions.
	CreateAgentToolSessionID(messageID, toolCallID string) string

	// ParseAgentToolSessionID parses a composite session ID.
	ParseAgentToolSessionID(sessionID string) (messageID string, toolCallID string, ok bool)

	// ------------------------------------------------------------------------
	// Lifecycle
	// ------------------------------------------------------------------------

	// Shutdown gracefully shuts down the kernel.
	Shutdown()
}

// MCPResourceContents holds the contents of an MCP resource.
type MCPResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mime_type,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}
