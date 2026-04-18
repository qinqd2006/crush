// Package adapter provides implementations of kernel interfaces that wrap
// internal engine packages. This allows external packages (like UI) to
// depend only on kernel interfaces while using the existing engine implementations.
package adapter

import (
	"context"

	"github.com/mosaic2025002/crush/engine/config"
	"github.com/mosaic2025002/crush/engine/message"
	"github.com/mosaic2025002/crush/engine/permission"
	"github.com/mosaic2025002/crush/engine/session"
	"github.com/mosaic2025002/crush/engine/workspace"
	"github.com/mosaic2025002/crush/kernel"
)

// WorkspaceAdapter wraps an engine workspace.Workspace and implements
// kernel.Workspace by converting between engine and kernel types.
type WorkspaceAdapter struct {
	impl workspace.Workspace
}

// NewWorkspaceAdapter creates a new WorkspaceAdapter wrapping an engine workspace.
func NewWorkspaceAdapter(impl workspace.Workspace) *WorkspaceAdapter {
	return &WorkspaceAdapter{impl: impl}
}

// Compile-time check that WorkspaceAdapter implements kernel.Workspace.
var _ kernel.Workspace = (*WorkspaceAdapter)(nil)

// ------------------------------------------------------------------------
// Identity
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) Name() string {
	return "Crush"
}

func (a *WorkspaceAdapter) Description() string {
	return "AI coding assistant powered by LLMs"
}

// ------------------------------------------------------------------------
// Agent Execution
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) RunPrompt(ctx context.Context, sessionID, prompt string, attachments ...kernel.Attachment) error {
	engineAttachments := make([]message.Attachment, len(attachments))
	for i, att := range attachments {
		engineAttachments[i] = message.Attachment{
			FilePath: att.FilePath,
			FileName: att.FileName,
			MimeType: att.MimeType,
			Content:  att.Content,
		}
	}
	return a.impl.AgentRun(ctx, sessionID, prompt, engineAttachments...)
}

func (a *WorkspaceAdapter) Cancel(sessionID string) {
	a.impl.AgentCancel(sessionID)
}

func (a *WorkspaceAdapter) CancelAll() {
	// Engine workspace doesn't have AgentCancelAll, cancel current session
}

func (a *WorkspaceAdapter) IsBusy() bool {
	return a.impl.AgentIsBusy()
}

func (a *WorkspaceAdapter) IsSessionBusy(sessionID string) bool {
	return a.impl.AgentIsSessionBusy(sessionID)
}

func (a *WorkspaceAdapter) QueuedPrompts(sessionID string) int {
	return a.impl.AgentQueuedPrompts(sessionID)
}

func (a *WorkspaceAdapter) ClearQueue(sessionID string) {
	a.impl.AgentClearQueue(sessionID)
}

func (a *WorkspaceAdapter) Summarize(ctx context.Context, sessionID string) error {
	return a.impl.AgentSummarize(ctx, sessionID)
}

// ------------------------------------------------------------------------
// Messages
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) ListMessages(ctx context.Context, sessionID string) ([]kernel.Message, error) {
	msgs, err := a.impl.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return a.messagesToKernel(msgs), nil
}

func (a *WorkspaceAdapter) SubscribeMessages(ctx context.Context) <-chan kernel.Event[kernel.Message] {
	ch := make(chan kernel.Event[kernel.Message], 100)
	return ch
}

func (a *WorkspaceAdapter) messagesToKernel(msgs []message.Message) []kernel.Message {
	result := make([]kernel.Message, len(msgs))
	for i, m := range msgs {
		result[i] = a.messageToKernel(&m)
	}
	return result
}

