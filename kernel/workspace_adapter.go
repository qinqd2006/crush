package kernel

import (
	"context"
)

// WorkspaceAdapter implements kernel.Workspace by wrapping an engine workspace.
// This allows the UI to depend only on kernel while using existing engine implementations.
type WorkspaceAdapter struct {
	name        string
	description string
	impl        WorkspaceImpl // The actual implementation (e.g., from engine)
}

// WorkspaceImpl is the interface for the underlying workspace implementation.
// This allows different implementations (engine, remote, mock) to be plugged in.
type WorkspaceImpl interface {
	// Sessions
	CreateSession(ctx context.Context, title string) (Session, error)
	GetSession(ctx context.Context, sessionID string) (Session, error)
	ListSessions(ctx context.Context) ([]Session, error)
	SaveSession(ctx context.Context, session Session) (Session, error)
	DeleteSession(ctx context.Context, sessionID string) error

	// Messages
	ListMessages(ctx context.Context, sessionID string) ([]Message, error)

	// Agent
	RunPrompt(ctx context.Context, sessionID, prompt string, attachments ...Attachment) error
	Cancel(sessionID string)
	CancelAll()
	IsBusy() bool
	IsSessionBusy(sessionID string) bool
	QueuedPrompts(sessionID string) int
	ClearQueue(sessionID string)
	Summarize(ctx context.Context, sessionID string) error

	// Permissions
	PermissionGrant(req PermissionRequest)
	PermissionGrantPersistent(req PermissionRequest)
	PermissionDeny(req PermissionRequest)
	PermissionSkipRequests() bool
	PermissionSetSkipRequests(skip bool)

	// MCP
	GetMCPStates() map[string]MCPClientInfo
	RefreshMCPTools(ctx context.Context, name string)
	RefreshMCPPrompts(ctx context.Context, name string)
	RefreshMCPResources(ctx context.Context, name string)
	GetMCPPrompt(clientID, promptID string, args map[string]string) (string, error)
	ReadMCPResource(ctx context.Context, clientID, uri string) ([]MCPResourceContents, error)
	EnableDockerMCP(ctx context.Context) error
	DisableDockerMCP() error

	// LSP
	LSPStart(ctx context.Context, path string)
	LSPStopAll(ctx context.Context)
	GetLSPStates() map[string]LSPClientInfo

	// File Tracking
	RecordFileRead(ctx context.Context, sessionID, path string)
	LastReadTime(ctx context.Context, sessionID, path string) int64
	ListReadFiles(ctx context.Context, sessionID string) ([]string, error)

	// Config
	Config() ConfigProvider
	WorkingDir() string
	Resolver() VariableResolver

	// Model
	GetModel() ModelConfig
	SetModel(ctx context.Context, model ModelConfig) error
	GetDefaultSmallModel(providerID string) ModelConfig

	// Project
	ProjectNeedsInitialization() (bool, error)
	MarkProjectInitialized() error
	InitializePrompt() (string, error)

	// Config mutations
	UpdatePreferredModel(scope Scope, modelType ModelType, model ModelConfig) error
	SetCompactMode(scope Scope, enabled bool) error
	SetProviderAPIKey(scope Scope, providerID string, apiKey any) error
	SetConfigField(scope Scope, key string, value any) error
	RemoveConfigField(scope Scope, key string) error

	// Lifecycle
	Shutdown()
}

// NewWorkspaceAdapter creates a new kernel.Workspace implementation.
func NewWorkspaceAdapter(name, description string, impl WorkspaceImpl) WorkspaceAdapter {
	return WorkspaceAdapter{
		name:        name,
		description: description,
		impl:        impl,
	}
}

// Compile-time check that WorkspaceAdapter implements Workspace.
var _ Workspace = (*WorkspaceAdapter)(nil)

// ------------------------------------------------------------------------
// Identity
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) Name() string {
	return w.name
}

func (w *WorkspaceAdapter) Description() string {
	return w.description
}

// ------------------------------------------------------------------------
// Agent Execution
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) RunPrompt(ctx context.Context, sessionID, prompt string, attachments ...Attachment) error {
	return w.impl.RunPrompt(ctx, sessionID, prompt, attachments...)
}

func (w *WorkspaceAdapter) Cancel(sessionID string) {
	w.impl.Cancel(sessionID)
}

func (w *WorkspaceAdapter) CancelAll() {
	w.impl.CancelAll()
}

func (w *WorkspaceAdapter) IsBusy() bool {
	return w.impl.IsBusy()
}

func (w *WorkspaceAdapter) IsSessionBusy(sessionID string) bool {
	return w.impl.IsSessionBusy(sessionID)
}

// AgentIsReady is an alias for IsBusy for backwards compatibility.
func (w *WorkspaceAdapter) AgentIsReady() bool {
	return w.impl.IsBusy()
}

