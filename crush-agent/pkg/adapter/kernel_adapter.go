// Package adapter provides an implementation of the kernel.Kernel interface
// that wraps the internal agent/coordinator engine. This allows the UI
// to remain decoupled from the specific engine implementation.
package adapter

import (
	"context"
	"errors"

	"github.com/qinqd2006/crush/infra/pkg/commands"
	"github.com/qinqd2006/crush/infra/pkg/lsp"
	mcptools "github.com/qinqd2006/crush/infra/pkg/mcp"
	"github.com/qinqd2006/crush/infra/pkg/permission"
	"github.com/qinqd2006/crush/infra/pkg/pubsub"
	"github.com/qinqd2006/crush/kernel/pkg"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/agent"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/agent/notify"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/app"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/config"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/filetracker"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/history"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/message"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/session"
)

var ErrAgentNotFound = errors.New("agent not found")

// Adapter implements the kernel.Kernel interface by wrapping internal packages.
type Adapter struct {
	// Services
	sessions    session.Service
	messages    message.Service
	permissions permission.Service
	history     history.Service
	fileTracker filetracker.Service

	// Agent
	coordinator   agent.Coordinator
	notifications *pubsub.Broker[notify.Notification]

	// Agent registry for multi-agent support.
	agents       map[string]kernel.Agent
	currentAgent string

	// LSP
	lspManager *lsp.Manager

	// Config
	store *config.ConfigStore

	// Subscriptions - using interface{} for now since types differ
	sessionSub   <-chan pubsub.Event[session.Session]
	messageSub   <-chan pubsub.Event[message.Message]
	permSub      <-chan pubsub.Event[permission.PermissionRequest]
	permNotifSub <-chan pubsub.Event[permission.PermissionNotification]
	histSub      <-chan pubsub.Event[history.File]
	mcpSub       <-chan pubsub.Event[mcptools.Event]
	lspSub       <-chan pubsub.Event[app.LSPEvent]
	notifSub     <-chan pubsub.Event[notify.Notification]
}

// NewAdapter creates a new kernel.Adapter wrapping the internal services.
func NewAdapter(
	store *config.ConfigStore,
	sessions session.Service,
	messages message.Service,
	permissions permission.Service,
	history history.Service,
	fileTracker filetracker.Service,
	coordinator agent.Coordinator,
	notifications *pubsub.Broker[notify.Notification],
	lspManager *lsp.Manager,
) *Adapter {
	return &Adapter{
		store:         store,
		sessions:      sessions,
		messages:      messages,
		permissions:   permissions,
		history:       history,
		fileTracker:   fileTracker,
		coordinator:   coordinator,
		notifications: notifications,
		lspManager:    lspManager,
		agents:        make(map[string]kernel.Agent),
		currentAgent:  "coder",
	}
}

// Compile-time check that Adapter implements kernel.Kernel.
var _ kernel.Kernel = (*Adapter)(nil)

// mcpEventTypeToKernel converts an mcptools.EventType (uint) to kernel.MCPEventType (string).
func mcpEventTypeToKernel(t mcptools.EventType) kernel.MCPEventType {
	switch t {
	case mcptools.EventStateChanged:
		return kernel.MCPEventStateChanged
	case mcptools.EventToolsListChanged:
		return kernel.MCPEventToolsListChanged
	case mcptools.EventPromptsListChanged:
		return kernel.MCPEventPromptsListChanged
	case mcptools.EventResourcesListChanged:
		return kernel.MCPEventResourcesListChanged
	default:
		return kernel.MCPEventType("")
	}
}

// ------------------------------------------------------------------------
// Agent Execution
// ------------------------------------------------------------------------

func (a *Adapter) RunPrompt(ctx context.Context, sessionID, prompt string, attachments ...kernel.Attachment) error {
	var msgAttachments []message.Attachment
	for _, att := range attachments {
		msgAttachments = append(msgAttachments, message.Attachment{
			FilePath: att.FilePath,
			FileName: att.FileName,
			MimeType: att.MimeType,
			Content:  att.Content,
		})
	}
	_, err := a.coordinator.Run(ctx, sessionID, prompt, msgAttachments...)
	return err
}

