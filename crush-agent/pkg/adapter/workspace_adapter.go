// Package adapter provides implementations of kernel interfaces that wrap
// internal engine packages. This allows external packages (like UI) to
// depend only on kernel interfaces while using the existing engine implementations.
package adapter

import (
	"context"
	"fmt"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/qinqd2006/crush/infra/pkg/commands"
	"github.com/qinqd2006/crush/infra/pkg/permission"
	"github.com/qinqd2006/crush/kernel/pkg"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/agent/tools"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/config"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/history"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/message"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/session"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/workspace"
)

// WorkspaceAdapter wraps an engine workspace.Workspace and implements
// kernel.Workspace by converting between engine and kernel types.
type WorkspaceAdapter struct {
	impl  workspace.Workspace
	store *config.ConfigStore
}

// NewWorkspaceAdapter creates a new WorkspaceAdapter wrapping an engine workspace.
func NewWorkspaceAdapter(impl workspace.Workspace, store *config.ConfigStore) *WorkspaceAdapter {
	return &WorkspaceAdapter{impl: impl, store: store}
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

func (a *WorkspaceAdapter) AgentCancel(sessionID string) {
	a.impl.AgentCancel(sessionID)
}

func (a *WorkspaceAdapter) CancelAll() {
	// Engine workspace doesn't have AgentCancelAll, cancel current session
}

func (a *WorkspaceAdapter) IsBusy() bool {
	return a.impl.AgentIsBusy()
}

func (a *WorkspaceAdapter) AgentIsBusy() bool {
	return a.impl.AgentIsBusy()
}

func (a *WorkspaceAdapter) IsSessionBusy(sessionID string) bool {
	return a.impl.AgentIsSessionBusy(sessionID)
}

// AgentIsReady is an alias for IsBusy for backwards compatibility.
func (a *WorkspaceAdapter) AgentIsReady() bool {
	return a.impl.AgentIsReady()
}

// AgentIsSessionBusy is an alias for IsSessionBusy for backwards compatibility.
func (a *WorkspaceAdapter) AgentIsSessionBusy(sessionID string) bool {
	return a.impl.AgentIsSessionBusy(sessionID)
}

func (a *WorkspaceAdapter) QueuedPrompts(sessionID string) int {
	return a.impl.AgentQueuedPrompts(sessionID)
}

func (a *WorkspaceAdapter) AgentQueuedPrompts(sessionID string) int {
	return a.impl.AgentQueuedPrompts(sessionID)
}

func (a *WorkspaceAdapter) QueuedPromptsList(sessionID string) []string {
	return a.impl.AgentQueuedPromptsList(sessionID)
}

func (a *WorkspaceAdapter) AgentQueuedPromptsList(sessionID string) []string {
	return a.impl.AgentQueuedPromptsList(sessionID)
}

func (a *WorkspaceAdapter) ClearQueue(sessionID string) {
	a.impl.AgentClearQueue(sessionID)
}

func (a *WorkspaceAdapter) AgentClearQueue(sessionID string) {
	a.impl.AgentClearQueue(sessionID)
}

func (a *WorkspaceAdapter) Summarize(ctx context.Context, sessionID string) error {
	return a.impl.AgentSummarize(ctx, sessionID)
}

func (a *WorkspaceAdapter) AgentSummarize(ctx context.Context, sessionID string) error {
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

func (a *WorkspaceAdapter) ListUserMessages(ctx context.Context, sessionID string) ([]kernel.Message, error) {
	msgs, err := a.impl.ListUserMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return a.messagesToKernel(msgs), nil
}

func (a *WorkspaceAdapter) ListAllUserMessages(ctx context.Context) ([]kernel.Message, error) {
	msgs, err := a.impl.ListAllUserMessages(ctx)
	if err != nil {
		return nil, err
	}
	return a.messagesToKernel(msgs), nil
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
		counts := a.impl.LSPGetDiagnosticCounts(name)
		result[name] = kernel.LSPClientInfo{
			Name:  s.Name,
			State: kernel.LSPState(s.State),
			Error: s.Error,
			Diagnostics: kernel.DiagnosticCounts{
				Error:       counts.Error,
				Warning:     counts.Warning,
				Information: counts.Information,
				Hint:        counts.Hint,
			},
		}
	}
	return result
}

func (a *WorkspaceAdapter) LSPGetStates() map[string]kernel.LSPClientInfo {
	return a.GetLSPStates()
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

func (a *WorkspaceAdapter) FileTrackerRecordRead(ctx context.Context, sessionID, path string) {
	a.RecordFileRead(ctx, sessionID, path)
}

func (a *WorkspaceAdapter) LastReadTime(ctx context.Context, sessionID, path string) int64 {
	return a.impl.FileTrackerLastReadTime(ctx, sessionID, path).Unix()
}

func (a *WorkspaceAdapter) FileTrackerLastReadTime(ctx context.Context, sessionID, path string) int64 {
	return a.LastReadTime(ctx, sessionID, path)
}

func (a *WorkspaceAdapter) ListReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	return a.impl.FileTrackerListReadFiles(ctx, sessionID)
}

func (a *WorkspaceAdapter) FileTrackerListReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	return a.ListReadFiles(ctx, sessionID)
}