func (a *WorkspaceAdapter) messageToKernel(m *message.Message) kernel.Message {
	parts := make([]kernel.ContentPart, 0, len(m.Parts))
	for _, p := range m.Parts {
		switch tp := p.(type) {
		case message.TextContent:
			parts = append(parts, kernel.TextContent{Text: tp.Text})
		case message.ReasoningContent:
			parts = append(parts, kernel.ReasoningContent{
				Thinking:   tp.Thinking,
				Signature:  tp.Signature,
				StartedAt:  tp.StartedAt,
				FinishedAt: tp.FinishedAt,
			})
		case message.ToolCall:
			parts = append(parts, kernel.ToolCallContent{
				ID:               tp.ID,
				Name:             tp.Name,
				Input:            tp.Input,
				ProviderExecuted: tp.ProviderExecuted,
				Finished:         tp.Finished,
			})
		case message.ToolResult:
			parts = append(parts, kernel.ToolResultContent{
				ToolCallID: tp.ToolCallID,
				Name:       tp.Name,
				Content:    tp.Content,
				Data:       []byte(tp.Data),
				MIMEType:   tp.MIMEType,
				Metadata:   tp.Metadata,
				IsError:    tp.IsError,
			})
		case message.Finish:
			parts = append(parts, kernel.FinishContent{
				Reason:  kernel.FinishReason(tp.Reason),
				Time:    tp.Time,
				Message: tp.Message,
				Details: tp.Details,
			})
		case message.ImageURLContent:
			parts = append(parts, kernel.ImageURLContent{
				URL:    tp.URL,
				Detail: tp.Detail,
			})
		case message.BinaryContent:
			parts = append(parts, kernel.BinaryContent{
				Path:     tp.Path,
				MIMEType: tp.MIMEType,
				Data:     tp.Data,
			})
		}
	}
	return kernel.Message{
		ID:               m.ID,
		Role:             kernel.MessageRole(m.Role),
		SessionID:        m.SessionID,
		Parts:            parts,
		Model:            m.Model,
		Provider:         m.Provider,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		IsSummaryMessage: m.IsSummaryMessage,
	}
}

// ------------------------------------------------------------------------
// Sessions
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) CreateSession(ctx context.Context, title string) (kernel.Session, error) {
	sess, err := a.impl.CreateSession(ctx, title)
	if err != nil {
		return kernel.Session{}, err
	}
	return a.sessionToKernel(&sess), nil
}

func (a *WorkspaceAdapter) GetSession(ctx context.Context, sessionID string) (kernel.Session, error) {
	sess, err := a.impl.GetSession(ctx, sessionID)
	if err != nil {
		return kernel.Session{}, err
	}
	return a.sessionToKernel(&sess), nil
}

func (a *WorkspaceAdapter) ListSessions(ctx context.Context) ([]kernel.Session, error) {
	sessions, err := a.impl.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]kernel.Session, len(sessions))
	for i, s := range sessions {
		result[i] = a.sessionToKernel(&s)
	}
	return result, nil
}

func (a *WorkspaceAdapter) SaveSession(ctx context.Context, sess kernel.Session) (kernel.Session, error) {
	s := a.kernelToSession(&sess)
	saved, err := a.impl.SaveSession(ctx, s)
	if err != nil {
		return kernel.Session{}, err
	}
	return a.sessionToKernel(&saved), nil
}

func (a *WorkspaceAdapter) DeleteSession(ctx context.Context, sessionID string) error {
	return a.impl.DeleteSession(ctx, sessionID)
}

func (a *WorkspaceAdapter) SubscribeSessions(ctx context.Context) <-chan kernel.Event[kernel.Session] {
	ch := make(chan kernel.Event[kernel.Session], 100)
	return ch
}

