package adapter

import (
	"context"
	"log/slog"
	"sync"

	"github.com/qinqd2006/crush/kernel/pkg"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/agent"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/message"
)

// LocalAgentAdapter wraps a SessionAgent and implements the kernel.Agent interface.
// It converts SessionAgent's blocking Run() into kernel.Agent's async streaming pattern
// by running the agent in a goroutine and subscribing to message updates.
type LocalAgentAdapter struct {
	name         string
	description  string
	sessionAgent agent.SessionAgent
	messages     message.Service
	systemPrompt string
}

var _ kernel.Agent = (*LocalAgentAdapter)(nil)

// NewLocalAgentAdapter creates a new LocalAgentAdapter wrapping the given SessionAgent.
func NewLocalAgentAdapter(name, description string, sa agent.SessionAgent, messages message.Service) *LocalAgentAdapter {
	return &LocalAgentAdapter{
		name:         name,
		description:  description,
		sessionAgent: sa,
		messages:     messages,
	}
}

// Compile-time check that LocalAgentAdapter implements kernel.Agent.
var _ kernel.Agent = (*LocalAgentAdapter)(nil)

// Name implements kernel.Agent.
func (a *LocalAgentAdapter) Name() string {
	return a.name
}

// Description implements kernel.Agent.
func (a *LocalAgentAdapter) Description() string {
	return a.description
}

// SystemPrompt implements kernel.Agent.
// Returns empty string as system prompt is managed internally by SessionAgent.
func (a *LocalAgentAdapter) SystemPrompt(ctx context.Context, sessionID string) (string, error) {
	return "", nil
}

// AddMemory implements kernel.Agent.
// Not yet implemented for LocalAgentAdapter.
func (a *LocalAgentAdapter) AddMemory(ctx context.Context, sessionID, content string) error {
	return nil
}

// GetMemories implements kernel.Agent.
// Not yet implemented for LocalAgentAdapter.
func (a *LocalAgentAdapter) GetMemories(ctx context.Context, sessionID string) ([]string, error) {
	return nil, nil
}

// ClearMemories implements kernel.Agent.
// Not yet implemented for LocalAgentAdapter.
func (a *LocalAgentAdapter) ClearMemories(ctx context.Context, sessionID string) error {
	return nil
}

// ListTools implements kernel.Agent.
// Returns empty list as tools are managed internally by SessionAgent.
func (a *LocalAgentAdapter) ListTools(ctx context.Context) ([]kernel.ToolDefinition, error) {
	return nil, nil
}

// ExecuteTool implements kernel.Agent.
// Tool execution is handled internally by SessionAgent.
func (a *LocalAgentAdapter) ExecuteTool(ctx context.Context, call kernel.ToolCall, infra kernel.Infra) (*kernel.ToolResult, error) {
	return nil, nil
}

// Run implements kernel.Agent.
// It runs the SessionAgent in a goroutine and streams messages via messageCh.
func (a *LocalAgentAdapter) Run(ctx context.Context, sessionID, prompt string, attachments []kernel.Attachment, messageCh chan<- kernel.Event[kernel.Message]) error {
	// Convert kernel attachments to message attachments.
	var msgAttachments []message.Attachment
	for _, att := range attachments {
		msgAttachments = append(msgAttachments, message.Attachment{
			FilePath: att.FilePath,
			FileName: att.FileName,
			MimeType: att.MimeType,
			Content:  att.Content,
		})
	}

	// Subscribe to message updates and forward them to messageCh.
	var wg sync.WaitGroup
	msgCtx, msgCancel := context.WithCancel(ctx)
	defer msgCancel()

	// Subscribe to message events and stream them.
	wg.Add(1)
	go func() {
		defer wg.Done()
		sub := a.messages.Subscribe(msgCtx)
		for {
			select {
			case <-msgCtx.Done():
				return
			case event, ok := <-sub:
				if !ok {
					return
				}
				// Only forward messages for this session.
				if event.Payload.SessionID != sessionID {
					continue
				}
				// Convert message.Message to kernel.Message.
				kernelMsg := a.messageToKernel(&event.Payload)
				select {
				case messageCh <- kernel.Event[kernel.Message]{
					Type:    kernel.EventType(event.Type),
					Payload: kernelMsg,
				}:
				case <-msgCtx.Done():
					return
				}
			}
		}
	}()

	// Run the SessionAgent in a goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := a.sessionAgent.Run(ctx, agent.SessionAgentCall{
			SessionID:   sessionID,
			Prompt:      prompt,
			Attachments: msgAttachments,
		})
		if err != nil {
			slog.Error("SessionAgent.Run error", "session_id", sessionID, "error", err)
		}
		msgCancel()
	}()

	wg.Wait()
	return nil
}

// Cancel implements kernel.Agent.
func (a *LocalAgentAdapter) Cancel(sessionID string) {
	a.sessionAgent.Cancel(sessionID)
}

// CancelAll implements kernel.Agent.
func (a *LocalAgentAdapter) CancelAll() {
	a.sessionAgent.CancelAll()
}

// IsBusy implements kernel.Agent.
func (a *LocalAgentAdapter) IsBusy() bool {
	return a.sessionAgent.IsBusy()
}

// IsSessionBusy implements kernel.Agent.
func (a *LocalAgentAdapter) IsSessionBusy(sessionID string) bool {
	return a.sessionAgent.IsSessionBusy(sessionID)
}

// GetModel implements kernel.Agent.
func (a *LocalAgentAdapter) GetModel() kernel.ModelConfig {
	m := a.sessionAgent.Model()
	return kernel.ModelConfig{
		ProviderID: m.ModelCfg.Provider,
		ModelID:    m.ModelCfg.Model,
	}
}

// SetModel implements kernel.Agent.
func (a *LocalAgentAdapter) SetModel(ctx context.Context, model kernel.ModelConfig) error {
	// Model updates would require re-initializing the SessionAgent.
	// This is a stub for future implementation.
	return nil
}

// GetDefaultSmallModel implements kernel.Agent.
func (a *LocalAgentAdapter) GetDefaultSmallModel(providerID string) kernel.ModelConfig {
	return kernel.ModelConfig{}
}

// Init implements kernel.Agent.
// No-op for LocalAgentAdapter as SessionAgent is already initialized.
func (a *LocalAgentAdapter) Init(ctx context.Context, infra kernel.Infra) error {
	return nil
}

// messageToKernel converts a message.Message to a kernel.Message.
func (a *LocalAgentAdapter) messageToKernel(m *message.Message) kernel.Message {
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
