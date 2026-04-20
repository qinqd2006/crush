# Multi-Agent Architecture Migration Guide

## Overview

This document describes the migration path from the current Crush architecture to a multi-agent architecture where multiple agents (Crush, OpenCode, Hermes) can share the same UI infrastructure.

## Current Architecture

```
ui/
  └── imports engine/* directly (100+ imports)
       ├── engine/message
       ├── engine/session
       ├── engine/config
       └── ... (20+ packages)

engine/
  └── contains all implementations
       ├── agent/
       ├── app/
       ├── workspace/
       └── ...

kernel/
  └── interfaces only (no implementations)
       ├── agent.go      (Agent interface)
       ├── infra.go      (Infra interface)
       ├── workspace.go  (Workspace interface)
       └── workspace_adapter.go
```

## Target Architecture

```
ui/
  └── imports kernel/* only
       └── kernel (Workspace interface)

kernel/
  ├── Agent interface (Crush, OpenCode, Hermes implementations)
  ├── Infra interface (MCP, LSP, Storage, Permissions, Skills)
  └── Workspace interface (UI contract)

engine/
  └── implementations for current Crush agent
       └── can be wrapped by kernel adapters
```

## Migration Strategy

### Phase 1: Complete ✅

**Kernel Interfaces** - Done
- `kernel/agent.go` - Agent interface
- `kernel/infra.go` - Infra interface  
- `kernel/workspace.go` - Workspace interface
- `kernel/workspace_adapter.go` - Adapter implementation
- `kernel/kernel.go` - Core types (Message, Session, etc.)

### Phase 2: UI Type Migration (Pending)

**Approach**: Incremental migration of UI files from `engine/*` imports to `kernel` types.

#### Step 2.1: Change Workspace Import

In `ui/common/common.go`:
```go
// Before
import "github.com/qinqd2006/crush/engine/workspace"

// After  
import "github.com/qinqd2006/crush/kernel"
```

However, `kernel.Workspace` and `engine/workspace.Workspace` have different method signatures:

| engine/workspace.Workspace | kernel.Workspace |
|---------------------------|------------------|
| `ListMessages() []message.Message` | `ListMessages() []kernel.Message` |
| `Config() *config.Config` | `Config() ConfigProvider` |

The types are similar but not identical, requiring adapters.

#### Step 2.2: Create Type Adapters

Create adapter functions to convert between engine and kernel types:

```go
// engine/message.Message → kernel.Message
func MessageToKernel(m *message.Message) kernel.Message {
    return kernel.Message{
        ID:        m.ID,
        Role:      kernel.MessageRole(m.Role),
        Parts:     PartsToKernel(m.Parts),
        // ...
    }
}
```

#### Step 2.3: Migrate Chat Components

The `ui/chat/` package uses many engine types:
- `ui/chat/messages.go` - Uses `message.Message`, `message.Attachment`
- `ui/chat/tools.go` - Uses `message.ToolResult`, `message.ToolCall`
- etc.

**Recommendation**: Start with `ui/chat/messages.go` since it's fundamental.

### Phase 3: Agent Implementations (Pending)

Create new agent packages:

```
kernel/
agents/
  ├── crush/
  │   ├── agent.go    (implements kernel.Agent)
  │   ├── prompts.go
  │   └── tools.go
  ├── opencode/
  │   └── ...
  └── hermes/
      └── ...
```

## Files to Change for UI Migration

### Critical Files (Start Here)

1. **`ui/common/common.go`**
   - Change `workspace.Workspace` → `kernel.Workspace`
   - Change `Config() *config.Config` → `Config() kernel.ConfigProvider`

2. **`ui/model/ui.go`**
   - Imports: `engine/workspace`, `engine/message`, `engine/config`
   - Uses workspace for: sessions, messages, config, MCP, LSP

3. **`ui/chat/messages.go`**
   - Uses `message.Message` extensively
   - Uses `message.Attachment`, `message.ToolResult`

### Supporting Files

- `ui/chat/tools.go`, `ui/chat/bash.go`, `ui/chat/file.go`
- `ui/dialog/sessions.go`, `ui/dialog/actions.go`
- `ui/model/session.go`, `ui/model/header.go`

## Type Mapping Reference

| Engine Type | Kernel Type | Adapter Method |
|------------|-------------|---------------|
| `message.Message` | `kernel.Message` | `messageToKernel()` |
| `message.Attachment` | `kernel.Attachment` | Manual conversion |
| `session.Session` | `kernel.Session` | `sessionToKernel()` |
| `config.Config` | `kernel.ConfigProvider` | `ConfigAdapter` |
| `message.TextContent` | `kernel.TextContent` | Direct copy |
| `message.ToolCall` | `kernel.ToolCallContent` | Field rename |
| `message.ToolResult` | `kernel.ToolResultContent` | Field rename |
| `message.Finish` | `kernel.FinishContent` | Field rename |
| `message.ImageURLContent` | `kernel.ImageURLContent` | Direct copy |
| `message.BinaryContent` | `kernel.BinaryContent` | Direct copy |

## Key Adapters

### WorkspaceAdapter (`engine/adapter/workspace_adapter.go`)

Provides `kernel.Workspace` implementation wrapping engine workspace:

```go
type WorkspaceAdapter struct {
    impl workspace.Workspace  // Engine's workspace interface
}

func NewWorkspaceAdapter(impl workspace.Workspace) *WorkspaceAdapter
```

### ConfigAdapter (`engine/adapter/workspace_adapter.go`)

Wraps `*config.Config` to implement `kernel.ConfigProvider`:

```go
type ConfigAdapter struct {
    cfg *config.Config
}
```

### ResolverAdapter (`engine/adapter/workspace_adapter.go`)

Converts between engine and kernel variable resolver signatures:
- Engine: `ResolveValue(key) (string, error)`
- Kernel: `Resolve(key) (string, bool)`

## Testing

Run tests after each migration step:
```bash
go test ./kernel/...
go test ./engine/...
go test ./ui/...
go build ./cmd
```

## Notes

1. **Breaking Changes**: The migration involves breaking changes to the UI layer.
2. **Gradual Migration**: Can migrate package by package.
3. **Backward Compatibility**: Can maintain both `engine/*` and `kernel` imports during transition.
4. **Testing Important**: Each UI component should be tested after migration.
