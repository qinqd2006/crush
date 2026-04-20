// Package kernel defines the core interface between the UI layer and the
// agent/coordinator engine. This package contains no engine-specific
// code and can be used by any Go project that wants to build a UI for
// an AI agent.
package kernel

import "encoding/json"

// ToolCategory categorizes tools for UI display purposes.
type ToolCategory string

const (
	ToolCategorySystem ToolCategory = "system"
	ToolCategoryFile   ToolCategory = "file"
	ToolCategoryWeb    ToolCategory = "web"
	ToolCategoryMCP    ToolCategory = "mcp"
	ToolCategoryAgent  ToolCategory = "agent"
	ToolCategoryOther  ToolCategory = "other"
)

// ToolInfo provides metadata about a tool for UI rendering.
type ToolInfo struct {
	Name        string       `json:"name"`
	Category    ToolCategory `json:"category"`
	Description string       `json:"description,omitempty"`
}

// ToolRenderer defines the interface for rendering tool inputs and outputs.
// UI implementations should register renderers for each tool they want to
// display with custom formatting.
type ToolRenderer interface {
	// Name returns the tool name this renderer handles.
	Name() string

	// RenderArguments renders the tool arguments for display.
	// Input is raw JSON from the tool call.
	RenderArguments(input json.RawMessage, width int) string

	// RenderResult renders the tool result for display.
	RenderResult(result *ToolResult, width int) string

	// RenderSummary returns a one-line summary for collapsed view.
	RenderSummary(input json.RawMessage) string
}

// Registry is a registry of tool renderers.
// It maps tool names to their renderers.
type Registry struct {
	renderers map[string]ToolRenderer
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{
		renderers: make(map[string]ToolRenderer),
	}
}

// Register registers a tool renderer.
func (r *Registry) Register(renderer ToolRenderer) {
	r.renderers[renderer.Name()] = renderer
}

// Get returns the renderer for a tool name, or nil if not found.
func (r *Registry) Get(name string) ToolRenderer {
	return r.renderers[name]
}

// RenderArguments renders tool arguments using the registered renderer.
// Falls back to raw JSON if no renderer is registered.
func (r *Registry) RenderArguments(name string, input json.RawMessage, width int) string {
	if renderer, ok := r.renderers[name]; ok {
		return renderer.RenderArguments(input, width)
	}
	return string(input)
}

// RenderResult renders tool result using the registered renderer.
// Falls back to raw output if no renderer is registered.
func (r *Registry) RenderResult(name string, result *ToolResult, width int) string {
	if renderer, ok := r.renderers[name]; ok {
		return renderer.RenderResult(result, width)
	}
	if result.IsError {
		return result.Content
	}
	return result.Content
}

// RenderSummary returns a summary using the registered renderer.
// Falls back to tool name if no renderer is registered.
func (r *Registry) RenderSummary(name string, input json.RawMessage) string {
	if renderer, ok := r.renderers[name]; ok {
		return renderer.RenderSummary(input)
	}
	return name
}

// ============================================================================
// Tool Parameters and Metadata (Neutral Types for UI Layer)
// ============================================================================
// These types are used by the UI layer to unmarshal tool parameters and
// metadata without depending on the engine/agent/tools package.

// BashNoOutput is the output when a bash command produces no output.
const BashNoOutput = "no output"

// BashParams defines the parameters for the bash tool.
type BashParams struct {
	Description         string `json:"description,omitempty"`
	Command             string `json:"command"`
	WorkingDir          string `json:"working_dir,omitempty"`
	RunInBackground     bool   `json:"run_in_background,omitempty"`
	AutoBackgroundAfter int    `json:"auto_background_after,omitempty"`
}

// BashResponseMetadata contains metadata from bash tool execution.
type BashResponseMetadata struct {
	StartTime        int64  `json:"start_time"`
	EndTime          int64  `json:"end_time"`
	Output           string `json:"output"`
	Description      string `json:"description"`
	WorkingDirectory string `json:"working_directory"`
	Background       bool   `json:"background,omitempty"`
	ShellID          string `json:"shell_id,omitempty"`
}

// JobOutputParams defines the parameters for the job_output tool.
type JobOutputParams struct {
	ShellID string `json:"shell_id"`
	Wait    bool   `json:"wait,omitempty"`
}

// JobOutputResponseMetadata contains metadata from job_output tool execution.
type JobOutputResponseMetadata struct {
	ShellID          string `json:"shell_id"`
	Command          string `json:"command"`
	Description      string `json:"description"`
	Done             bool   `json:"done"`
	WorkingDirectory string `json:"working_directory"`
}

// JobKillParams defines the parameters for the job_kill tool.
type JobKillParams struct {
	ShellID string `json:"shell_id"`
}

