// Package remote provides a kernel.Kernel implementation for client/server mode.
// It wraps the client.Client and translates proto events to kernel events.
package remote

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/mosaic2025002/crush/engine/client"
	"github.com/mosaic2025002/crush/engine/message"
	"github.com/mosaic2025002/crush/engine/proto"
	"github.com/mosaic2025002/crush/engine/pubsub"
	"github.com/mosaic2025002/crush/kernel"
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
			Name:            state.Name,
			State:           kernel.LSPState(state.State),
			Error:           state.Error,
			DiagnosticCount: state.DiagnosticCount,
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
		Type:            kernel.LSPEventType(p.Type),
		Name:            p.Name,
		DiagnosticCount: p.DiagnosticCount,
		Error:           p.Error,
	}
}

func protoFileToKernel(p proto.File) kernel.HistoryFile {
	return kernel.HistoryFile{
		Path: p.Path,
	}
}