func (a *WorkspaceAdapter) sessionToKernel(s *session.Session) kernel.Session {
	todos := make([]kernel.Todo, len(s.Todos))
	for i, t := range s.Todos {
		todos[i] = kernel.Todo{
			Content:    t.Content,
			Status:     string(t.Status),
			ActiveForm: t.ActiveForm,
		}
	}
	return kernel.Session{
		ID:               s.ID,
		ParentSessionID:  s.ParentSessionID,
		Title:            s.Title,
		MessageCount:     s.MessageCount,
		PromptTokens:     s.PromptTokens,
		CompletionTokens: s.CompletionTokens,
		Cost:             s.Cost,
		Todos:            todos,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func (a *WorkspaceAdapter) kernelToSession(s *kernel.Session) session.Session {
	todos := make([]session.Todo, len(s.Todos))
	for i, t := range s.Todos {
		todos[i] = session.Todo{
			Content:    t.Content,
			Status:     session.TodoStatus(t.Status),
			ActiveForm: t.ActiveForm,
		}
	}
	return session.Session{
		ID:               s.ID,
		ParentSessionID:  s.ParentSessionID,
		Title:            s.Title,
		MessageCount:     s.MessageCount,
		PromptTokens:     s.PromptTokens,
		CompletionTokens: s.CompletionTokens,
		Cost:             s.Cost,
		Todos:            todos,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

// ------------------------------------------------------------------------
// Permissions
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) PermissionGrant(req kernel.PermissionRequest) {
	a.impl.PermissionGrant(a.kernelToPermissionReq(&req))
}

func (a *WorkspaceAdapter) PermissionGrantPersistent(req kernel.PermissionRequest) {
	a.impl.PermissionGrantPersistent(a.kernelToPermissionReq(&req))
}

func (a *WorkspaceAdapter) PermissionDeny(req kernel.PermissionRequest) {
	a.impl.PermissionDeny(a.kernelToPermissionReq(&req))
}

func (a *WorkspaceAdapter) PermissionSkipRequests() bool {
	return a.impl.PermissionSkipRequests()
}

func (a *WorkspaceAdapter) PermissionSetSkipRequests(skip bool) {
	a.impl.PermissionSetSkipRequests(skip)
}

func (a *WorkspaceAdapter) SubscribePermissions(ctx context.Context) <-chan kernel.Event[kernel.PermissionRequest] {
	ch := make(chan kernel.Event[kernel.PermissionRequest], 100)
	return ch
}

func (a *WorkspaceAdapter) SubscribePermissionNotifications(ctx context.Context) <-chan kernel.Event[kernel.PermissionNotification] {
	ch := make(chan kernel.Event[kernel.PermissionNotification], 100)
	return ch
}

func (a *WorkspaceAdapter) kernelToPermissionReq(req *kernel.PermissionRequest) permission.PermissionRequest {
	return permission.PermissionRequest{
		ID:          req.ID,
		SessionID:   req.SessionID,
		ToolCallID:  req.ToolCallID,
		ToolName:    req.ToolName,
		Description: req.Description,
		Action:      req.Action,
		Path:        req.Path,
	}
}

// ------------------------------------------------------------------------
// Notifications
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) SubscribeNotifications(ctx context.Context) <-chan kernel.Event[kernel.Notification] {
	ch := make(chan kernel.Event[kernel.Notification], 100)
	return ch
}

// ------------------------------------------------------------------------
// MCP Management
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) GetMCPStates() map[string]kernel.MCPClientInfo {
	states := a.impl.MCPGetStates()
	result := make(map[string]kernel.MCPClientInfo, len(states))
	for name, s := range states {
		result[name] = kernel.MCPClientInfo{
			Name:           s.Name,
			State:          kernel.MCPState(s.State),
			Error:          s.Error,
			ToolCount:      s.Counts.Tools,
			PromptsCount:   s.Counts.Prompts,
			ResourcesCount: s.Counts.Resources,
		}
	}
	return result
}

func (a *WorkspaceAdapter) SubscribeMCP(ctx context.Context) <-chan kernel.Event[kernel.MCPEvent] {
	ch := make(chan kernel.Event[kernel.MCPEvent], 100)
	return ch
}

func (a *WorkspaceAdapter) RefreshMCPTools(ctx context.Context, name string) {
	a.impl.RefreshMCPTools(ctx, name)
}

func (a *WorkspaceAdapter) RefreshMCPPrompts(ctx context.Context, name string) {
	a.impl.MCPRefreshPrompts(ctx, name)
}

func (a *WorkspaceAdapter) RefreshMCPResources(ctx context.Context, name string) {
	a.impl.MCPRefreshResources(ctx, name)
}

func (a *WorkspaceAdapter) GetMCPPrompt(clientID, promptID string, args map[string]string) (string, error) {
	return a.impl.GetMCPPrompt(clientID, promptID, args)
}

func (a *WorkspaceAdapter) ReadMCPResource(ctx context.Context, clientID, uri string) ([]kernel.MCPResourceContents, error) {
	contents, err := a.impl.ReadMCPResource(ctx, clientID, uri)
	if err != nil {
		return nil, err
	}
	result := make([]kernel.MCPResourceContents, len(contents))
	for i, c := range contents {
		result[i] = kernel.MCPResourceContents{
			URI:      c.URI,
			MimeType: c.MIMEType,
			Text:     c.Text,
			Blob:     c.Blob,
		}
	}
	return result, nil
}

func (a *WorkspaceAdapter) EnableDockerMCP(ctx context.Context) error {
	return a.impl.EnableDockerMCP(ctx)
}

func (a *WorkspaceAdapter) DisableDockerMCP() error {
	return a.impl.DisableDockerMCP()
}

// ------------------------------------------------------------------------
// LSP Events
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) LSPStart(ctx context.Context, path string) {
	a.impl.LSPStart(ctx, path)
}