func (a *Adapter) Cancel(sessionID string) {
	a.coordinator.Cancel(sessionID)
}

func (a *Adapter) AgentCancel(sessionID string) {
	a.coordinator.Cancel(sessionID)
}

func (a *Adapter) CancelAll() {
	a.coordinator.CancelAll()
}

func (a *Adapter) IsBusy() bool {
	return a.coordinator.IsBusy()
}

func (a *Adapter) AgentIsReady() bool {
	return a.coordinator.IsBusy()
}

func (a *Adapter) AgentIsBusy() bool {
	return a.coordinator.IsBusy()
}

func (a *Adapter) IsSessionBusy(sessionID string) bool {
	return a.coordinator.IsSessionBusy(sessionID)
}

func (a *Adapter) QueuedPrompts(sessionID string) int {
	return a.coordinator.QueuedPrompts(sessionID)
}

func (a *Adapter) AgentQueuedPrompts(sessionID string) int {
	return a.coordinator.QueuedPrompts(sessionID)
}

func (a *Adapter) QueuedPromptsList(sessionID string) []string {
	return a.coordinator.QueuedPromptsList(sessionID)
}

func (a *Adapter) AgentQueuedPromptsList(sessionID string) []string {
	return a.coordinator.QueuedPromptsList(sessionID)
}

func (a *Adapter) ClearQueue(sessionID string) {
	a.coordinator.ClearQueue(sessionID)
}

func (a *Adapter) AgentClearQueue(sessionID string) {
	a.coordinator.ClearQueue(sessionID)
}

func (a *Adapter) Summarize(ctx context.Context, sessionID string) error {
	return a.coordinator.Summarize(ctx, sessionID)
}

func (a *Adapter) AgentSummarize(ctx context.Context, sessionID string) error {
	return a.coordinator.Summarize(ctx, sessionID)
}

// ------------------------------------------------------------------------
// Messages
// ------------------------------------------------------------------------

func (a *Adapter) ListMessages(ctx context.Context, sessionID string) ([]kernel.Message, error) {
	msgs, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return a.messagesToKernel(msgs), nil
}

func (a *Adapter) SubscribeMessages(ctx context.Context) <-chan kernel.Event[kernel.Message] {
	ch := make(chan kernel.Event[kernel.Message], 100)
	if a.messageSub == nil {
		a.messageSub = a.messages.Subscribe(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case event, ok := <-a.messageSub:
				if !ok {
					close(ch)
					return
				}
				for _, msg := range a.messagesToKernel([]message.Message{event.Payload}) {
					ch <- kernel.Event[kernel.Message]{
						Type:    kernel.EventType(event.Type),
						Payload: msg,
					}
				}
			}
		}
	}()
	return ch
}

func (a *Adapter) messagesToKernel(msgs []message.Message) []kernel.Message {
	result := make([]kernel.Message, len(msgs))
	for i, m := range msgs {
		result[i] = a.messageToKernel(&m)
	}
	return result
}

func (a *Adapter) messageToKernel(m *message.Message) kernel.Message {
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

func (a *Adapter) CreateSession(ctx context.Context, title string) (kernel.Session, error) {
	sess, err := a.sessions.Create(ctx, title)
	if err != nil {
		return kernel.Session{}, err
	}
	return a.sessionToKernel(&sess), nil
}

func (a *Adapter) GetSession(ctx context.Context, sessionID string) (kernel.Session, error) {
	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return kernel.Session{}, err
	}
	return a.sessionToKernel(&sess), nil
}

func (a *Adapter) ListSessions(ctx context.Context) ([]kernel.Session, error) {
	sessions, err := a.sessions.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]kernel.Session, len(sessions))
	for i, s := range sessions {
		result[i] = a.sessionToKernel(&s)
	}
	return result, nil
}

func (a *Adapter) SaveSession(ctx context.Context, sess kernel.Session) (kernel.Session, error) {
	s := a.kernelToSession(&sess)
	saved, err := a.sessions.Save(ctx, s)
	if err != nil {
		return kernel.Session{}, err
	}
	return a.sessionToKernel(&saved), nil
}