// AgentIsSessionBusy is an alias for IsSessionBusy for backwards compatibility.
func (w *WorkspaceAdapter) AgentIsSessionBusy(sessionID string) bool {
	return w.impl.IsSessionBusy(sessionID)
}

func (w *WorkspaceAdapter) QueuedPrompts(sessionID string) int {
	return w.impl.QueuedPrompts(sessionID)
}

func (w *WorkspaceAdapter) ClearQueue(sessionID string) {
	w.impl.ClearQueue(sessionID)
}

func (w *WorkspaceAdapter) Summarize(ctx context.Context, sessionID string) error {
	return w.impl.Summarize(ctx, sessionID)
}

// ------------------------------------------------------------------------
// Messages
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) ListMessages(ctx context.Context, sessionID string) ([]Message, error) {
	return w.impl.ListMessages(ctx, sessionID)
}

func (w *WorkspaceAdapter) SubscribeMessages(ctx context.Context) <-chan Event[Message] {
	// TODO: Implement using pubsub
	ch := make(chan Event[Message], 100)
	return ch
}

// ------------------------------------------------------------------------
// Sessions
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) CreateSession(ctx context.Context, title string) (Session, error) {
	return w.impl.CreateSession(ctx, title)
}

func (w *WorkspaceAdapter) GetSession(ctx context.Context, sessionID string) (Session, error) {
	return w.impl.GetSession(ctx, sessionID)
}

func (w *WorkspaceAdapter) ListSessions(ctx context.Context) ([]Session, error) {
	return w.impl.ListSessions(ctx)
}

func (w *WorkspaceAdapter) SaveSession(ctx context.Context, session Session) (Session, error) {
	return w.impl.SaveSession(ctx, session)
}

func (w *WorkspaceAdapter) DeleteSession(ctx context.Context, sessionID string) error {
	return w.impl.DeleteSession(ctx, sessionID)
}

func (w *WorkspaceAdapter) SubscribeSessions(ctx context.Context) <-chan Event[Session] {
	// TODO: Implement using pubsub
	ch := make(chan Event[Session], 100)
	return ch
}

// ------------------------------------------------------------------------
// Permissions
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) PermissionGrant(req PermissionRequest) {
	w.impl.PermissionGrant(req)
}

func (w *WorkspaceAdapter) PermissionGrantPersistent(req PermissionRequest) {
	w.impl.PermissionGrantPersistent(req)
}

func (w *WorkspaceAdapter) PermissionDeny(req PermissionRequest) {
	w.impl.PermissionDeny(req)
}

func (w *WorkspaceAdapter) PermissionSkipRequests() bool {
	return w.impl.PermissionSkipRequests()
}

func (w *WorkspaceAdapter) PermissionSetSkipRequests(skip bool) {
	w.impl.PermissionSetSkipRequests(skip)
}

func (w *WorkspaceAdapter) SubscribePermissions(ctx context.Context) <-chan Event[PermissionRequest] {
	// TODO: Implement using pubsub
	ch := make(chan Event[PermissionRequest], 100)
	return ch
}

func (w *WorkspaceAdapter) SubscribePermissionNotifications(ctx context.Context) <-chan Event[PermissionNotification] {
	// TODO: Implement using pubsub
	ch := make(chan Event[PermissionNotification], 100)
	return ch
}

// ------------------------------------------------------------------------
// Notifications
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) SubscribeNotifications(ctx context.Context) <-chan Event[Notification] {
	// TODO: Implement using pubsub
	ch := make(chan Event[Notification], 100)
	return ch
}

// ------------------------------------------------------------------------
// MCP Management
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) GetMCPStates() map[string]MCPClientInfo {
	return w.impl.GetMCPStates()
}

func (w *WorkspaceAdapter) SubscribeMCP(ctx context.Context) <-chan Event[MCPEvent] {
	// TODO: Implement using pubsub
	ch := make(chan Event[MCPEvent], 100)
	return ch
}

func (w *WorkspaceAdapter) RefreshMCPTools(ctx context.Context, name string) {
	w.impl.RefreshMCPTools(ctx, name)
}

func (w *WorkspaceAdapter) RefreshMCPPrompts(ctx context.Context, name string) {
	w.impl.RefreshMCPPrompts(ctx, name)
}

func (w *WorkspaceAdapter) RefreshMCPResources(ctx context.Context, name string) {
	w.impl.RefreshMCPResources(ctx, name)
}

func (w *WorkspaceAdapter) GetMCPPrompt(clientID, promptID string, args map[string]string) (string, error) {
	return w.impl.GetMCPPrompt(clientID, promptID, args)
}

func (w *WorkspaceAdapter) ReadMCPResource(ctx context.Context, clientID, uri string) ([]MCPResourceContents, error) {
	return w.impl.ReadMCPResource(ctx, clientID, uri)
}

func (w *WorkspaceAdapter) EnableDockerMCP(ctx context.Context) error {
	return w.impl.EnableDockerMCP(ctx)
}

func (w *WorkspaceAdapter) DisableDockerMCP() error {
	return w.impl.DisableDockerMCP()
}

