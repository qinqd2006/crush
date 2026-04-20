// Package chat provides UI components for displaying chat messages and tool calls.
package chat

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/qinqd2006/crush/kernel/pkg"
	"github.com/qinqd2006/crush/ui/pkg/common"
	"github.com/qinqd2006/crush/ui/pkg/styles"
)

// DefaultRegistry creates a registry with all built-in tool renderers registered.
// This provides a ready-to-use registry for typical Crush installations.
func DefaultRegistry(sty *styles.Styles) *kernel.Registry {
	reg := kernel.NewRegistry()

	// Register built-in tool renderers
	reg.Register(&BashRenderer{sty: sty})
	reg.Register(&ViewRenderer{sty: sty})
	reg.Register(&EditRenderer{sty: sty})
	reg.Register(&WriteRenderer{sty: sty})
	reg.Register(&GlobRenderer{sty: sty})
	reg.Register(&GrepRenderer{sty: sty})
	reg.Register(&LSRenderer{sty: sty})
	reg.Register(&FetchRenderer{sty: sty})
	reg.Register(&DownloadRenderer{sty: sty})
	reg.Register(&TodoRenderer{sty: sty})

	return reg
}

// ============================================================================
// Individual Tool Renderers
// Each renderer handles a specific tool type.
// They are decoupled from the engine: they only receive json.RawMessage.
// ============================================================================

// BashRenderer renders bash tool calls.
type BashRenderer struct {
	sty *styles.Styles
}

func (r *BashRenderer) Name() string { return "bash" }

func (r *BashRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(input, &params) == nil {
		cmd := params.Command
		if len(cmd) > 50 {
			cmd = cmd[:50] + "..."
		}
		return "Running: " + cmd
	}
	return "bash"
}

func (r *BashRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	cmd := params.Command
	if len(cmd) > width {
		cmd = ansi.Truncate(cmd, width-3, "...")
	}

	prompt := r.sty.Tool.ContentLine.Render("$ " + cmd)
	return r.sty.Tool.Body.Render(prompt)
}

func (r *BashRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}
	if result.Content == "" {
		return r.sty.Tool.StateCancelled.Render("(no output)")
	}
	return r.sty.Tool.ContentLine.Width(width).Render(result.Content)
}

// ViewRenderer renders view tool calls.
type ViewRenderer struct {
	sty *styles.Styles
}

func (r *ViewRenderer) Name() string { return "view" }

func (r *ViewRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(input, &params) == nil {
		return "View: " + params.FilePath
	}
	return "view"
}

func (r *ViewRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		FilePath string `json:"file_path"`
		Offset   int    `json:"offset"`
		Limit    int    `json:"limit"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.FilePath
	if params.Offset > 0 || params.Limit > 0 {
		summary += " ("
		if params.Offset > 0 {
			summary += "offset=" + itoa(params.Offset)
		}
		if params.Limit > 0 {
			if params.Offset > 0 {
				summary += ", "
			}
			summary += "limit=" + itoa(params.Limit)
		}
		summary += ")"
	}

	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *ViewRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}

	// Try to extract metadata from result.Data
	var meta struct {
		Content  string `json:"content"`
		FilePath string `json:"file_path"`
	}
	if len(result.Data) > 0 {
		json.Unmarshal(result.Data, &meta)
	}

	content := meta.Content
	if content == "" {
		content = result.Content
	}

	if meta.FilePath != "" && content != "" {
		highlighted, _ := common.SyntaxHighlight(r.sty, content, meta.FilePath, r.sty.Tool.ContentCodeBg)
		return r.sty.Tool.Body.Render(highlighted)
	}

	return result.Content
}

// EditRenderer renders edit tool calls.
type EditRenderer struct {
	sty *styles.Styles
}

func (r *EditRenderer) Name() string { return "edit" }

func (r *EditRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(input, &params) == nil {
		return "Edit: " + params.FilePath
	}
	return "edit"
}

func (r *EditRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		FilePath   string `json:"file_path"`
		OldContent string `json:"old_string"`
		NewContent string `json:"new_string"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.FilePath
	if params.OldContent != "" {
		oldTrunc := params.OldContent
		if len(oldTrunc) > 30 {
			oldTrunc = oldTrunc[:30] + "..."
		}
		summary += " (" + oldTrunc + ")"
	}

	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *EditRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}

	var meta struct {
		OldContent string `json:"old_content"`
		NewContent string `json:"new_content"`
	}
	if len(result.Data) > 0 {
		json.Unmarshal(result.Data, &meta)
	}

	if meta.OldContent != "" || meta.NewContent != "" {
		diffOutput := common.DiffFormatter(r.sty).
			Before("", meta.OldContent).
			After("", meta.NewContent).
			Width(width - 2).
			String()
		return r.sty.Tool.Body.Render(diffOutput)
	}

	return result.Content
}