func (a *Adapter) DeleteSession(ctx context.Context, sessionID string) error {
	return a.sessions.Delete(ctx, sessionID)
}

func (a *Adapter) SubscribeSessions(ctx context.Context) <-chan kernel.Event[kernel.Session] {
	ch := make(chan kernel.Event[kernel.Session], 100)
	if a.sessionSub == nil {
		a.sessionSub = a.sessions.Subscribe(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case event, ok := <-a.sessionSub:
				if !ok {
					close(ch)
					return
				}
				ch <- kernel.Event[kernel.Session]{
					Type:    kernel.EventType(event.Type),
					Payload: a.sessionToKernel(&event.Payload),
				}
			}
		}
	}()
	return ch
}

func (a *Adapter) sessionToKernel(s *session.Session) kernel.Session {
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

func (a *Adapter) kernelToSession(s *kernel.Session) session.Session {
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

func (a *Adapter) PermissionGrant(req kernel.PermissionRequest) {
	a.permissions.Grant(a.kernelToPermissionReq(&req))
}

func (a *Adapter) PermissionGrantPersistent(req kernel.PermissionRequest) {
	a.permissions.GrantPersistent(a.kernelToPermissionReq(&req))
}

func (a *Adapter) PermissionDeny(req kernel.PermissionRequest) {
	a.permissions.Deny(a.kernelToPermissionReq(&req))
}

func (a *Adapter) SubscribePermissions(ctx context.Context) <-chan kernel.Event[kernel.PermissionRequest] {
	ch := make(chan kernel.Event[kernel.PermissionRequest], 100)
	if a.permSub == nil {
		a.permSub = a.permissions.Subscribe(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case event, ok := <-a.permSub:
				if !ok {
					close(ch)
					return
				}
				ch <- kernel.Event[kernel.PermissionRequest]{
					Type:    kernel.EventType(event.Type),
					Payload: a.permissionToKernel(&event.Payload),
				}
			}
		}
	}()
	return ch
}

func (a *Adapter) SubscribePermissionNotifications(ctx context.Context) <-chan kernel.Event[kernel.PermissionNotification] {
	ch := make(chan kernel.Event[kernel.PermissionNotification], 100)
	if a.permNotifSub == nil {
		a.permNotifSub = a.permissions.SubscribeNotifications(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case event, ok := <-a.permNotifSub:
				if !ok {
					close(ch)
					return
				}
				ch <- kernel.Event[kernel.PermissionNotification]{
					Type:    kernel.EventType(event.Type),
					Payload: a.permissionNotifToKernel(&event.Payload),
				}
			}
		}
	}()
	return ch
}

func (a *Adapter) kernelToPermissionReq(req *kernel.PermissionRequest) permission.PermissionRequest {
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

func (a *Adapter) permissionToKernel(req *permission.PermissionRequest) kernel.PermissionRequest {
	return kernel.PermissionRequest{
		ID:          req.ID,
		SessionID:   req.SessionID,
		ToolCallID:  req.ToolCallID,
		ToolName:    req.ToolName,
		Description: req.Description,
		Action:      req.Action,
		Path:        req.Path,
	}
}

func (a *Adapter) permissionNotifToKernel(n *permission.PermissionNotification) kernel.PermissionNotification {
	return kernel.PermissionNotification{
		ToolCallID: n.ToolCallID,
		Granted:    n.Granted,
		Denied:     n.Denied,
	}
}

// ------------------------------------------------------------------------
// Agent Notifications
// ------------------------------------------------------------------------

func (a *Adapter) SubscribeNotifications(ctx context.Context) <-chan kernel.Event[kernel.Notification] {
	ch := make(chan kernel.Event[kernel.Notification], 100)
	if a.notifSub == nil {
		a.notifSub = a.notifications.Subscribe(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case event, ok := <-a.notifSub:
				if !ok {
					close(ch)
					return
				}
				ch <- kernel.Event[kernel.Notification]{
					Type: kernel.EventType(event.Type),
					Payload: kernel.Notification{
						SessionID:    event.Payload.SessionID,
						SessionTitle: event.Payload.SessionTitle,
						Type:         kernel.NotificationType(event.Payload.Type),
						ProviderID:   event.Payload.ProviderID,
					},
				}
			}
		}
	}()
	return ch
}