func (a *WorkspaceAdapter) LSPStopAll(ctx context.Context) {
	a.impl.LSPStopAll(ctx)
}

func (a *WorkspaceAdapter) GetLSPStates() map[string]kernel.LSPClientInfo {
	states := a.impl.LSPGetStates()
	result := make(map[string]kernel.LSPClientInfo, len(states))
	for name, s := range states {
		result[name] = kernel.LSPClientInfo{
			Name:            s.Name,
			State:           kernel.LSPState(s.State),
			Error:           s.Error,
			DiagnosticCount: s.DiagnosticCount,
		}
	}
	return result
}

func (a *WorkspaceAdapter) SubscribeLSP(ctx context.Context) <-chan kernel.Event[kernel.LSPEvent] {
	ch := make(chan kernel.Event[kernel.LSPEvent], 100)
	return ch
}

// ------------------------------------------------------------------------
// File History Events
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) SubscribeHistory(ctx context.Context) <-chan kernel.Event[kernel.HistoryFile] {
	ch := make(chan kernel.Event[kernel.HistoryFile], 100)
	return ch
}

func (a *WorkspaceAdapter) RecordFileRead(ctx context.Context, sessionID, path string) {
	a.impl.FileTrackerRecordRead(ctx, sessionID, path)
}

func (a *WorkspaceAdapter) LastReadTime(ctx context.Context, sessionID, path string) int64 {
	return a.impl.FileTrackerLastReadTime(ctx, sessionID, path).Unix()
}

func (a *WorkspaceAdapter) ListReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	return a.impl.FileTrackerListReadFiles(ctx, sessionID)
}

// ------------------------------------------------------------------------
// History
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) ListSessionHistory(ctx context.Context, sessionID string) ([]kernel.HistoryFile, error) {
	files, err := a.impl.ListSessionHistory(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	result := make([]kernel.HistoryFile, len(files))
	for i, f := range files {
		result[i] = kernel.HistoryFile{
			Path: f.Path,
		}
	}
	return result, nil
}

// ------------------------------------------------------------------------
// Session ID Helpers
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) CreateAgentToolSessionID(messageID, toolCallID string) string {
	return a.impl.CreateAgentToolSessionID(messageID, toolCallID)
}

func (a *WorkspaceAdapter) ParseAgentToolSessionID(sessionID string) (messageID string, toolCallID string, ok bool) {
	return a.impl.ParseAgentToolSessionID(sessionID)
}

// ------------------------------------------------------------------------
// Model Selection
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) GetModel() kernel.ModelConfig {
	m := a.impl.AgentModel()
	return kernel.ModelConfig{
		ProviderID: m.ModelCfg.Provider,
		ModelID:    m.ModelCfg.Model,
	}
}

func (a *WorkspaceAdapter) SetModel(ctx context.Context, model kernel.ModelConfig) error {
	return a.impl.UpdateAgentModel(ctx)
}

func (a *WorkspaceAdapter) GetDefaultSmallModel(providerID string) kernel.ModelConfig {
	m := a.impl.GetDefaultSmallModel(providerID)
	return kernel.ModelConfig{
		ProviderID: m.Provider,
		ModelID:    m.Model,
	}
}

func (a *WorkspaceAdapter) InitAgent(ctx context.Context) error {
	return a.impl.InitCoderAgent(ctx)
}

// ------------------------------------------------------------------------
// Configuration
// ------------------------------------------------------------------------

// ResolverAdapter wraps engine VariableResolver to implement kernel.VariableResolver.
type ResolverAdapter struct {
	impl config.VariableResolver
}

func (r *ResolverAdapter) Resolve(key string) (string, bool) {
	val, err := r.impl.ResolveValue(key)
	if err != nil {
		return "", false
	}
	return val, true
}

func (a *WorkspaceAdapter) Config() kernel.ConfigProvider {
	return &ConfigAdapter{cfg: a.impl.Config()}
}

