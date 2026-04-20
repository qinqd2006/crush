package kernel

import (
	"context"
	"time"
)

// ============================================================================
// Infra Interface
// ============================================================================

// Infra defines the shared infrastructure that all agents use.
// This includes MCP servers, LSP servers, storage, permissions, and skills.
type Infra interface {
	// ------------------------------------------------------------------------
	// MCP
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
	// LSP
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
	// Storage
	// ------------------------------------------------------------------------

	// SessionStorage returns the session storage interface.
	SessionStorage() SessionStorage

	// MessageStorage returns the message storage interface.
	MessageStorage() MessageStorage

	// ------------------------------------------------------------------------
	// Permissions
	// ------------------------------------------------------------------------

	// PermissionChecker returns the permission checker interface.
	PermissionChecker() PermissionChecker

	// ------------------------------------------------------------------------
	// Skills
	// ------------------------------------------------------------------------

	// SkillsLoader returns the skills loader interface.
	SkillsLoader() SkillsLoader

	// ------------------------------------------------------------------------
	// Config
	// ------------------------------------------------------------------------

	// Config returns the current configuration.
	Config() ConfigProvider

	// WorkingDir returns the current working directory.
	WorkingDir() string

	// Resolver returns the variable resolver for config.
	Resolver() VariableResolver

	// ------------------------------------------------------------------------
	// File Tracking
	// ------------------------------------------------------------------------

	// RecordFileRead records that a file was read.
	RecordFileRead(ctx context.Context, sessionID, path string)

	// LastReadTime returns the last time a file was read.
	LastReadTime(ctx context.Context, sessionID, path string) time.Time

	// ListReadFiles returns all files read in a session.
	ListReadFiles(ctx context.Context, sessionID string) ([]string, error)

	// ------------------------------------------------------------------------
	// Lifecycle
	// ------------------------------------------------------------------------

	// Shutdown gracefully shuts down the infrastructure.
	Shutdown()
}

// ============================================================================
// Storage Interfaces
// ============================================================================

// SessionStorage defines the interface for session persistence.
type SessionStorage interface {
	// Create creates a new session.
	Create(ctx context.Context, session Session) (Session, error)

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (Session, error)

	// List returns all sessions.
	List(ctx context.Context) ([]Session, error)

	// Update updates a session.
	Update(ctx context.Context, session Session) (Session, error)

	// Delete deletes a session.
	Delete(ctx context.Context, id string) error
}

// MessageStorage defines the interface for message persistence.
type MessageStorage interface {
	// Create creates a new message.
	Create(ctx context.Context, message Message) (Message, error)

	// Get retrieves a message by ID.
	Get(ctx context.Context, id string) (Message, error)

	// List returns all messages for a session.
	List(ctx context.Context, sessionID string) ([]Message, error)

	// ListUserMessages returns only user messages for a session.
	ListUserMessages(ctx context.Context, sessionID string) ([]Message, error)

	// ListAllUserMessages returns all user messages across all sessions.
	ListAllUserMessages(ctx context.Context) ([]Message, error)

	// Update updates a message.
	Update(ctx context.Context, message Message) (Message, error)

	// Delete deletes a message.
	Delete(ctx context.Context, id string) error

	// DeleteSession deletes all messages for a session.
	DeleteSession(ctx context.Context, sessionID string) error
}

// ============================================================================
// Permission Interface
// ============================================================================

// PermissionChecker defines the interface for permission checking.
type PermissionChecker interface {
	// Request asks for permission to perform an action.
	// Returns a channel that will receive the permission result.
	Request(ctx context.Context, req PermissionRequest) <-chan PermissionResult

	// Check checks if an action is allowed without prompting.
	Check(ctx context.Context, toolName, path string) bool

	// Grant grants a permission request.
	Grant(req PermissionRequest)

	// GrantPersistent grants a permission request persistently.
	GrantPersistent(req PermissionRequest)

	// Deny denies a permission request.
	Deny(req PermissionRequest)

	// SkipRequests sets whether to skip permission requests.
	SkipRequests(skip bool)

	// Subscribe returns a channel for permission requests.
	Subscribe(ctx context.Context) <-chan Event[PermissionRequest]

	// SubscribeNotifications returns a channel for permission results.
	SubscribeNotifications(ctx context.Context) <-chan Event[PermissionNotification]
}