// ------------------------------------------------------------------------
// MCP Management
// ------------------------------------------------------------------------

func (a *Adapter) GetMCPStates() map[string]kernel.MCPClientInfo {
	states := mcptools.GetStates()
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

func (a *Adapter) SubscribeMCP(ctx context.Context) <-chan kernel.Event[kernel.MCPEvent] {
	ch := make(chan kernel.Event[kernel.MCPEvent], 100)
	if a.mcpSub == nil {
		a.mcpSub = mcptools.SubscribeEvents(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case event, ok := <-a.mcpSub:
				if !ok {
					close(ch)
					return
				}
				ch <- kernel.Event[kernel.MCPEvent]{
					Type: kernel.EventType(event.Type),
					Payload: kernel.MCPEvent{
						Type:           mcpEventTypeToKernel(event.Payload.Type),
						Name:           event.Payload.Name,
						State:          kernel.MCPState(event.Payload.State),
						Error:          event.Payload.Error,
						ToolCount:      event.Payload.Counts.Tools,
						PromptsCount:   event.Payload.Counts.Prompts,
						ResourcesCount: event.Payload.Counts.Resources,
					},
				}
			}
		}
	}()
	return ch
}

func (a *Adapter) RefreshMCPTools(ctx context.Context, name string) {
	mcptools.RefreshTools(ctx, a.store, name)
}

func (a *Adapter) RefreshMCPPrompts(ctx context.Context, name string) {
	mcptools.RefreshPrompts(ctx, name)
}

func (a *Adapter) RefreshMCPResources(ctx context.Context, name string) {
	mcptools.RefreshResources(ctx, name)
}

func (a *Adapter) GetMCPPrompt(clientID, promptID string, args map[string]string) (string, error) {
	return commands.GetMCPPrompt(a.store, clientID, promptID, args)
}

func (a *Adapter) ReadMCPResource(ctx context.Context, clientID, uri string) ([]kernel.MCPResourceContents, error) {
	contents, err := mcptools.ReadResource(ctx, a.store, clientID, uri)
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

func (a *Adapter) EnableDockerMCP(ctx context.Context) error {
	mcpConfig, err := a.store.PrepareDockerMCPConfig()
	if err != nil {
		return err
	}
	if err := mcptools.InitializeSingle(ctx, config.DockerMCPName, a.store); err != nil {
		disableErr := mcptools.DisableSingle(a.store, config.DockerMCPName)
		delete(a.store.Config().MCP, config.DockerMCPName)
		return errors.Join(err, disableErr)
	}
	if err := a.store.PersistDockerMCPConfig(mcpConfig); err != nil {
		disableErr := mcptools.DisableSingle(a.store, config.DockerMCPName)
		delete(a.store.Config().MCP, config.DockerMCPName)
		return errors.Join(err, disableErr)
	}
	return nil
}

func (a *Adapter) DisableDockerMCP() error {
	if err := mcptools.DisableSingle(a.store, config.DockerMCPName); err != nil {
		return err
	}
	return a.store.DisableDockerMCP()
}

// ------------------------------------------------------------------------
// LSP Events
// ------------------------------------------------------------------------

func (a *Adapter) GetLSPStates() map[string]kernel.LSPClientInfo {
	states := app.GetLSPStates()
	result := make(map[string]kernel.LSPClientInfo, len(states))
	for name, s := range states {
		var diagnostics kernel.DiagnosticCounts
		if s.Client != nil {
			counts := s.Client.GetDiagnosticCounts()
			diagnostics = kernel.DiagnosticCounts{
				Error:       counts.Error,
				Warning:     counts.Warning,
				Information: counts.Information,
				Hint:        counts.Hint,
			}
		}
		result[name] = kernel.LSPClientInfo{
			Name:        s.Name,
			State:       kernel.LSPState(s.State),
			Error:       s.Error,
			Diagnostics: diagnostics,
		}
	}
	return result
}

func (a *Adapter) SubscribeLSP(ctx context.Context) <-chan kernel.Event[kernel.LSPEvent] {
	ch := make(chan kernel.Event[kernel.LSPEvent], 100)
	if a.lspSub == nil {
		a.lspSub = app.SubscribeLSPEvents(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case event, ok := <-a.lspSub:
				if !ok {
					close(ch)
					return
				}
				ch <- kernel.Event[kernel.LSPEvent]{
					Type: kernel.EventType(event.Type),
					Payload: kernel.LSPEvent{
						Type: kernel.LSPEventType(event.Payload.Type),
						Name: event.Payload.Name,
						Diagnostics: kernel.DiagnosticCounts{
							Error: event.Payload.DiagnosticCount,
						},
						Error: event.Payload.Error,
					},
				}
			}
		}
	}()
	return ch
}

