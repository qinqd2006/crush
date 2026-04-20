package model

import (
	"context"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/qinqd2006/crush/crush-agent/pkg/config"
	"github.com/qinqd2006/crush/infra/pkg/csync"
	"github.com/qinqd2006/crush/kernel/pkg"
	"github.com/qinqd2006/crush/ui/pkg/common"
	"github.com/stretchr/testify/require"
)

func TestCurrentModelSupportsImages(t *testing.T) {
	t.Parallel()

	t.Run("returns false when config is nil", func(t *testing.T) {
		t.Parallel()

		ui := newTestUIWithConfig(t, nil)
		require.False(t, ui.currentModelSupportsImages())
	})

	t.Run("returns false when coder agent is missing", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			Providers: csync.NewMap[string, config.ProviderConfig](),
			Agents:    map[string]config.Agent{},
		}
		ui := newTestUIWithConfig(t, cfg)
		require.False(t, ui.currentModelSupportsImages())
	})

	t.Run("returns false when model is not found", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			Providers: csync.NewMap[string, config.ProviderConfig](),
			Agents: map[string]config.Agent{
				config.AgentCoder: {Model: config.SelectedModelTypeLarge},
			},
		}
		ui := newTestUIWithConfig(t, cfg)
		require.False(t, ui.currentModelSupportsImages())
	})

	t.Run("returns true when current model supports images", func(t *testing.T) {
		t.Parallel()

		providers := csync.NewMap[string, config.ProviderConfig]()
		providers.Set("test-provider", config.ProviderConfig{
			ID: "test-provider",
			Models: []catwalk.Model{
				{ID: "test-model", SupportsImages: true},
			},
		})

		cfg := &config.Config{
			Models: map[config.SelectedModelType]config.SelectedModel{
				config.SelectedModelTypeLarge: {
					Provider: "test-provider",
					Model:    "test-model",
				},
			},
			Providers: providers,
			Agents: map[string]config.Agent{
				config.AgentCoder: {Model: config.SelectedModelTypeLarge},
			},
		}

		ui := newTestUIWithConfig(t, cfg)
		require.True(t, ui.currentModelSupportsImages())
	})
}

func newTestUIWithConfig(t *testing.T, cfg *config.Config) *UI {
	t.Helper()

	return &UI{
		com: &common.Common{
			Workspace: &testWorkspaceInterface{cfg: cfg, configProvider: newTestConfigProvider(cfg)},
		},
	}
}

// testConfigProvider implements kernel.ConfigProvider for testing.
type testConfigProvider struct {
	cfg *config.Config
}

func newTestConfigProvider(cfg *config.Config) *testConfigProvider {
	return &testConfigProvider{cfg: cfg}
}

func (p *testConfigProvider) GetCoderAgent() kernel.AgentView {
	if p.cfg == nil {
		return kernel.AgentView{}
	}
	agentCfg, ok := p.cfg.Agents[config.AgentCoder]
	if !ok {
		return kernel.AgentView{}
	}
	selectedModel, selOk := p.cfg.Models[agentCfg.Model]
	if !selOk {
		return kernel.AgentView{Role: "coder", Name: agentCfg.Name}
	}
	model := p.cfg.GetModel(selectedModel.Provider, selectedModel.Model)
	if model == nil {
		return kernel.AgentView{
			Role:         "coder",
			Name:         agentCfg.Name,
			CurrentModel: kernel.ModelView{Type: kernel.SelectedModelType(agentCfg.Model)},
		}
	}
	return kernel.AgentView{
		Role: "coder",
		Name: agentCfg.Name,
		CurrentModel: kernel.ModelView{
			Type:          kernel.SelectedModelType(agentCfg.Model),
			Name:          model.Name,
			ProviderID:    selectedModel.Provider,
			ModelID:       selectedModel.Model,
			ContextWindow: model.ContextWindow,
			SupportsImage: model.SupportsImages,
		},
	}
}

func (p *testConfigProvider) GetSmallModel() (kernel.ModelView, bool) {
	if p.cfg == nil {
		return kernel.ModelView{}, false
	}
	selectedModel, ok := p.cfg.Models[config.SelectedModelTypeSmall]
	if !ok {
		return kernel.ModelView{}, false
	}
	model := p.cfg.GetModel(selectedModel.Provider, selectedModel.Model)
	if model == nil {
		return kernel.ModelView{
			Type:       kernel.SelectedModelTypeSmall,
			ProviderID: selectedModel.Provider,
			ModelID:    selectedModel.Model,
		}, true
	}
	return kernel.ModelView{
		Type:          kernel.SelectedModelTypeSmall,
		Name:          model.Name,
		ProviderID:    selectedModel.Provider,
		ModelID:       selectedModel.Model,
		ContextWindow: model.ContextWindow,
		SupportsImage: model.SupportsImages,
	}, true
}