// PermissionResult represents the result of a permission request.
type PermissionResult struct {
	Granted    bool
	Denied     bool
	SkipRemain bool
}

// ============================================================================
// Skills Interface
// ============================================================================

// SkillsLoader defines the interface for loading agent skills.
type SkillsLoader interface {
	// Load loads skills from the configured directories.
	Load(ctx context.Context) error

	// List returns all available skills.
	List() []Skill

	// Get returns a skill by name.
	Get(name string) *Skill

	// Subscribe returns a channel for skill events.
	Subscribe(ctx context.Context) <-chan Event[SkillEvent]
}

// Skill represents an agent skill.
type Skill struct {
	Name        string
	Description string
	Path        string
	Commands    []SkillCommand
}

// SkillCommand represents a command exposed by a skill.
type SkillCommand struct {
	Name        string
	Description string
	Args        []string
}

// SkillEvent represents an event from the skills system.
type SkillEvent struct {
	Type  SkillEventType
	Skill Skill
}

// SkillEventType represents the type of skill event.
type SkillEventType string

const (
	SkillEventLoaded   SkillEventType = "loaded"
	SkillEventUnloaded SkillEventType = "unloaded"
)

// ============================================================================
// Config Interface
// ============================================================================

// ConfigProvider defines the interface for configuration access.
type ConfigProvider interface {
	// Get returns the full configuration.
	Get() *Config

	// IsConfigured returns true if at least one provider is configured.
	IsConfigured() bool

	// GetPreferredModel returns the preferred model for a scope.
	GetPreferredModel(scope string) ModelConfig

	// SetPreferredModel sets the preferred model for a scope.
	SetPreferredModel(scope string, model ModelConfig) error

	// GetModelInfo returns basic model information.
	GetModelInfo(providerID, modelID string) ModelInfo

	// GetProviderInfo returns basic provider information.
	GetProviderInfo(providerID string) ProviderInfo

	// ------------------------------------------------------------------------
	// Dialog/UI Support
	// ------------------------------------------------------------------------

	// Providers returns all configured providers for display.
	Providers() ([]Provider, error)

	// GetAgent returns the agent configuration by name.
	GetAgent(name string) (AgentConfig, bool)

	// GetSelectedModel returns the selected model configuration by type.
	GetSelectedModel(modelType SelectedModelType) (SelectedModel, bool)

	// GetModel returns the model configuration by provider and model ID.
	GetModel(providerID, modelID string) (Model, bool)

	// ------------------------------------------------------------------------
	// Config Mutations
	// ------------------------------------------------------------------------

	// SetConfigField sets a configuration field at the given scope.
	SetConfigField(scope Scope, key string, value any) error

	// ------------------------------------------------------------------------
	// UI View Helpers (returns UI-friendly view types)
	// ------------------------------------------------------------------------

	// GetCoderAgent returns the coder agent with its current model capabilities.
	GetCoderAgent() AgentView

	// GetSmallModel returns the small/fast model configuration if set.
	GetSmallModel() (ModelView, bool)

	// GetModelViewByType returns the model view by selected model type.
	GetModelViewByType(modelType SelectedModelType) (ModelView, bool)

	// GetGlobalOptions returns global application options.
	GetGlobalOptions() GlobalOptions

	// TUIOptions returns TUI-specific display preferences.
	TUIOptions() TUIOptions

	// GetProvider returns a provider by ID.
	GetProvider(providerID string) (Provider, bool)

	// ------------------------------------------------------------------------
	// Engine-Specific Metadata (P3 - projected for UI)
	// ------------------------------------------------------------------------

	// ListMCPServers returns a list of MCP server metadata for UI display.
	ListMCPServers() []MCPServerInfo

	// ListCustomCommands returns a list of custom command metadata for UI display.
	ListCustomCommands() []CustomCommandInfo
}

// ============================================================================
// Config Types
// ============================================================================

// SelectedModelType represents the type of selected model.
type SelectedModelType string

const (
	SelectedModelTypeLarge SelectedModelType = "large"
	SelectedModelTypeSmall SelectedModelType = "small"
)

// AgentName represents the name of an agent.
type AgentName string

const (
	AgentCoder AgentName = "coder"
)