// ------------------------------------------------------------------------
// History
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) ListSessionHistory(ctx context.Context, sessionID string) ([]kernel.HistoryFile, error) {
	files, err := a.impl.ListSessionHistory(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// Group files by path
	filesByPath := make(map[string][]history.File)
	for _, f := range files {
		filesByPath[f.Path] = append(filesByPath[f.Path], f)
	}
	result := make([]kernel.HistoryFile, 0, len(filesByPath))
	for path, versions := range filesByPath {
		if len(versions) == 0 {
			continue
		}
		// Find first and last versions by Version field
		first := versions[0]
		last := versions[0]
		for _, v := range versions {
			if v.Version < first.Version {
				first = v
			}
			if v.Version > last.Version {
				last = v
			}
		}
		// Build FileVersion slice
		kernelVersions := make([]kernel.FileVersion, len(versions))
		for i, v := range versions {
			kernelVersions[i] = kernel.FileVersion{
				Path:      v.Path,
				Hash:      fmt.Sprintf("%d", v.Version), // Use Version as hash since we don't have actual hash
				Timestamp: v.CreatedAt,
			}
		}
		result = append(result, kernel.HistoryFile{
			Path:     path,
			Content:  last.Content, // Use latest content for diffing
			Versions: kernelVersions,
		})
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

func (a *WorkspaceAdapter) AgentModel() kernel.AgentModel {
	m := a.impl.AgentModel()
	kernelModel, ok := a.Config().GetModel(m.ModelCfg.Provider, m.ModelCfg.Model)
	if !ok {
		return kernel.AgentModel{}
	}
	return kernel.AgentModel{
		Model: kernelModel,
		ModelCfg: kernel.SelectedModel{
			Model:            m.ModelCfg.Model,
			Provider:         m.ModelCfg.Provider,
			ReasoningEffort:  m.ModelCfg.ReasoningEffort,
			Think:            m.ModelCfg.Think,
			MaxTokens:        m.ModelCfg.MaxTokens,
			Temperature:      m.ModelCfg.Temperature,
			TopP:             m.ModelCfg.TopP,
			TopK:             m.ModelCfg.TopK,
			FrequencyPenalty: m.ModelCfg.FrequencyPenalty,
			PresencePenalty:  m.ModelCfg.PresencePenalty,
			ProviderOptions:  m.ModelCfg.ProviderOptions,
		},
	}
}

func (a *WorkspaceAdapter) SetModel(ctx context.Context, model kernel.ModelConfig) error {
	return a.impl.UpdateAgentModel(ctx)
}

func (a *WorkspaceAdapter) UpdateAgentModel(ctx context.Context) error {
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

func (a *WorkspaceAdapter) InitCoderAgent(ctx context.Context) error {
	return a.InitAgent(ctx)
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
	return &ConfigAdapter{store: a.store}
}

// EngineConfig returns the underlying engine config for backwards compatibility.
func (a *WorkspaceAdapter) EngineConfig() any {
	return a.impl.Config()
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
type ConfigAdapter struct {
	store *config.ConfigStore
}

func (c *ConfigAdapter) Get() *kernel.Config {
	return c.toKernelConfig()
}

func (c *ConfigAdapter) IsConfigured() bool {
	cfg := c.store.Config()
	return cfg.IsConfigured()
}

func (c *ConfigAdapter) toKernelConfig() *kernel.Config {
	cfg := c.store.Config()
	kcfg := &kernel.Config{
		Models:       make(map[kernel.SelectedModelType]kernel.SelectedModel),
		RecentModels: make(map[kernel.SelectedModelType][]kernel.SelectedModel),
		Agents:       make(map[string]kernel.AgentConfig),
		Providers:    make(map[string]kernel.Provider),
	}

	// Copy selected models
	for k, v := range cfg.Models {
		kcfg.Models[kernel.SelectedModelType(k)] = kernel.SelectedModel{
			Model:            v.Model,
			Provider:         v.Provider,
			ReasoningEffort:  v.ReasoningEffort,
			Think:            v.Think,
			MaxTokens:        v.MaxTokens,
			Temperature:      v.Temperature,
			TopP:             v.TopP,
			TopK:             v.TopK,
			FrequencyPenalty: v.FrequencyPenalty,
			PresencePenalty:  v.PresencePenalty,
			ProviderOptions:  v.ProviderOptions,
		}
	}

	// Copy recent models
	for k, v := range cfg.RecentModels {
		recent := make([]kernel.SelectedModel, len(v))
		for i, r := range v {
			recent[i] = kernel.SelectedModel{
				Model:            r.Model,
				Provider:         r.Provider,
				ReasoningEffort:  r.ReasoningEffort,
				Think:            r.Think,
				MaxTokens:        r.MaxTokens,
				Temperature:      r.Temperature,
				TopP:             r.TopP,
				TopK:             r.TopK,
				FrequencyPenalty: r.FrequencyPenalty,
				PresencePenalty:  r.PresencePenalty,
				ProviderOptions:  r.ProviderOptions,
			}
		}
		kcfg.RecentModels[kernel.SelectedModelType(k)] = recent
	}

	// Copy agents
	for k, v := range cfg.Agents {
		kcfg.Agents[k] = kernel.AgentConfig{
			ID:           v.ID,
			Name:         v.Name,
			Description:  v.Description,
			Model:        kernel.SelectedModelType(v.Model),
			AllowedTools: v.AllowedTools,
			AllowedMCP:   v.AllowedMCP,
			ContextPaths: v.ContextPaths,
		}
	}

	// Copy providers (from config.Providers map)
	for k, v := range cfg.Providers.Seq2() {
		prov := kernel.Provider{
			ID:          k,
			Name:        v.Name,
			Type:        string(v.Type),
			APIEndpoint: v.BaseURL,
			Models:      make([]kernel.Model, len(v.Models)),
		}
		for i, m := range v.Models {
			prov.Models[i] = kernel.Model{
				ID:                     m.ID,
				Name:                   m.Name,
				CostPer1MIn:            m.CostPer1MIn,
				CostPer1MOut:           m.CostPer1MOut,
				CostPer1MInCached:      m.CostPer1MInCached,
				CostPer1MOutCached:     m.CostPer1MOutCached,
				ContextWindow:          m.ContextWindow,
				DefaultMaxTokens:       m.DefaultMaxTokens,
				CanReason:              m.CanReason,
				ReasoningLevels:        m.ReasoningLevels,
				DefaultReasoningEffort: m.DefaultReasoningEffort,
				SupportsImages:         m.SupportsImages,
			}
		}
		kcfg.Providers[k] = prov
	}

	return kcfg
}

func (c *ConfigAdapter) SetConfigField(scope kernel.Scope, key string, value any) error {
	return c.store.SetConfigField(scopeToConfig(scope), key, value)
}

func (c *ConfigAdapter) GetModelInfo(providerID, modelID string) kernel.ModelInfo {
	model := c.store.Config().GetModel(providerID, modelID)
	if model == nil {
		return kernel.ModelInfo{Name: "Unknown Model"}
	}
	return kernel.ModelInfo{Name: model.Name}
}

func (c *ConfigAdapter) GetProviderInfo(providerID string) kernel.ProviderInfo {
	if providerConfig, ok := c.store.Config().Providers.Get(providerID); ok {
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

func (c *ConfigAdapter) Providers() ([]kernel.Provider, error) {
	providers := make([]kernel.Provider, 0)
	for id, prov := range c.store.Config().Providers.Seq2() {
		kprov := kernel.Provider{
			ID:          id,
			Name:        prov.Name,
			Type:        string(prov.Type),
			APIEndpoint: prov.BaseURL,
			Models:      make([]kernel.Model, len(prov.Models)),
		}
		for i, m := range prov.Models {
			kprov.Models[i] = kernel.Model{
				ID:                     m.ID,
				Name:                   m.Name,
				CostPer1MIn:            m.CostPer1MIn,
				CostPer1MOut:           m.CostPer1MOut,
				CostPer1MInCached:      m.CostPer1MInCached,
				CostPer1MOutCached:     m.CostPer1MOutCached,
				ContextWindow:          m.ContextWindow,
				DefaultMaxTokens:       m.DefaultMaxTokens,
				CanReason:              m.CanReason,
				ReasoningLevels:        m.ReasoningLevels,
				DefaultReasoningEffort: m.DefaultReasoningEffort,
				SupportsImages:         m.SupportsImages,
			}
		}
		providers = append(providers, kprov)
	}
	return providers, nil
}

func (c *ConfigAdapter) GetAgent(name string) (kernel.AgentConfig, bool) {
	agent, ok := c.store.Config().Agents[name]
	if !ok {
		return kernel.AgentConfig{}, false
	}
	return kernel.AgentConfig{
		ID:           agent.ID,
		Name:         agent.Name,
		Description:  agent.Description,
		Model:        kernel.SelectedModelType(agent.Model),
		AllowedTools: agent.AllowedTools,
		AllowedMCP:   agent.AllowedMCP,
		ContextPaths: agent.ContextPaths,
	}, true
}

func (c *ConfigAdapter) GetSelectedModel(modelType kernel.SelectedModelType) (kernel.SelectedModel, bool) {
	sel, ok := c.store.Config().Models[config.SelectedModelType(modelType)]
	if !ok {
		return kernel.SelectedModel{}, false
	}
	return kernel.SelectedModel{
		Model:            sel.Model,
		Provider:         sel.Provider,
		ReasoningEffort:  sel.ReasoningEffort,
		Think:            sel.Think,
		MaxTokens:        sel.MaxTokens,
		Temperature:      sel.Temperature,
		TopP:             sel.TopP,
		TopK:             sel.TopK,
		FrequencyPenalty: sel.FrequencyPenalty,
		PresencePenalty:  sel.PresencePenalty,
		ProviderOptions:  sel.ProviderOptions,
	}, true
}

func (c *ConfigAdapter) GetModel(providerID, modelID string) (kernel.Model, bool) {
	model := c.store.Config().GetModel(providerID, modelID)
	if model == nil {
		return kernel.Model{}, false
	}
	return kernel.Model{
		ID:                     model.ID,
		Name:                   model.Name,
		CostPer1MIn:            model.CostPer1MIn,
		CostPer1MOut:           model.CostPer1MOut,
		CostPer1MInCached:      model.CostPer1MInCached,
		CostPer1MOutCached:     model.CostPer1MOutCached,
		ContextWindow:          model.ContextWindow,
		DefaultMaxTokens:       model.DefaultMaxTokens,
		CanReason:              model.CanReason,
		ReasoningLevels:        model.ReasoningLevels,
		DefaultReasoningEffort: model.DefaultReasoningEffort,
		SupportsImages:         model.SupportsImages,
	}, true
}

// ------------------------------------------------------------------------
// UI View Helpers
// ------------------------------------------------------------------------

// GetCoderAgent returns the coder agent with its current model capabilities.
func (c *ConfigAdapter) GetCoderAgent() kernel.AgentView {
	cfg := c.store.Config()
	agentCfg, ok := cfg.Agents[config.AgentCoder]
	if !ok {
		return kernel.AgentView{}
	}

	selectedModel, selOk := cfg.Models[agentCfg.Model]
	if !selOk {
		return kernel.AgentView{Role: "coder", Name: agentCfg.Name}
	}

	modelView, _ := c.selectedModelToModelView(kernel.SelectedModelType(agentCfg.Model), selectedModel)
	return kernel.AgentView{
		Role:         "coder",
		Name:         agentCfg.Name,
		CurrentModel: modelView,
	}
}

// GetSmallModel returns the small/fast model configuration if set.
func (c *ConfigAdapter) GetSmallModel() (kernel.ModelView, bool) {
	cfg := c.store.Config()
	selectedModel, ok := cfg.Models[config.SelectedModelTypeSmall]
	if !ok {
		return kernel.ModelView{}, false
	}
	return c.selectedModelToModelView(kernel.SelectedModelTypeSmall, selectedModel)
}

// GetModelViewByType returns the model view by selected model type.
func (c *ConfigAdapter) GetModelViewByType(modelType kernel.SelectedModelType) (kernel.ModelView, bool) {
	cfg := c.store.Config()
	selectedModel, ok := cfg.Models[config.SelectedModelType(modelType)]
	if !ok {
		return kernel.ModelView{}, false
	}
	return c.selectedModelToModelView(modelType, selectedModel)
}

// selectedModelToModelView converts a selected model to a ModelView.
func (c *ConfigAdapter) selectedModelToModelView(modelType kernel.SelectedModelType, selectedModel config.SelectedModel) (kernel.ModelView, bool) {
	model := c.store.Config().GetModel(selectedModel.Provider, selectedModel.Model)
	if model == nil {
		return kernel.ModelView{
			Type:       modelType,
			ProviderID: selectedModel.Provider,
			ModelID:    selectedModel.Model,
		}, false
	}
	return kernel.ModelView{
		Type:          modelType,
		Name:          model.Name,
		ProviderID:    selectedModel.Provider,
		ModelID:       selectedModel.Model,
		ContextWindow: model.ContextWindow,
		SupportsImage: model.SupportsImages,
	}, true
}

// GetGlobalOptions returns global application options.
func (c *ConfigAdapter) GetGlobalOptions() kernel.GlobalOptions {
	cfg := c.store.Config()
	opts := cfg.Options
	if opts == nil {
		return kernel.GlobalOptions{}
	}
	progress := true
	if opts.Progress != nil {
		progress = *opts.Progress
	}
	return kernel.GlobalOptions{
		DisableNotifications: opts.DisableNotifications,
		InitializeAs:         opts.InitializeAs,
		Progress:             progress,
	}
}

// TUIOptions returns TUI-specific display preferences.
func (c *ConfigAdapter) TUIOptions() kernel.TUIOptions {
	cfg := c.store.Config()
	opts := cfg.Options
	if opts == nil || opts.TUI == nil {
		return kernel.TUIOptions{}
	}
	tuiOpts := opts.TUI
	transparent := false
	if tuiOpts.Transparent != nil {
		transparent = *tuiOpts.Transparent
	}
	depth, items := tuiOpts.Completions.Limits()
	return kernel.TUIOptions{
		CompactMode: tuiOpts.CompactMode,
		DiffMode:    tuiOpts.DiffMode,
		Completions: kernel.CompletionsLimits{
			MaxDepth: depth,
			MaxItems: items,
		},
		Transparent: transparent,
	}
}

// GetProvider returns a provider by ID.
func (c *ConfigAdapter) GetProvider(providerID string) (kernel.Provider, bool) {
	cfg := c.store.Config()
	providerCfg, ok := cfg.Providers.Get(providerID)
	if !ok {
		return kernel.Provider{}, false
	}
	return kernel.Provider{
		ID:   providerCfg.ID,
		Name: providerCfg.Name,
		Type: string(providerCfg.Type),
	}, true
}

// ListMCPServers returns a list of MCP server metadata for UI display.
func (c *ConfigAdapter) ListMCPServers() []kernel.MCPServerInfo {
	cfg := c.store.Config()
	mcps := cfg.MCP.Sorted()
	servers := make([]kernel.MCPServerInfo, len(mcps))
	for i, mcp := range mcps {
		servers[i] = kernel.MCPServerInfo{
			Name: mcp.Name,
			Type: string(mcp.MCP.Type),
		}
	}
	return servers
}

// ListCustomCommands returns a list of custom command metadata for UI display.
func (c *ConfigAdapter) ListCustomCommands() []kernel.CustomCommandInfo {
	cfg := c.store.Config()
	dataDir := ".crush"
	if cfg.Options != nil && cfg.Options.DataDirectory != "" {
		dataDir = cfg.Options.DataDirectory
	}
	cmds, err := commands.LoadCustomCommands(dataDir)
	if err != nil {
		return nil
	}
	result := make([]kernel.CustomCommandInfo, len(cmds))
	for i, cmd := range cmds {
		result[i] = kernel.CustomCommandInfo{
			Name:        cmd.Name,
			Description: cmd.Content,
		}
	}
	return result
}

// ------------------------------------------------------------------------
// Config Mutations
// ------------------------------------------------------------------------

func (a *WorkspaceAdapter) UpdatePreferredModel(scope kernel.Scope, modelType kernel.SelectedModelType, model kernel.SelectedModel) error {
	return a.impl.UpdatePreferredModel(scopeToConfig(scope), modelType, model)
}

func (a *WorkspaceAdapter) SetCompactMode(scope kernel.Scope, enabled bool) error {
	return a.impl.SetCompactMode(scopeToConfig(scope), enabled)
}

func (a *WorkspaceAdapter) SetProviderAPIKey(scope kernel.Scope, providerID string, apiKey any) error {
	return a.impl.SetProviderAPIKey(scopeToConfig(scope), providerID, apiKey)
}

func (a *WorkspaceAdapter) TestProviderConnection(provider kernel.Provider, apiKey string) error {
	providerConfig := config.ProviderConfig{
		ID:      provider.ID,
		Name:    provider.Name,
		APIKey:  apiKey,
		Type:    catwalk.Type(provider.Type),
		BaseURL: provider.APIEndpoint,
	}
	return providerConfig.TestConnection(a.impl.Resolver())
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
		ExpiresIn:    token.ExpiresIn,
		ExpiresAt:    token.ExpiresAt,
		TokenType:    "Bearer",
	}, true
}

func (a *WorkspaceAdapter) RefreshOAuthToken(ctx context.Context, scope kernel.Scope, providerID string) error {
	return a.impl.RefreshOAuthToken(ctx, scopeToConfig(scope), providerID)
}

func (a *WorkspaceAdapter) GetDockerMCPAvailability() (bool, bool) {
	return config.DockerMCPAvailabilityCached()
}

func (a *WorkspaceAdapter) RefreshDockerMCPAvailability() bool {
	return config.RefreshDockerMCPAvailability()
}

func (a *WorkspaceAdapter) GlobalConfigPath() string {
	return config.GlobalConfigData()
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

// ResetCache clears compiled regex caches used by search tools.
func (a *WorkspaceAdapter) ResetCache() {
	tools.ResetCache()
}