func (p *testConfigProvider) GetModelViewByType(modelType kernel.SelectedModelType) (kernel.ModelView, bool) {
	if p.cfg == nil {
		return kernel.ModelView{}, false
	}
	selectedModel, ok := p.cfg.Models[config.SelectedModelType(modelType)]
	if !ok {
		return kernel.ModelView{}, false
	}
	model := p.cfg.GetModel(selectedModel.Provider, selectedModel.Model)
	if model == nil {
		return kernel.ModelView{
			Type:       modelType,
			ProviderID: selectedModel.Provider,
			ModelID:    selectedModel.Model,
		}, true
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

// Unused ConfigProvider methods - not called by currentModelSupportsImages
func (p *testConfigProvider) Get() *kernel.Config { return nil }
func (p *testConfigProvider) IsConfigured() bool  { return p.cfg != nil && p.cfg.IsConfigured() }
func (p *testConfigProvider) GetPreferredModel(scope string) kernel.ModelConfig {
	return kernel.ModelConfig{}
}
func (p *testConfigProvider) SetPreferredModel(scope string, model kernel.ModelConfig) error {
	return nil
}
func (p *testConfigProvider) GetModelInfo(providerID, modelID string) kernel.ModelInfo {
	return kernel.ModelInfo{}
}
func (p *testConfigProvider) GetProviderInfo(providerID string) kernel.ProviderInfo {
	return kernel.ProviderInfo{}
}
func (p *testConfigProvider) Providers() ([]kernel.Provider, error) { return nil, nil }
func (p *testConfigProvider) GetAgent(name string) (kernel.AgentConfig, bool) {
	return kernel.AgentConfig{}, false
}
func (p *testConfigProvider) GetSelectedModel(modelType kernel.SelectedModelType) (kernel.SelectedModel, bool) {
	return kernel.SelectedModel{}, false
}
func (p *testConfigProvider) GetModel(providerID, modelID string) (kernel.Model, bool) {
	return kernel.Model{}, false
}
func (p *testConfigProvider) SetConfigField(scope kernel.Scope, key string, value any) error {
	return nil
}
func (p *testConfigProvider) GetGlobalOptions() kernel.GlobalOptions {
	if p.cfg == nil || p.cfg.Options == nil {
		return kernel.GlobalOptions{}
	}
	progress := true
	if p.cfg.Options.Progress != nil {
		progress = *p.cfg.Options.Progress
	}
	return kernel.GlobalOptions{
		DisableNotifications: p.cfg.Options.DisableNotifications,
		InitializeAs:         p.cfg.Options.InitializeAs,
		Progress:             progress,
	}
}
func (p *testConfigProvider) TUIOptions() kernel.TUIOptions {
	if p.cfg == nil || p.cfg.Options == nil || p.cfg.Options.TUI == nil {
		return kernel.TUIOptions{}
	}
	tuiOpts := p.cfg.Options.TUI
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
func (p *testConfigProvider) GetProvider(providerID string) (kernel.Provider, bool) {
	if p.cfg == nil {
		return kernel.Provider{}, false
	}
	providerCfg, ok := p.cfg.Providers.Get(providerID)
	if !ok {
		return kernel.Provider{}, false
	}
	return kernel.Provider{
		ID:   providerCfg.ID,
		Name: providerCfg.Name,
		Type: string(providerCfg.Type),
	}, true
}
func (p *testConfigProvider) ListMCPServers() []kernel.MCPServerInfo {
	if p.cfg == nil {
		return nil
	}
	mcps := p.cfg.MCP.Sorted()
	servers := make([]kernel.MCPServerInfo, len(mcps))
	for i, mcp := range mcps {
		servers[i] = kernel.MCPServerInfo{
			Name: mcp.Name,
			Type: string(mcp.MCP.Type),
		}
	}
	return servers
}
func (p *testConfigProvider) ListCustomCommands() []kernel.CustomCommandInfo { return nil }

// testWorkspaceInterface is a minimal workspace stub implementing kernel.Workspace for unit tests.
type testWorkspaceInterface struct {
	cfg            *config.Config
	configProvider *testConfigProvider
}

func (w *testWorkspaceInterface) EngineConfig() any {
	return w.cfg
}

// Minimal implementations of kernel.Workspace methods - these are not called by currentModelSupportsImages
func (w *testWorkspaceInterface) Name() string        { return "" }
func (w *testWorkspaceInterface) Description() string { return "" }
func (w *testWorkspaceInterface) RunPrompt(context.Context, string, string, ...kernel.Attachment) error {
	return nil
}
func (w *testWorkspaceInterface) Cancel(string)                                {}
func (w *testWorkspaceInterface) CancelAll()                                   {}
func (w *testWorkspaceInterface) IsBusy() bool                                 { return false }
func (w *testWorkspaceInterface) IsSessionBusy(string) bool                    { return false }
func (w *testWorkspaceInterface) AgentIsReady() bool                           { return false }
func (w *testWorkspaceInterface) AgentIsBusy() bool                            { return false }
func (w *testWorkspaceInterface) AgentIsSessionBusy(string) bool               { return false }
func (w *testWorkspaceInterface) QueuedPrompts(string) int                     { return 0 }
func (w *testWorkspaceInterface) AgentQueuedPrompts(string) int                { return 0 }
func (w *testWorkspaceInterface) QueuedPromptsList(string) []string            { return nil }
func (w *testWorkspaceInterface) AgentQueuedPromptsList(string) []string       { return nil }
func (w *testWorkspaceInterface) AgentCancel(string)                           {}
func (w *testWorkspaceInterface) ClearQueue(string)                            {}
func (w *testWorkspaceInterface) AgentClearQueue(string)                       {}
func (w *testWorkspaceInterface) Summarize(context.Context, string) error      { return nil }
func (w *testWorkspaceInterface) AgentSummarize(context.Context, string) error { return nil }
func (w *testWorkspaceInterface) ListMessages(context.Context, string) ([]kernel.Message, error) {
	return nil, nil
}
func (w *testWorkspaceInterface) ListUserMessages(context.Context, string) ([]kernel.Message, error) {
	return nil, nil
}
func (w *testWorkspaceInterface) ListAllUserMessages(context.Context) ([]kernel.Message, error) {
	return nil, nil
}
func (w *testWorkspaceInterface) SubscribeMessages(context.Context) <-chan kernel.Event[kernel.Message] {
	return nil
}
func (w *testWorkspaceInterface) CreateSession(context.Context, string) (kernel.Session, error) {
	return kernel.Session{}, nil
}
func (w *testWorkspaceInterface) GetSession(context.Context, string) (kernel.Session, error) {
	return kernel.Session{}, nil
}
func (w *testWorkspaceInterface) ListSessions(context.Context) ([]kernel.Session, error) {
	return nil, nil
}
func (w *testWorkspaceInterface) SaveSession(context.Context, kernel.Session) (kernel.Session, error) {
	return kernel.Session{}, nil
}
func (w *testWorkspaceInterface) DeleteSession(context.Context, string) error { return nil }
func (w *testWorkspaceInterface) SubscribeSessions(context.Context) <-chan kernel.Event[kernel.Session] {
	return nil
}
func (w *testWorkspaceInterface) PermissionGrant(kernel.PermissionRequest)           {}
func (w *testWorkspaceInterface) PermissionGrantPersistent(kernel.PermissionRequest) {}
func (w *testWorkspaceInterface) PermissionDeny(kernel.PermissionRequest)            {}
func (w *testWorkspaceInterface) PermissionSkipRequests() bool                       { return false }
func (w *testWorkspaceInterface) PermissionSetSkipRequests(bool)                     {}
func (w *testWorkspaceInterface) SubscribePermissions(context.Context) <-chan kernel.Event[kernel.PermissionRequest] {
	return nil
}
func (w *testWorkspaceInterface) SubscribePermissionNotifications(context.Context) <-chan kernel.Event[kernel.PermissionNotification] {
	return nil
}
func (w *testWorkspaceInterface) SubscribeNotifications(context.Context) <-chan kernel.Event[kernel.Notification] {
	return nil
}
func (w *testWorkspaceInterface) GetMCPStates() map[string]kernel.MCPClientInfo { return nil }
func (w *testWorkspaceInterface) SubscribeMCP(context.Context) <-chan kernel.Event[kernel.MCPEvent] {
	return nil
}
func (w *testWorkspaceInterface) RefreshMCPTools(context.Context, string)     {}
func (w *testWorkspaceInterface) RefreshMCPPrompts(context.Context, string)   {}
func (w *testWorkspaceInterface) RefreshMCPResources(context.Context, string) {}
func (w *testWorkspaceInterface) GetMCPPrompt(string, string, map[string]string) (string, error) {
	return "", nil
}
func (w *testWorkspaceInterface) ReadMCPResource(context.Context, string, string) ([]kernel.MCPResourceContents, error) {
	return nil, nil
}
func (w *testWorkspaceInterface) EnableDockerMCP(context.Context) error         { return nil }
func (w *testWorkspaceInterface) DisableDockerMCP() error                       { return nil }
func (w *testWorkspaceInterface) LSPStart(context.Context, string)              {}
func (w *testWorkspaceInterface) LSPStopAll(context.Context)                    {}
func (w *testWorkspaceInterface) GetLSPStates() map[string]kernel.LSPClientInfo { return nil }
func (w *testWorkspaceInterface) LSPGetStates() map[string]kernel.LSPClientInfo { return nil }
func (w *testWorkspaceInterface) SubscribeLSP(context.Context) <-chan kernel.Event[kernel.LSPEvent] {
	return nil
}
func (w *testWorkspaceInterface) SubscribeHistory(context.Context) <-chan kernel.Event[kernel.HistoryFile] {
	return nil
}
func (w *testWorkspaceInterface) RecordFileRead(context.Context, string, string)        {}
func (w *testWorkspaceInterface) FileTrackerRecordRead(context.Context, string, string) {}
func (w *testWorkspaceInterface) LastReadTime(context.Context, string, string) int64    { return 0 }
func (w *testWorkspaceInterface) FileTrackerLastReadTime(context.Context, string, string) int64 {
	return 0
}
func (w *testWorkspaceInterface) ListReadFiles(context.Context, string) ([]string, error) {
	return nil, nil
}
func (w *testWorkspaceInterface) FileTrackerListReadFiles(context.Context, string) ([]string, error) {
	return nil, nil
}
func (w *testWorkspaceInterface) ListSessionHistory(context.Context, string) ([]kernel.HistoryFile, error) {
	return nil, nil
}
func (w *testWorkspaceInterface) CreateAgentToolSessionID(string, string) string { return "" }
func (w *testWorkspaceInterface) ParseAgentToolSessionID(string) (string, string, bool) {
	return "", "", false
}
func (w *testWorkspaceInterface) GetModel() kernel.ModelConfig                       { return kernel.ModelConfig{} }
func (w *testWorkspaceInterface) AgentModel() kernel.AgentModel                      { return kernel.AgentModel{} }
func (w *testWorkspaceInterface) SetModel(context.Context, kernel.ModelConfig) error { return nil }
func (w *testWorkspaceInterface) GetDefaultSmallModel(string) kernel.ModelConfig {
	return kernel.ModelConfig{}
}
func (w *testWorkspaceInterface) InitAgent(context.Context) error        { return nil }
func (w *testWorkspaceInterface) InitCoderAgent(context.Context) error   { return nil }
func (w *testWorkspaceInterface) UpdateAgentModel(context.Context) error { return nil }
func (w *testWorkspaceInterface) Config() kernel.ConfigProvider          { return w.configProvider }
func (w *testWorkspaceInterface) WorkingDir() string                     { return "" }
func (w *testWorkspaceInterface) Resolver() kernel.VariableResolver      { return nil }
func (w *testWorkspaceInterface) UpdatePreferredModel(kernel.Scope, kernel.SelectedModelType, kernel.SelectedModel) error {
	return nil
}
func (w *testWorkspaceInterface) SetCompactMode(kernel.Scope, bool) error              { return nil }
func (w *testWorkspaceInterface) SetProviderAPIKey(kernel.Scope, string, any) error    { return nil }
func (w *testWorkspaceInterface) TestProviderConnection(kernel.Provider, string) error { return nil }
func (w *testWorkspaceInterface) SetConfigField(kernel.Scope, string, any) error       { return nil }
func (w *testWorkspaceInterface) RemoveConfigField(kernel.Scope, string) error         { return nil }
func (w *testWorkspaceInterface) ImportCopilot() (*kernel.OAuthToken, bool)            { return nil, false }
func (w *testWorkspaceInterface) RefreshOAuthToken(context.Context, kernel.Scope, string) error {
	return nil
}
func (w *testWorkspaceInterface) GetDockerMCPAvailability() (bool, bool)    { return false, false }
func (w *testWorkspaceInterface) RefreshDockerMCPAvailability() bool        { return false }
func (w *testWorkspaceInterface) GlobalConfigPath() string                  { return "" }
func (w *testWorkspaceInterface) ProjectNeedsInitialization() (bool, error) { return false, nil }
func (w *testWorkspaceInterface) MarkProjectInitialized() error             { return nil }
func (w *testWorkspaceInterface) InitializePrompt() (string, error)         { return "", nil }
func (w *testWorkspaceInterface) Subscribe(kernel.Program)                  {}
func (w *testWorkspaceInterface) KernelSubscribe(kernel.Program)            {}
func (w *testWorkspaceInterface) Shutdown()                                 {}
func (w *testWorkspaceInterface) ResetCache()                               {}
