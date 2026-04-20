package kernel

// MCPTool represents a tool provided by an MCP server.
type MCPTool struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

// MCPResource represents a resource provided by an MCP server.
type MCPResource struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mime_type,omitempty"`
	URI         string `json:"uri,omitempty"`
}

// MCPToolResult represents the result of running an MCP tool.
type MCPToolResult struct {
	Type      string `json:"type"` // "text", "image", "media"
	Content   string `json:"content"`
	Data      []byte `json:"data,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}
