// Package remote provides a kernel.Kernel implementation for client/server mode.
// It wraps the client.Client and translates proto events to kernel events.
package remote

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/qinqd2006/crush/infra/pkg/pubsub"
	"github.com/qinqd2006/crush/kernel/pkg"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/client"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/config"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/message"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/proto"
)

// Compile-time check that RemoteKernel implements kernel.Kernel.
var _ kernel.Kernel = (*RemoteKernel)(nil)

// RemoteKernel implements kernel.Kernel for client/server mode.
// It translates proto events from the server into kernel events.
type RemoteKernel struct {
	client      *client.Client
	workspaceID string

	// Subscription channels
	sessionsCh    chan kernel.Event[kernel.Session]
	messagesCh    chan kernel.Event[kernel.Message]
	permissionsCh chan kernel.Event[kernel.PermissionRequest]
	permNotifCh   chan kernel.Event[kernel.PermissionNotification]
	notifsCh      chan kernel.Event[kernel.Notification]
	mcpCh         chan kernel.Event[kernel.MCPEvent]
	lspCh         chan kernel.Event[kernel.LSPEvent]
	historyCh     chan kernel.Event[kernel.HistoryFile]

	mu         sync.RWMutex
	cancelFunc context.CancelFunc
}

// NewRemoteKernel creates a new RemoteKernel wrapping the given client and workspace ID.
func NewRemoteKernel(c *client.Client, workspaceID string) kernel.Kernel {
	return &RemoteKernel{
		client:      c,
		workspaceID: workspaceID,
	}
}

// ============================================================================
// Configuration
// ============================================================================

// remoteConfigAdapter is a ConfigProvider for RemoteKernel.
// In client/server mode, config is managed server-side.
type remoteConfigAdapter struct {
	kernel *RemoteKernel
}

func (r *remoteConfigAdapter) Get() *kernel.Config {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return &kernel.Config{}
	}
	return r.toKernelConfig(cfg)
}

func (r *remoteConfigAdapter) IsConfigured() bool {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return false
	}
	return cfg.IsConfigured()
}

func (r *remoteConfigAdapter) toKernelConfig(cfg *config.Config) *kernel.Config {
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

func (r *remoteConfigAdapter) GetPreferredModel(scope string) kernel.ModelConfig {
	return kernel.ModelConfig{}
}
func (r *remoteConfigAdapter) SetPreferredModel(scope string, model kernel.ModelConfig) error {
	return nil
}
func (r *remoteConfigAdapter) GetModelInfo(providerID, modelID string) kernel.ModelInfo {
	if cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID); err == nil {
		if model := cfg.GetModel(providerID, modelID); model != nil {
			return kernel.ModelInfo{Name: model.Name}
		}
	}
	return kernel.ModelInfo{Name: "Unknown Model"}
}
func (r *remoteConfigAdapter) GetProviderInfo(providerID string) kernel.ProviderInfo {
	if cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID); err == nil {
		if prov, ok := cfg.Providers.Get(providerID); ok {
			return kernel.ProviderInfo{Name: prov.Name}
		}
	}
	return kernel.ProviderInfo{Name: providerID}
}
func (r *remoteConfigAdapter) Providers() ([]kernel.Provider, error) {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return nil, err
	}

	providers := make([]kernel.Provider, 0)
	for id, prov := range cfg.Providers.Seq2() {
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
func (r *remoteConfigAdapter) GetAgent(name string) (kernel.AgentConfig, bool) {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return kernel.AgentConfig{}, false
	}
	if agent, ok := cfg.Agents[name]; ok {
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
	return kernel.AgentConfig{}, false
}
func (r *remoteConfigAdapter) GetSelectedModel(modelType kernel.SelectedModelType) (kernel.SelectedModel, bool) {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return kernel.SelectedModel{}, false
	}
	if sel, ok := cfg.Models[config.SelectedModelType(modelType)]; ok {
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
	return kernel.SelectedModel{}, false
}
func (r *remoteConfigAdapter) GetModel(providerID, modelID string) (kernel.Model, bool) {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return kernel.Model{}, false
	}
	if model := cfg.GetModel(providerID, modelID); model != nil {
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
	return kernel.Model{}, false
}

// GetCoderAgent returns the coder agent with its current model capabilities.
func (r *remoteConfigAdapter) GetCoderAgent() kernel.AgentView {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return kernel.AgentView{}
	}
	agentCfg, ok := cfg.Agents[config.AgentCoder]
	if !ok {
		return kernel.AgentView{}
	}

	selectedModel, selOk := cfg.Models[agentCfg.Model]
	if !selOk {
		return kernel.AgentView{Role: "coder", Name: agentCfg.Name}
	}

	modelView, _ := r.selectedModelToModelView(kernel.SelectedModelType(agentCfg.Model), selectedModel)
	return kernel.AgentView{
		Role:         "coder",
		Name:         agentCfg.Name,
		CurrentModel: modelView,
	}
}