// Model represents an AI model configuration (kernel-native version of catwalk.Model).
type Model struct {
	ID                     string
	Name                   string
	CostPer1MIn            float64
	CostPer1MOut           float64
	CostPer1MInCached      float64
	CostPer1MOutCached     float64
	ContextWindow          int64
	DefaultMaxTokens       int64
	CanReason              bool
	ReasoningLevels        []string
	DefaultReasoningEffort string
	SupportsImages         bool
}

// Provider represents an AI provider configuration.
type Provider struct {
	ID          string
	Name        string
	Type        string // "openai", "anthropic", "gemini", etc.
	APIEndpoint string
	Models      []Model
}

// SelectedModel represents a selected model configuration.
type SelectedModel struct {
	Model            string
	Provider         string
	ReasoningEffort  string
	Think            bool
	MaxTokens        int64
	Temperature      *float64
	TopP             *float64
	TopK             *int64
	FrequencyPenalty *float64
	PresencePenalty  *float64
	ProviderOptions  map[string]any
}

// AgentModel represents the current agent model with UI-specific details.
// It combines the model configuration with the model metadata.
type AgentModel struct {
	Model    Model         // Model metadata (name, context window, reasoning support, etc.)
	ModelCfg SelectedModel // Selected model configuration (provider, reasoning effort, etc.)
}

// AgentConfig represents an agent configuration.
type AgentConfig struct {
	ID           string
	Name         string
	Description  string
	Model        SelectedModelType
	AllowedTools []string
	AllowedMCP   map[string][]string
	ContextPaths []string
}

// Config holds the application configuration.
type Config struct {
	// Selected model configurations (large/small)
	Models map[SelectedModelType]SelectedModel

	// Recently used models per type
	RecentModels map[SelectedModelType][]SelectedModel

	// Agent configurations
	Agents map[string]AgentConfig

	// Provider configurations
	Providers map[string]Provider
}

// ModelInfo holds basic model information.
type ModelInfo struct {
	Name string
}

// ProviderInfo holds basic provider information.
type ProviderInfo struct {
	Name string
}

// ModelView is a UI-oriented model capability view.
// It provides only the information the UI needs for rendering.
type ModelView struct {
	Type          SelectedModelType // "main", "small"
	Name          string            // Display name like "Claude 3.5 Sonnet"
	ProviderID    string            // Provider ID like "anthropic"
	ModelID       string            // Model ID like "claude-3-5-sonnet-20241022"
	ContextWindow int64             // Context window size for percentage calculation
	SupportsImage bool              // Whether images can be uploaded
}

// AgentView is a UI-oriented agent capability view.
type AgentView struct {
	Role         string    // "coder"
	Name         string    // Display name
	CurrentModel ModelView // The model this agent is using
}

// GlobalOptions holds global application options that UI may need to read.
type GlobalOptions struct {
	DisableNotifications bool
	InitializeAs         string
	Progress             bool
}

// CompletionsLimits holds limits for the completions UI.
type CompletionsLimits struct {
	MaxDepth int
	MaxItems int
}

// TUIOptions holds TUI-specific display preferences.
type TUIOptions struct {
	CompactMode bool
	DiffMode    string
	Completions CompletionsLimits
	Transparent bool
}

// ConfigView adds UI-friendly methods to ConfigProvider.
// These methods return view types that hide internal config complexity.
type ConfigView interface {
	// GetCoderAgent returns the coder agent with its current model capabilities.
	GetCoderAgent() AgentView

	// GetSmallModel returns the small/fast model configuration if set.
	GetSmallModel() (ModelView, bool)

	// GetModelByType returns the model configuration by selected model type.
	GetModelByType(modelType SelectedModelType) (ModelView, bool)
}

// VariableResolver defines the interface for resolving config variables.
type VariableResolver interface {
	// Resolve resolves a variable to its value.
	Resolve(key string) (string, bool)
}

// MCPServerInfo holds minimal MCP server metadata for UI display.
type MCPServerInfo struct {
	Name string // Server name (e.g., "filesystem", "github")
	Type string // Connection type: "stdio", "sse", "http"
}

// CustomCommandInfo holds minimal custom command metadata for UI display.
type CustomCommandInfo struct {
	Name        string // Command name
	Description string // Brief description
}