// JobKillResponseMetadata contains metadata from job_kill tool execution.
type JobKillResponseMetadata struct {
	ShellID     string `json:"shell_id"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

// ViewParams defines the parameters for the view tool.
type ViewParams struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// ViewResourceType identifies special resource types for view results.
type ViewResourceType string

const (
	ViewResourceUnset ViewResourceType = ""
	ViewResourceSkill ViewResourceType = "skill"
)

// ViewResponseMetadata contains metadata from view tool execution.
type ViewResponseMetadata struct {
	FilePath            string           `json:"file_path"`
	Content             string           `json:"content"`
	ResourceType        ViewResourceType `json:"resource_type,omitempty"`
	ResourceName        string           `json:"resource_name,omitempty"`
	ResourceDescription string           `json:"resource_description,omitempty"`
}

// WriteParams defines the parameters for the write tool.
type WriteParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// EditParams defines the parameters for the edit tool.
type EditParams struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// EditResponseMetadata contains metadata from edit tool execution.
type EditResponseMetadata struct {
	Additions  int    `json:"additions"`
	Removals   int    `json:"removals"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

// MultiEditOperation defines a single edit operation.
type MultiEditOperation struct {
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// MultiEditParams defines the parameters for the multiedit tool.
type MultiEditParams struct {
	FilePath string               `json:"file_path"`
	Edits    []MultiEditOperation `json:"edits"`
}

// FailedEdit describes an edit that failed during multiedit.
type FailedEdit struct {
	Index int                `json:"index"`
	Error string             `json:"error"`
	Edit  MultiEditOperation `json:"edit"`
}

// MultiEditResponseMetadata contains metadata from multiedit tool execution.
type MultiEditResponseMetadata struct {
	Additions    int          `json:"additions"`
	Removals     int          `json:"removals"`
	OldContent   string       `json:"old_content,omitempty"`
	NewContent   string       `json:"new_content,omitempty"`
	EditsApplied int          `json:"edits_applied"`
	EditsFailed  []FailedEdit `json:"edits_failed,omitempty"`
}

// DownloadParams defines the parameters for the download tool.
type DownloadParams struct {
	URL      string `json:"url"`
	FilePath string `json:"file_path"`
	Timeout  int    `json:"timeout,omitempty"`
}

// AgenticFetchParams defines the parameters for the agentic fetch tool.
type AgenticFetchParams struct {
	URL    string `json:"url,omitempty"`
	Prompt string `json:"prompt"`
}

// FetchParams defines the parameters for the fetch tool.
type FetchParams struct {
	URL     string `json:"url"`
	Format  string `json:"format"`
	Timeout int    `json:"timeout,omitempty"`
}

// WebFetchParams defines the parameters for the web_fetch tool.
type WebFetchParams struct {
	URL string `json:"url"`
}

// WebSearchParams defines the parameters for the web_search tool.
type WebSearchParams struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

// GrepParams defines the parameters for the grep tool.
type GrepParams struct {
	Pattern     string `json:"pattern"`
	Path        string `json:"path,omitempty"`
	Include     string `json:"include,omitempty"`
	LiteralText bool   `json:"literal_text,omitempty"`
}

// GlobParams defines the parameters for the glob tool.
type GlobParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// LSParams defines the parameters for the ls tool.
type LSParams struct {
	Path   string   `json:"path,omitempty"`
	Ignore []string `json:"ignore,omitempty"`
	Depth  int      `json:"depth,omitempty"`
}

// SourcegraphParams defines the parameters for the sourcegraph tool.
type SourcegraphParams struct {
	Query         string `json:"query"`
	Count         int    `json:"count,omitempty"`
	ContextWindow int    `json:"context_window,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
}

// DiagnosticsParams defines the parameters for the diagnostics tool.
type DiagnosticsParams struct {
	FilePath string `json:"file_path,omitempty"`
}

// ReferencesParams defines the parameters for the references tool.
type ReferencesParams struct {
	Symbol string `json:"symbol"`
	Path   string `json:"path,omitempty"`
}

// LSPRestartParams defines the parameters for the lsp_restart tool.
type LSPRestartParams struct {
	Name string `json:"name,omitempty"`
}

// TodosParams defines the parameters for the todos tool.
type TodosParams struct {
	Todos []TodoItem `json:"todos"`
}

// TodoItem represents a single todo item.
type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"active_form,omitempty"`
}

// TodosResponseMetadata contains metadata from todos tool execution.
type TodosResponseMetadata struct {
	IsNew         bool     `json:"is_new"`
	Todos         []Todo   `json:"todos"`
	JustCompleted []string `json:"just_completed,omitempty"`
	JustStarted   string   `json:"just_started,omitempty"`
	Completed     int      `json:"completed"`
	Total         int      `json:"total"`
}

// AgentParams defines the parameters for the agent tool.
type AgentParams struct {
	Prompt string `json:"prompt"`
}