// GetSmallModel returns the small/fast model configuration if set.
func (r *remoteConfigAdapter) GetSmallModel() (kernel.ModelView, bool) {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return kernel.ModelView{}, false
	}
	selectedModel, ok := cfg.Models[config.SelectedModelTypeSmall]
	if !ok {
		return kernel.ModelView{}, false
	}
	return r.selectedModelToModelView(kernel.SelectedModelTypeSmall, selectedModel)
}

// GetModelViewByType returns the model view by selected model type.
func (r *remoteConfigAdapter) GetModelViewByType(modelType kernel.SelectedModelType) (kernel.ModelView, bool) {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return kernel.ModelView{}, false
	}
	selectedModel, ok := cfg.Models[config.SelectedModelType(modelType)]
	if !ok {
		return kernel.ModelView{}, false
	}
	return r.selectedModelToModelView(modelType, selectedModel)
}

// selectedModelToModelView converts a selected model to a ModelView.
func (r *remoteConfigAdapter) selectedModelToModelView(modelType kernel.SelectedModelType, selectedModel config.SelectedModel) (kernel.ModelView, bool) {
	cfg, _ := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	model := cfg.GetModel(selectedModel.Provider, selectedModel.Model)
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
func (r *remoteConfigAdapter) GetGlobalOptions() kernel.GlobalOptions {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil || cfg.Options == nil {
		return kernel.GlobalOptions{}
	}
	progress := true
	if cfg.Options.Progress != nil {
		progress = *cfg.Options.Progress
	}
	return kernel.GlobalOptions{
		DisableNotifications: cfg.Options.DisableNotifications,
		InitializeAs:         cfg.Options.InitializeAs,
		Progress:             progress,
	}
}

// TUIOptions returns TUI-specific display preferences.
func (r *remoteConfigAdapter) TUIOptions() kernel.TUIOptions {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil || cfg.Options == nil || cfg.Options.TUI == nil {
		return kernel.TUIOptions{}
	}
	tuiOpts := cfg.Options.TUI
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
func (r *remoteConfigAdapter) GetProvider(providerID string) (kernel.Provider, bool) {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return kernel.Provider{}, false
	}
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
func (r *remoteConfigAdapter) ListMCPServers() []kernel.MCPServerInfo {
	cfg, err := r.kernel.client.GetConfig(context.Background(), r.kernel.workspaceID)
	if err != nil {
		return nil
	}
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
// For remote mode, this returns an empty list as custom commands are loaded locally.
func (r *remoteConfigAdapter) ListCustomCommands() []kernel.CustomCommandInfo {
	return nil
}

// scopeToConfig converts kernel.Scope to config.Scope.
func scopeToRemoteConfig(s kernel.Scope) config.Scope {
	switch s {
	case "global":
		return config.ScopeGlobal
	case "workspace":
		return config.ScopeWorkspace
	default:
		return config.ScopeGlobal
	}
}

func (r *remoteConfigAdapter) SetConfigField(scope kernel.Scope, key string, value any) error {
	return r.kernel.client.SetConfigField(
		context.Background(),
		r.kernel.workspaceID,
		scopeToRemoteConfig(scope),
		key,
		value,
	)
}

func (k *RemoteKernel) Config() kernel.ConfigProvider {
	return &remoteConfigAdapter{kernel: k}
}

// ============================================================================
// Lifecycle
// ============================================================================

func (k *RemoteKernel) Shutdown() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.cancelFunc != nil {
		k.cancelFunc()
	}
}

// ============================================================================
// Agent Management
// ============================================================================

// ListAgents implements kernel.Kernel.
// For remote mode, returns the agents configured on the server.
func (k *RemoteKernel) ListAgents() []string {
	cfg, err := k.client.GetConfig(context.Background(), k.workspaceID)
	if err != nil || cfg == nil {
		return []string{"coder"}
	}
	names := make([]string, 0, len(cfg.Agents))
	for name := range cfg.Agents {
		names = append(names, name)
	}
	return names
}

// GetActiveAgent implements kernel.Kernel.
// For remote mode, always returns "coder" as the active agent.
func (k *RemoteKernel) GetActiveAgent() string {
	return "coder"
}

// SetActiveAgent implements kernel.Kernel.
// For remote mode, agent switching is not yet supported.
func (k *RemoteKernel) SetActiveAgent(name string) error {
	return errors.New("agent switching not supported in remote mode")
}

// GetAgent implements kernel.Kernel.
// For remote mode, returns nil as agents are managed server-side.
func (k *RemoteKernel) GetAgent(name string) kernel.Agent {
	return nil
}

// ============================================================================
// Agent Execution
// ============================================================================

func (k *RemoteKernel) RunPrompt(ctx context.Context, sessionID, prompt string, attachments ...kernel.Attachment) error {
	// Convert kernel attachments to message.Attachments
	var atts []message.Attachment
	for _, a := range attachments {
		atts = append(atts, message.Attachment{
			FilePath: a.FilePath,
			FileName: a.FileName,
			MimeType: a.MimeType,
			Content:  a.Content,
		})
	}

	return k.client.SendMessage(ctx, k.workspaceID, sessionID, prompt, atts...)
}

func (k *RemoteKernel) Cancel(sessionID string) {
	ctx := context.Background()
	_ = k.client.CancelAgentSession(ctx, k.workspaceID, sessionID)
}

func (k *RemoteKernel) CancelAll() {
	// The client doesn't support canceling all agents, cancel each session
	ctx := context.Background()
	sessions, err := k.client.ListSessions(ctx, k.workspaceID)
	if err != nil {
		return
	}
	for _, sess := range sessions {
		_ = k.client.CancelAgentSession(ctx, k.workspaceID, sess.ID)
	}
}

func (k *RemoteKernel) IsBusy() bool {
	ctx := context.Background()
	info, err := k.client.GetAgentInfo(ctx, k.workspaceID)
	if err != nil {
		return false
	}
	return info.IsBusy
}

func (k *RemoteKernel) IsSessionBusy(sessionID string) bool {
	// The proto API doesn't support per-session busy check,
	// so we check global busy status
	return k.IsBusy()
}

func (k *RemoteKernel) QueuedPrompts(sessionID string) int {
	ctx := context.Background()
	count, err := k.client.GetAgentSessionQueuedPrompts(ctx, k.workspaceID, sessionID)
	if err != nil {
		return 0
	}
	return count
}

func (k *RemoteKernel) ClearQueue(sessionID string) {
	ctx := context.Background()
	_ = k.client.ClearAgentSessionQueuedPrompts(ctx, k.workspaceID, sessionID)
}

func (k *RemoteKernel) Summarize(ctx context.Context, sessionID string) error {
	return k.client.AgentSummarizeSession(ctx, k.workspaceID, sessionID)
}

// ============================================================================
// Messages
// ============================================================================

func (k *RemoteKernel) ListMessages(ctx context.Context, sessionID string) ([]kernel.Message, error) {
	protoMsgs, err := k.client.ListMessages(ctx, k.workspaceID, sessionID)
	if err != nil {
		return nil, err
	}
	return protoMessagesToKernel(protoMsgs), nil
}

func (k *RemoteKernel) SubscribeMessages(ctx context.Context) <-chan kernel.Event[kernel.Message] {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.messagesCh == nil {
		k.messagesCh = make(chan kernel.Event[kernel.Message], 100)
	}
	return k.messagesCh
}

// ============================================================================
// Sessions
// ============================================================================

func (k *RemoteKernel) CreateSession(ctx context.Context, title string) (kernel.Session, error) {
	sess, err := k.client.CreateSession(ctx, k.workspaceID, title)
	if err != nil {
		return kernel.Session{}, err
	}
	return protoSessionToKernel(*sess), nil
}

func (k *RemoteKernel) GetSession(ctx context.Context, sessionID string) (kernel.Session, error) {
	sess, err := k.client.GetSession(ctx, k.workspaceID, sessionID)
	if err != nil {
		return kernel.Session{}, err
	}
	return protoSessionToKernel(*sess), nil
}

func (k *RemoteKernel) ListSessions(ctx context.Context) ([]kernel.Session, error) {
	sessions, err := k.client.ListSessions(ctx, k.workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]kernel.Session, len(sessions))
	for i, s := range sessions {
		result[i] = protoSessionToKernel(s)
	}
	return result, nil
}

func (k *RemoteKernel) SaveSession(ctx context.Context, session kernel.Session) (kernel.Session, error) {
	protoSess := kernelSessionToProto(session)
	updated, err := k.client.SaveSession(ctx, k.workspaceID, protoSess)
	if err != nil {
		return kernel.Session{}, err
	}
	return protoSessionToKernel(*updated), nil
}

func (k *RemoteKernel) DeleteSession(ctx context.Context, sessionID string) error {
	return k.client.DeleteSession(ctx, k.workspaceID, sessionID)
}

func (k *RemoteKernel) SubscribeSessions(ctx context.Context) <-chan kernel.Event[kernel.Session] {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.sessionsCh == nil {
		k.sessionsCh = make(chan kernel.Event[kernel.Session], 100)
	}
	return k.sessionsCh
}

// ============================================================================
// Permissions
// ============================================================================

func (k *RemoteKernel) PermissionGrant(req kernel.PermissionRequest) {
	k.grantPermission(req, proto.PermissionAllow)
}

func (k *RemoteKernel) PermissionGrantPersistent(req kernel.PermissionRequest) {
	k.grantPermission(req, proto.PermissionAllowForSession)
}

func (k *RemoteKernel) PermissionDeny(req kernel.PermissionRequest) {
	k.grantPermission(req, proto.PermissionDeny)
}

func (k *RemoteKernel) grantPermission(req kernel.PermissionRequest, action proto.PermissionAction) {
	grant := proto.PermissionGrant{
		Permission: kernelPermissionRequestToProto(req),
		Action:     action,
	}
	_ = k.client.GrantPermission(context.Background(), k.workspaceID, grant)
}

func (k *RemoteKernel) SubscribePermissions(ctx context.Context) <-chan kernel.Event[kernel.PermissionRequest] {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.permissionsCh == nil {
		k.permissionsCh = make(chan kernel.Event[kernel.PermissionRequest], 100)
	}
	return k.permissionsCh
}

func (k *RemoteKernel) SubscribePermissionNotifications(ctx context.Context) <-chan kernel.Event[kernel.PermissionNotification] {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.permNotifCh == nil {
		k.permNotifCh = make(chan kernel.Event[kernel.PermissionNotification], 100)
	}
	return k.permNotifCh
}

// ============================================================================
// Agent Notifications
// ============================================================================

func (k *RemoteKernel) SubscribeNotifications(ctx context.Context) <-chan kernel.Event[kernel.Notification] {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.notifsCh == nil {
		k.notifsCh = make(chan kernel.Event[kernel.Notification], 100)
	}
	return k.notifsCh
}

// ============================================================================
// MCP Management
// ============================================================================

func (k *RemoteKernel) GetMCPStates() map[string]kernel.MCPClientInfo {
	ctx := context.Background()
	states, err := k.client.MCPGetStates(ctx, k.workspaceID)
	if err != nil {
		return nil
	}
	result := make(map[string]kernel.MCPClientInfo)
	for name, state := range states {
		result[name] = kernel.MCPClientInfo{
			Name:           state.Name,
			State:          kernel.MCPState(state.State),
			Error:          state.Error,
			ToolCount:      state.ToolCount,
			PromptsCount:   state.PromptCount,
			ResourcesCount: state.ResourceCount,
		}
	}
	return result
}

func (k *RemoteKernel) SubscribeMCP(ctx context.Context) <-chan kernel.Event[kernel.MCPEvent] {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.mcpCh == nil {
		k.mcpCh = make(chan kernel.Event[kernel.MCPEvent], 100)
	}
	return k.mcpCh
}

func (k *RemoteKernel) RefreshMCPTools(ctx context.Context, name string) {
	_ = k.client.RefreshMCPTools(ctx, k.workspaceID, name)
}

func (k *RemoteKernel) RefreshMCPPrompts(ctx context.Context, name string) {
	_ = k.client.MCPRefreshPrompts(ctx, k.workspaceID, name)
}

func (k *RemoteKernel) RefreshMCPResources(ctx context.Context, name string) {
	_ = k.client.MCPRefreshResources(ctx, k.workspaceID, name)
}

func (k *RemoteKernel) GetMCPPrompt(clientID, promptID string, args map[string]string) (string, error) {
	ctx := context.Background()
	return k.client.GetMCPPrompt(ctx, k.workspaceID, clientID, promptID, args)
}

func (k *RemoteKernel) ReadMCPResource(ctx context.Context, clientID, uri string) ([]kernel.MCPResourceContents, error) {
	contents, err := k.client.ReadMCPResource(ctx, k.workspaceID, clientID, uri)
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

func (k *RemoteKernel) EnableDockerMCP(ctx context.Context) error {
	return k.client.EnableDockerMCP(ctx, k.workspaceID)
}

func (k *RemoteKernel) DisableDockerMCP() error {
	ctx := context.Background()
	return k.client.DisableDockerMCP(ctx, k.workspaceID)
}

// ============================================================================
// LSP Events
// ============================================================================

func (k *RemoteKernel) GetLSPStates() map[string]kernel.LSPClientInfo {
	ctx := context.Background()
	states, err := k.client.GetLSPs(ctx, k.workspaceID)
	if err != nil {
		return nil
	}
	result := make(map[string]kernel.LSPClientInfo)
	for name, state := range states {
		result[name] = kernel.LSPClientInfo{
			Name:  state.Name,
			State: kernel.LSPState(state.State),
			Error: state.Error,
			Diagnostics: kernel.DiagnosticCounts{
				Error: state.DiagnosticCount,
			},
		}
	}
	return result
}

func (k *RemoteKernel) SubscribeLSP(ctx context.Context) <-chan kernel.Event[kernel.LSPEvent] {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.lspCh == nil {
		k.lspCh = make(chan kernel.Event[kernel.LSPEvent], 100)
	}
	return k.lspCh
}

// ============================================================================
// File History Events
// ============================================================================

func (k *RemoteKernel) SubscribeHistory(ctx context.Context) <-chan kernel.Event[kernel.HistoryFile] {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.historyCh == nil {
		k.historyCh = make(chan kernel.Event[kernel.HistoryFile], 100)
	}
	return k.historyCh
}

func (k *RemoteKernel) RecordFileRead(ctx context.Context, sessionID, path string) {
	_ = k.client.FileTrackerRecordRead(ctx, k.workspaceID, sessionID, path)
}

func (k *RemoteKernel) LastReadTime(ctx context.Context, sessionID, path string) int64 {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	t, _ := k.client.FileTrackerLastReadTime(ctx, k.workspaceID, sessionID, path)
	return t.Unix()
}

func (k *RemoteKernel) ListReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	return k.client.FileTrackerListReadFiles(ctx, k.workspaceID, sessionID)
}

// ============================================================================
// Session ID Helpers
// ============================================================================

func (k *RemoteKernel) CreateAgentToolSessionID(messageID, toolCallID string) string {
	return messageID + ":" + toolCallID
}

func (k *RemoteKernel) ParseAgentToolSessionID(sessionID string) (messageID, toolCallID string, ok bool) {
	// Simple split on ":"
	for i := 0; i < len(sessionID); i++ {
		if sessionID[i] == ':' {
			return sessionID[:i], sessionID[i+1:], true
		}
	}
	return "", "", false
}

// ============================================================================
// Event Translation
// ============================================================================

// StartEventTranslation starts goroutines that translate proto events from
// the server's SSE stream into kernel events sent to the subscription channels.
func (k *RemoteKernel) StartEventTranslation(ctx context.Context) error {
	k.mu.Lock()
	if k.cancelFunc != nil {
		k.mu.Unlock()
		return nil // Already started
	}
	ctx, k.cancelFunc = context.WithCancel(ctx)
	k.mu.Unlock()

	evc, err := k.client.SubscribeEvents(ctx, k.workspaceID)
	if err != nil {
		return err
	}

	go func() {
		for ev := range evc {
			k.dispatchEvent(ev)
		}
		slog.Info("RemoteKernel event subscription ended")
	}()

	return nil
}

func (k *RemoteKernel) dispatchEvent(ev any) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	switch e := ev.(type) {
	case pubsub.Event[proto.Session]:
		if k.sessionsCh != nil {
			select {
			case k.sessionsCh <- kernel.Event[kernel.Session]{
				Type:    pubsubEventTypeToKernel(e.Type),
				Payload: protoSessionToKernel(e.Payload),
			}:
			default:
				slog.Warn("sessions channel full, dropping event")
			}
		}

	case pubsub.Event[proto.Message]:
		if k.messagesCh != nil {
			select {
			case k.messagesCh <- kernel.Event[kernel.Message]{
				Type:    pubsubEventTypeToKernel(e.Type),
				Payload: protoMessageToKernel(e.Payload),
			}:
			default:
				slog.Warn("messages channel full, dropping event")
			}
		}

	case pubsub.Event[proto.PermissionRequest]:
		if k.permissionsCh != nil {
			select {
			case k.permissionsCh <- kernel.Event[kernel.PermissionRequest]{
				Type:    pubsubEventTypeToKernel(e.Type),
				Payload: protoPermissionRequestToKernel(e.Payload),
			}:
			default:
				slog.Warn("permissions channel full, dropping event")
			}
		}

	case pubsub.Event[proto.PermissionNotification]:
		if k.permNotifCh != nil {
			select {
			case k.permNotifCh <- kernel.Event[kernel.PermissionNotification]{
				Type:    pubsubEventTypeToKernel(e.Type),
				Payload: protoPermissionNotificationToKernel(e.Payload),
			}:
			default:
				slog.Warn("permission notifications channel full, dropping event")
			}
		}

	case pubsub.Event[proto.AgentEvent]:
		if k.notifsCh != nil {
			select {
			case k.notifsCh <- kernel.Event[kernel.Notification]{
				Type:    pubsubEventTypeToKernel(e.Type),
				Payload: protoAgentEventToKernel(e.Payload),
			}:
			default:
				slog.Warn("notifications channel full, dropping event")
			}
		}

	case pubsub.Event[proto.MCPEvent]:
		if k.mcpCh != nil {
			select {
			case k.mcpCh <- kernel.Event[kernel.MCPEvent]{
				Type:    pubsubEventTypeToKernel(e.Type),
				Payload: protoMCPEventToKernel(e.Payload),
			}:
			default:
				slog.Warn("MCP channel full, dropping event")
			}
		}

	case pubsub.Event[proto.LSPEvent]:
		if k.lspCh != nil {
			select {
			case k.lspCh <- kernel.Event[kernel.LSPEvent]{
				Type:    pubsubEventTypeToKernel(e.Type),
				Payload: protoLSPEventToKernel(e.Payload),
			}:
			default:
				slog.Warn("LSP channel full, dropping event")
			}
		}

	case pubsub.Event[proto.File]:
		if k.historyCh != nil {
			select {
			case k.historyCh <- kernel.Event[kernel.HistoryFile]{
				Type:    pubsubEventTypeToKernel(e.Type),
				Payload: protoFileToKernel(e.Payload),
			}:
			default:
				slog.Warn("history channel full, dropping event")
			}
		}
	}
}