// ------------------------------------------------------------------------
// File History Events
// ------------------------------------------------------------------------

func (a *Adapter) SubscribeHistory(ctx context.Context) <-chan kernel.Event[kernel.HistoryFile] {
	ch := make(chan kernel.Event[kernel.HistoryFile], 100)
	if a.histSub == nil {
		a.histSub = a.history.Subscribe(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case event, ok := <-a.histSub:
				if !ok {
					close(ch)
					return
				}
				ch <- kernel.Event[kernel.HistoryFile]{
					Type: kernel.EventType(event.Type),
					Payload: kernel.HistoryFile{
						Path: event.Payload.Path,
					},
				}
			}
		}
	}()
	return ch
}

func (a *Adapter) RecordFileRead(ctx context.Context, sessionID, path string) {
	a.fileTracker.RecordRead(ctx, sessionID, path)
}

func (a *Adapter) LastReadTime(ctx context.Context, sessionID, path string) int64 {
	return a.fileTracker.LastReadTime(ctx, sessionID, path).Unix()
}

func (a *Adapter) ListReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	return a.fileTracker.ListReadFiles(ctx, sessionID)
}

// ------------------------------------------------------------------------
// Session ID Helpers
// ------------------------------------------------------------------------

func (a *Adapter) CreateAgentToolSessionID(messageID, toolCallID string) string {
	return a.sessions.CreateAgentToolSessionID(messageID, toolCallID)
}

func (a *Adapter) ParseAgentToolSessionID(sessionID string) (messageID string, toolCallID string, ok bool) {
	return a.sessions.ParseAgentToolSessionID(sessionID)
}

// ------------------------------------------------------------------------
// Lifecycle
// ------------------------------------------------------------------------

func (a *Adapter) Shutdown() {
	a.lspManager.KillAll(context.Background())
}

// ------------------------------------------------------------------------
// Configuration
// ------------------------------------------------------------------------

func (a *Adapter) Config() kernel.ConfigProvider {
	return &ConfigAdapter{store: a.store}
}

// ------------------------------------------------------------------------
// Agent Management
// ------------------------------------------------------------------------

// RegisterAgent registers an agent with the adapter.
// This is called during initialization to set up all agents.
func (a *Adapter) RegisterAgent(agent kernel.Agent) {
	a.agents[agent.Name()] = agent
}

// ListAgents implements kernel.Kernel.
func (a *Adapter) ListAgents() []string {
	names := make([]string, 0, len(a.agents))
	for name := range a.agents {
		names = append(names, name)
	}
	return names
}

// GetActiveAgent implements kernel.Kernel.
func (a *Adapter) GetActiveAgent() string {
	return a.currentAgent
}

// SetActiveAgent implements kernel.Kernel.
func (a *Adapter) SetActiveAgent(name string) error {
	if _, ok := a.agents[name]; !ok {
		return ErrAgentNotFound
	}
	a.currentAgent = name
	return nil
}

// GetAgent implements kernel.Kernel.
func (a *Adapter) GetAgent(name string) kernel.Agent {
	return a.agents[name]
}

// activeAgent returns the currently active kernel.Agent.
func (a *Adapter) activeAgent() kernel.Agent {
	return a.agents[a.currentAgent]
}