func (a *WorkspaceAdapter) WorkingDir() string {
	return a.impl.WorkingDir()
}

func (a *WorkspaceAdapter) Resolver() kernel.VariableResolver {
	return &ResolverAdapter{impl: a.impl.Resolver()}
}

// scopeToConfig converts kernel.Scope (string) to config.Scope (int).
func scopeToConfig(s kernel.Scope) config.Scope {
	switch s {
	case "global":
		return config.ScopeGlobal
	case "workspace":
		return config.ScopeWorkspace
	default:
		return config.ScopeGlobal
	}
}

// ConfigAdapter wraps engine config.Config to implement kernel.ConfigProvider.
// Note: Some methods return nil/placeholder since the underlying Config
// and ConfigStore have different structures.
type ConfigAdapter struct {
	cfg *config.Config
}

func (c *ConfigAdapter) Get() *kernel.Config {
	return nil
}

func (c *ConfigAdapter) GetModelInfo(providerID, modelID string) kernel.ModelInfo {
	model := c.cfg.GetModel(providerID, modelID)
	if model == nil {
		return kernel.ModelInfo{Name: "Unknown Model"}
	}
	return kernel.ModelInfo{Name: model.Name}
}

func (c *ConfigAdapter) GetProviderInfo(providerID string) kernel.ProviderInfo {
	if providerConfig, ok := c.cfg.Providers.Get(providerID); ok {
		return kernel.ProviderInfo{Name: providerConfig.Name}
	}
	return kernel.ProviderInfo{Name: providerID}
}

func (c *ConfigAdapter) GetPreferredModel(scope string) kernel.ModelConfig {
	return kernel.ModelConfig{}
}

func (c *ConfigAdapter) SetPreferredModel(scope string, model kernel.ModelConfig) error {
	return nil // Config doesn't have this method
}

// ------------------------------------------------------------------------
// Config Mutations
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) UpdatePreferredModel(scope kernel.Scope, modelType kernel.ModelType, model kernel.ModelConfig) error {
	return a.impl.UpdatePreferredModel(scopeToConfig(scope), config.SelectedModelType(modelType), config.SelectedModel{
		Provider: model.ProviderID,
		Model:    model.ModelID,
	})
}

func (a *WorkspaceAdapter) SetCompactMode(scope kernel.Scope, enabled bool) error {
	return a.impl.SetCompactMode(scopeToConfig(scope), enabled)
}

func (a *WorkspaceAdapter) SetProviderAPIKey(scope kernel.Scope, providerID string, apiKey any) error {
	return a.impl.SetProviderAPIKey(scopeToConfig(scope), providerID, apiKey)
}

func (a *WorkspaceAdapter) SetConfigField(scope kernel.Scope, key string, value any) error {
	return a.impl.SetConfigField(scopeToConfig(scope), key, value)
}

func (a *WorkspaceAdapter) RemoveConfigField(scope kernel.Scope, key string) error {
	return a.impl.RemoveConfigField(scopeToConfig(scope), key)
}

func (a *WorkspaceAdapter) ImportCopilot() (*kernel.OAuthToken, bool) {
	token, ok := a.impl.ImportCopilot()
	if !ok || token == nil {
		return nil, false
	}
	return &kernel.OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
		TokenType:    "Bearer",
	}, true
}

func (a *WorkspaceAdapter) RefreshOAuthToken(ctx context.Context, scope kernel.Scope, providerID string) error {
	return a.impl.RefreshOAuthToken(ctx, scopeToConfig(scope), providerID)
}

// ------------------------------------------------------------------------
// Project Lifecycle
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) ProjectNeedsInitialization() (bool, error) {
	return a.impl.ProjectNeedsInitialization()
}

func (a *WorkspaceAdapter) MarkProjectInitialized() error {
	return a.impl.MarkProjectInitialized()
}

func (a *WorkspaceAdapter) InitializePrompt() (string, error) {
	return a.impl.InitializePrompt()
}

// ------------------------------------------------------------------------
// Events
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) Subscribe(program kernel.Program) {
}

func (a *WorkspaceAdapter) KernelSubscribe(program kernel.Program) {
}

// ------------------------------------------------------------------------
// Lifecycle
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) Shutdown() {
	a.impl.Shutdown()
}