// ============================================================================
// Translation Helpers
// ============================================================================

func pubsubEventTypeToKernel(t pubsub.EventType) kernel.EventType {
	switch t {
	case pubsub.CreatedEvent:
		return kernel.EventCreated
	case pubsub.UpdatedEvent:
		return kernel.EventUpdated
	case pubsub.DeletedEvent:
		return kernel.EventDeleted
	default:
		return kernel.EventCreated
	}
}

func protoSessionToKernel(p proto.Session) kernel.Session {
	return kernel.Session{
		ID:               p.ID,
		ParentSessionID:  p.ParentSessionID,
		Title:            p.Title,
		MessageCount:     p.MessageCount,
		PromptTokens:     p.PromptTokens,
		CompletionTokens: p.CompletionTokens,
		Cost:             p.Cost,
		Todos:            nil,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

func kernelSessionToProto(k kernel.Session) proto.Session {
	return proto.Session{
		ID:               k.ID,
		ParentSessionID:  k.ParentSessionID,
		Title:            k.Title,
		MessageCount:     k.MessageCount,
		PromptTokens:     k.PromptTokens,
		CompletionTokens: k.CompletionTokens,
		Cost:             k.Cost,
		CreatedAt:        k.CreatedAt,
		UpdatedAt:        k.UpdatedAt,
	}
}

func protoMessageToKernel(p proto.Message) kernel.Message {
	parts := make([]kernel.ContentPart, 0, len(p.Parts))
	for _, part := range p.Parts {
		switch tp := part.(type) {
		case proto.TextContent:
			parts = append(parts, kernel.TextContent{Text: tp.Text})
		case proto.ReasoningContent:
			parts = append(parts, kernel.ReasoningContent{
				Thinking:   tp.Thinking,
				Signature:  tp.Signature,
				StartedAt:  tp.StartedAt,
				FinishedAt: tp.FinishedAt,
			})
		case proto.ToolCall:
			parts = append(parts, kernel.ToolCallContent{
				ID:               tp.ID,
				Name:             tp.Name,
				Input:            tp.Input,
				ProviderExecuted: false,
				Finished:         tp.Finished,
			})
		case proto.ToolResult:
			parts = append(parts, kernel.ToolResultContent{
				ToolCallID: tp.ToolCallID,
				Name:       tp.Name,
				Content:    tp.Content,
				MIMEType:   "",
				Metadata:   tp.Metadata,
				IsError:    tp.IsError,
			})
		case proto.Finish:
			parts = append(parts, kernel.FinishContent{
				Reason:  kernel.FinishReason(tp.Reason),
				Time:    tp.Time,
				Message: tp.Message,
				Details: tp.Details,
			})
		}
	}
	return kernel.Message{
		ID:               p.ID,
		Role:             kernel.MessageRole(p.Role),
		SessionID:        p.SessionID,
		Parts:            parts,
		Model:            p.Model,
		Provider:         p.Provider,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		IsSummaryMessage: false,
	}
}

func protoMessagesToKernel(p []proto.Message) []kernel.Message {
	result := make([]kernel.Message, len(p))
	for i, msg := range p {
		result[i] = protoMessageToKernel(msg)
	}
	return result
}

func protoPermissionRequestToKernel(p proto.PermissionRequest) kernel.PermissionRequest {
	return kernel.PermissionRequest{
		ID:          p.ID,
		SessionID:   p.SessionID,
		ToolCallID:  p.ToolCallID,
		ToolName:    p.ToolName,
		Description: p.Description,
		Action:      p.Action,
		Path:        p.Path,
	}
}

func kernelPermissionRequestToProto(k kernel.PermissionRequest) proto.PermissionRequest {
	return proto.PermissionRequest{
		ID:          k.ID,
		SessionID:   k.SessionID,
		ToolCallID:  k.ToolCallID,
		ToolName:    k.ToolName,
		Description: k.Description,
		Action:      k.Action,
		Path:        k.Path,
	}
}

func protoPermissionNotificationToKernel(p proto.PermissionNotification) kernel.PermissionNotification {
	return kernel.PermissionNotification{
		ToolCallID: p.ToolCallID,
		Granted:    p.Granted,
		Denied:     p.Denied,
	}
}

func protoAgentEventToKernel(p proto.AgentEvent) kernel.Notification {
	return kernel.Notification{
		SessionID:    p.SessionID,
		SessionTitle: p.SessionTitle,
		Type:         kernel.NotificationType(p.Type),
	}
}

func protoMCPEventToKernel(p proto.MCPEvent) kernel.MCPEvent {
	return kernel.MCPEvent{
		Type:           kernel.MCPEventType(p.Type),
		Name:           p.Name,
		State:          kernel.MCPState(p.State),
		Error:          p.Error,
		ToolCount:      p.ToolCount,
		PromptsCount:   p.PromptCount,
		ResourcesCount: p.ResourceCount,
	}
}

func protoLSPEventToKernel(p proto.LSPEvent) kernel.LSPEvent {
	return kernel.LSPEvent{
		Type: kernel.LSPEventType(p.Type),
		Name: p.Name,
		Diagnostics: kernel.DiagnosticCounts{
			Error: p.DiagnosticCount,
		},
		Error: p.Error,
	}
}

func protoFileToKernel(p proto.File) kernel.HistoryFile {
	return kernel.HistoryFile{
		Path: p.Path,
	}
}