// WriteRenderer renders write tool calls.
type WriteRenderer struct {
	sty *styles.Styles
}

func (r *WriteRenderer) Name() string { return "write" }

func (r *WriteRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(input, &params) == nil {
		return "Write: " + params.FilePath
	}
	return "write"
}

func (r *WriteRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.FilePath
	if params.Content != "" {
		lines := countLines(params.Content)
		summary += " (" + itoa(lines) + " lines)"
	}

	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *WriteRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}
	return r.sty.Tool.StateCancelled.Render("File written successfully")
}

// GlobRenderer renders glob tool calls.
type GlobRenderer struct {
	sty *styles.Styles
}

func (r *GlobRenderer) Name() string { return "glob" }

func (r *GlobRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		Pattern string `json:"pattern"`
	}
	if json.Unmarshal(input, &params) == nil {
		return "Glob: " + params.Pattern
	}
	return "glob"
}

func (r *GlobRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.Pattern
	if params.Path != "" {
		summary += " in " + params.Path
	}

	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *GlobRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}
	return result.Content
}

// GrepRenderer renders grep tool calls.
type GrepRenderer struct {
	sty *styles.Styles
}

func (r *GrepRenderer) Name() string { return "grep" }

func (r *GrepRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		Pattern string `json:"pattern"`
	}
	if json.Unmarshal(input, &params) == nil {
		return "Grep: " + params.Pattern
	}
	return "grep"
}

func (r *GrepRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
		Include string `json:"include"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.Pattern
	if params.Path != "" {
		summary += " in " + params.Path
	}
	if params.Include != "" {
		summary += " (" + params.Include + ")"
	}

	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *GrepRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}
	return result.Content
}

// LSRenderer renders ls tool calls.
type LSRenderer struct {
	sty *styles.Styles
}

func (r *LSRenderer) Name() string { return "ls" }

func (r *LSRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		Path string `json:"path"`
	}
	if json.Unmarshal(input, &params) == nil {
		if params.Path == "" {
			params.Path = "."
		}
		return "LS: " + params.Path
	}
	return "ls"
}

func (r *LSRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		Path   string `json:"path"`
		Ignore string `json:"ignore"`
		Depth  int    `json:"depth"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.Path
	if params.Path == "" {
		summary = "."
	}
	if params.Depth > 0 {
		summary += " (depth=" + itoa(params.Depth) + ")"
	}

	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *LSRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}
	return result.Content
}

// FetchRenderer renders fetch tool calls.
type FetchRenderer struct {
	sty *styles.Styles
}

func (r *FetchRenderer) Name() string { return "fetch" }

func (r *FetchRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(input, &params) == nil {
		return "Fetch: " + params.URL
	}
	return "fetch"
}

func (r *FetchRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		URL     string `json:"url"`
		Format  string `json:"format"`
		Timeout int    `json:"timeout"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.URL
	if params.Format != "" {
		summary += " [" + params.Format + "]"
	}

	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *FetchRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}
	return result.Content
}

// DownloadRenderer renders download tool calls.
type DownloadRenderer struct {
	sty *styles.Styles
}

func (r *DownloadRenderer) Name() string { return "download" }

func (r *DownloadRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		URL      string `json:"url"`
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(input, &params) == nil {
		return "Download: " + params.URL
	}
	return "download"
}

func (r *DownloadRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		URL      string `json:"url"`
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.URL + " → " + params.FilePath
	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *DownloadRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}
	return r.sty.Tool.StateCancelled.Render("Downloaded successfully")
}

// TodoRenderer renders todo tool calls.
type TodoRenderer struct {
	sty *styles.Styles
}

func (r *TodoRenderer) Name() string { return "todo" }

func (r *TodoRenderer) RenderSummary(input json.RawMessage) string {
	var params struct {
		Action string `json:"action"`
	}
	if json.Unmarshal(input, &params) == nil {
		return "Todo: " + params.Action
	}
	return "todo"
}

func (r *TodoRenderer) RenderArguments(input json.RawMessage, width int) string {
	var params struct {
		Action      string `json:"action"`
		Description string `json:"description"`
	}
	if json.Unmarshal(input, &params) != nil {
		return string(input)
	}

	summary := params.Action
	if params.Description != "" {
		summary += ": " + params.Description
	}

	return r.sty.Tool.Body.Render(r.sty.Tool.ContentLine.Render(summary))
}

func (r *TodoRenderer) RenderResult(result *kernel.ToolResult, width int) string {
	if result.IsError {
		return r.sty.Tool.ErrorMessage.Render(result.Content)
	}
	return result.Content
}

// ============================================================================
// Helper functions
// ============================================================================

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + uitoa(-i)
	}
	return uitoa(i)
}

func uitoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := 1
	for _, c := range s {
		if c == '\n' {
			count++
		}
	}
	return count
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + strings.Repeat(".", 3)
}
