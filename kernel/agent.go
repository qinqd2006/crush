// Package kernel defines the core interfaces for multi-agent UI architecture.
// It provides the abstraction layer that allows different agent implementations
// (Crush, OpenCode, Hermes, etc.) to share the same UI.
package kernel

import (
	"context"
)

// ============================================================================
// Agent Interface
// ============================================================================

// ToolDefinition describes a tool that can be called by the agent.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// ToolCall represents a request to execute a tool.
type ToolCall struct {
	ID     string
	Name   string
	Input  map[string]any
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ID      string
	Name    string
	Content string
	Data    []byte
	IsError bool
}

// Agent defines the interface that each agent implementation must satisfy.
// Each agent (Crush, OpenCode, Hermes, etc.) implements this interface
// to provide its specific behavior for prompt handling, memory, and tool execution.
type Agent interface {
	// ------------------------------------------------------------------------
	// Identity
	// ------------------------------------------------------------------------

	// Name returns the agent's identifier.
	Name() string

	// Description returns a human-readable description of the agent.
	Description() string

	// ------------------------------------------------------------------------
	// System Prompt
	// ------------------------------------------------------------------------

	// SystemPrompt returns the system prompt for this agent.
	// This can be dynamic based on context, project, etc.
	SystemPrompt(ctx context.Context, sessionID string) (string, error)

	// ------------------------------------------------------------------------
	// Memory Management
	// ------------------------------------------------------------------------

	// AddMemory adds a memory entry to the agent's context.
	AddMemory(ctx context.Context, sessionID, content string) error

	// GetMemories returns all memories for a session.
	GetMemories(ctx context.Context, sessionID string) ([]string, error)

	// ClearMemories clears all memories for a session.
	ClearMemories(ctx context.Context, sessionID string) error

	// ------------------------------------------------------------------------
	// Tools
	// ------------------------------------------------------------------------

	// ListTools returns all tools exposed by this agent.
	ListTools(ctx context.Context) ([]ToolDefinition, error)

	// ExecuteTool executes a tool call and returns the result.
	// The infra layer is responsible for actually running the tool.
	ExecuteTool(ctx context.Context, call ToolCall, infra Infra) (*ToolResult, error)

	// ------------------------------------------------------------------------
	// Execution
	// ------------------------------------------------------------------------

	// Run executes the agent with the given prompt and attachments.
	// This is called by the workspace when a user sends a message.
	// The agent should send messages back via the messageCh channel.
	Run(ctx context.Context, sessionID, prompt string, attachments []Attachment, messageCh chan<- Event[Message]) error

	// Cancel stops the current execution for a session.
	Cancel(sessionID string)

	// CancelAll stops all executions.
	CancelAll()

	// IsBusy returns true if the agent is processing any request.
	IsBusy() bool

	// IsSessionBusy returns true if the agent is processing for this session.
	IsSessionBusy(sessionID string) bool

	// ------------------------------------------------------------------------
	// Configuration
	// ------------------------------------------------------------------------

	// GetModel returns the current model configuration for this agent.
	GetModel() ModelConfig

	// SetModel updates the model configuration for this agent.
	SetModel(ctx context.Context, model ModelConfig) error

	// GetDefaultSmallModel returns a suitable small/fast model for quick tasks.
	GetDefaultSmallModel(providerID string) ModelConfig

	// ------------------------------------------------------------------------
	// Initialization
	// ------------------------------------------------------------------------

	// Init performs any necessary initialization for the agent.
	// This is called once when the workspace is created.
	Init(ctx context.Context, infra Infra) error
}

// ModelConfig holds the configuration for an LLM model.
type ModelConfig struct {
	ProviderID string
	ModelID    string
}