// ------------------------------------------------------------------------
// LSP Events
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) LSPStart(ctx context.Context, path string) {
	w.impl.LSPStart(ctx, path)
}

func (w *WorkspaceAdapter) LSPStopAll(ctx context.Context) {
	w.impl.LSPStopAll(ctx)
}

func (w *WorkspaceAdapter) GetLSPStates() map[string]LSPClientInfo {
	return w.impl.GetLSPStates()
}

func (w *WorkspaceAdapter) SubscribeLSP(ctx context.Context) <-chan Event[LSPEvent] {
	// TODO: Implement using pubsub
	ch := make(chan Event[LSPEvent], 100)
	return ch
}

// ------------------------------------------------------------------------
// File History Events
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) SubscribeHistory(ctx context.Context) <-chan Event[HistoryFile] {
	// TODO: Implement using pubsub
	ch := make(chan Event[HistoryFile], 100)
	return ch
}

func (w *WorkspaceAdapter) RecordFileRead(ctx context.Context, sessionID, path string) {
	w.impl.RecordFileRead(ctx, sessionID, path)
}

func (w *WorkspaceAdapter) LastReadTime(ctx context.Context, sessionID, path string) int64 {
	return w.impl.LastReadTime(ctx, sessionID, path)
}

func (w *WorkspaceAdapter) ListReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	return w.impl.ListReadFiles(ctx, sessionID)
}

// ------------------------------------------------------------------------
// History
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) ListSessionHistory(ctx context.Context, sessionID string) ([]HistoryFile, error) {
	// TODO: Implement
	return nil, nil
}

// ------------------------------------------------------------------------
// Session ID Helpers
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) CreateAgentToolSessionID(messageID, toolCallID string) string {
	// TODO: Implement
	return messageID + ":" + toolCallID
}

func (w *WorkspaceAdapter) ParseAgentToolSessionID(sessionID string) (messageID string, toolCallID string, ok bool) {
	// TODO: Implement
	return "", "", false
}

// ------------------------------------------------------------------------
// Model Selection
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) GetModel() ModelConfig {
	return w.impl.GetModel()
}

func (w *WorkspaceAdapter) SetModel(ctx context.Context, model ModelConfig) error {
	return w.impl.SetModel(ctx, model)
}

func (w *WorkspaceAdapter) GetDefaultSmallModel(providerID string) ModelConfig {
	return w.impl.GetDefaultSmallModel(providerID)
}

func (w *WorkspaceAdapter) InitAgent(ctx context.Context) error {
	// TODO: Implement
	return nil
}

// ------------------------------------------------------------------------
// Configuration
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) Config() ConfigProvider {
	return w.impl.Config()
}

func (w *WorkspaceAdapter) WorkingDir() string {
	return w.impl.WorkingDir()
}

func (w *WorkspaceAdapter) Resolver() VariableResolver {
	return w.impl.Resolver()
}

// ------------------------------------------------------------------------
// Config Mutations
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) UpdatePreferredModel(scope Scope, modelType ModelType, model ModelConfig) error {
	return w.impl.UpdatePreferredModel(scope, modelType, model)
}

func (w *WorkspaceAdapter) SetCompactMode(scope Scope, enabled bool) error {
	return w.impl.SetCompactMode(scope, enabled)
}

func (w *WorkspaceAdapter) SetProviderAPIKey(scope Scope, providerID string, apiKey any) error {
	return w.impl.SetProviderAPIKey(scope, providerID, apiKey)
}

func (w *WorkspaceAdapter) SetConfigField(scope Scope, key string, value any) error {
	return w.impl.SetConfigField(scope, key, value)
}

func (w *WorkspaceAdapter) RemoveConfigField(scope Scope, key string) error {
	return w.impl.RemoveConfigField(scope, key)
}

func (w *WorkspaceAdapter) ImportCopilot() (*OAuthToken, bool) {
	// TODO: Implement
	return nil, false
}

func (w *WorkspaceAdapter) RefreshOAuthToken(ctx context.Context, scope Scope, providerID string) error {
	// TODO: Implement
	return nil
}

// ------------------------------------------------------------------------
// Project Lifecycle
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) ProjectNeedsInitialization() (bool, error) {
	return w.impl.ProjectNeedsInitialization()
}

func (w *WorkspaceAdapter) MarkProjectInitialized() error {
	return w.impl.MarkProjectInitialized()
}

func (w *WorkspaceAdapter) InitializePrompt() (string, error) {
	return w.impl.InitializePrompt()
}

// ------------------------------------------------------------------------
// Events
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) Subscribe(program Program) {
	// TODO: Implement
}

func (w *WorkspaceAdapter) KernelSubscribe(program Program) {
	// TODO: Implement
}

// ------------------------------------------------------------------------
// Lifecycle
// ------------------------------------------------------------------------

func (w *WorkspaceAdapter) Shutdown() {
	w.impl.Shutdown()
}
