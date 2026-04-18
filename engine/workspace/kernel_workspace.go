package workspace

import (
	"context"
	"log/slog"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/kernel"
)

// KernelWorkspace implements the Workspace interface by delegating
// to a kernel.Kernel instance. Internally it uses AppWorkspace for
// Workspace operations and event subscription, providing a path for
// gradual migration to kernel-based access.
type KernelWorkspace struct {
	*AppWorkspace                 // Embedded - handles all Workspace operations
	kernel          kernel.Kernel // Exposed for direct kernel access
	subscribeCtx    context.Context
	subscribeCancel context.CancelFunc
	subscribeWG     sync.WaitGroup
}

// NewKernelWorkspace creates a new KernelWorkspace that wraps an AppWorkspace
// and exposes the kernel for decoupled access.
func NewKernelWorkspace(appWs *AppWorkspace, k kernel.Kernel) *KernelWorkspace {
	return &KernelWorkspace{
		AppWorkspace: appWs,
		kernel:       k,
	}
}

// Compile-time check that KernelWorkspace implements Workspace.
var _ Workspace = (*KernelWorkspace)(nil)

// Kernel returns the underlying kernel.Kernel instance.
func (w *KernelWorkspace) Kernel() kernel.Kernel {
	return w.kernel
}

// Subscribe is overridden to be a no-op for KernelWorkspace.
// Kernel-based subscriptions are handled by KernelSubscribe instead.
func (w *KernelWorkspace) Subscribe(program *tea.Program) {
	// No-op: kernel subscriptions are handled by KernelSubscribe
}

// KernelSessions returns all sessions as kernel.Session types.
func (w *KernelWorkspace) KernelSessions() []kernel.Session {
	sessions, _ := w.kernel.ListSessions(nil)
	return sessions
}

// KernelMessages returns all messages for a session as kernel.Message types.
func (w *KernelWorkspace) KernelMessages(sessionID string) []kernel.Message {
	msgs, _ := w.kernel.ListMessages(nil, sessionID)
	return msgs
}

// KernelEventMsg is a generic wrapper for kernel events that can be sent as tea.Msg.
type KernelEventMsg[T any] struct {
	Event kernel.Event[T]
}

// KernelSubscribe sets up subscriptions to kernel events and sends them
// to the tea.Program as tea.Msgs. This allows the UI to receive decoupled
// kernel events instead of internal pubsub events.
func (w *KernelWorkspace) KernelSubscribe(program *tea.Program) {
	if w.kernel == nil {
		slog.Warn("KernelSubscribe called but kernel is nil")
		return
	}

	w.subscribeCtx, w.subscribeCancel = context.WithCancel(context.Background())

	w.subscribeWG.Add(8) // 8 subscription goroutines

	// Subscribe to session events
	go func() {
		defer w.subscribeWG.Done()
		for event := range w.kernel.SubscribeSessions(w.subscribeCtx) {
			program.Send(KernelEventMsg[kernel.Session]{Event: event})
		}
	}()

	// Subscribe to message events
	go func() {
		defer w.subscribeWG.Done()
		for event := range w.kernel.SubscribeMessages(w.subscribeCtx) {
			program.Send(KernelEventMsg[kernel.Message]{Event: event})
		}
	}()

	// Subscribe to permission request events
	go func() {
		defer w.subscribeWG.Done()
		for event := range w.kernel.SubscribePermissions(w.subscribeCtx) {
			program.Send(KernelEventMsg[kernel.PermissionRequest]{Event: event})
		}
	}()

	// Subscribe to permission notification events
	go func() {
		defer w.subscribeWG.Done()
		for event := range w.kernel.SubscribePermissionNotifications(w.subscribeCtx) {
			program.Send(KernelEventMsg[kernel.PermissionNotification]{Event: event})
		}
	}()

	// Subscribe to notification events
	go func() {
		defer w.subscribeWG.Done()
		for event := range w.kernel.SubscribeNotifications(w.subscribeCtx) {
			program.Send(KernelEventMsg[kernel.Notification]{Event: event})
		}
	}()

	// Subscribe to MCP events
	go func() {
		defer w.subscribeWG.Done()
		for event := range w.kernel.SubscribeMCP(w.subscribeCtx) {
			program.Send(KernelEventMsg[kernel.MCPEvent]{Event: event})
		}
	}()

	// Subscribe to LSP events
	go func() {
		defer w.subscribeWG.Done()
		for event := range w.kernel.SubscribeLSP(w.subscribeCtx) {
			program.Send(KernelEventMsg[kernel.LSPEvent]{Event: event})
		}
	}()

	// Subscribe to history events
	go func() {
		defer w.subscribeWG.Done()
		for event := range w.kernel.SubscribeHistory(w.subscribeCtx) {
			program.Send(KernelEventMsg[kernel.HistoryFile]{Event: event})
		}
	}()
}

// Shutdown overrides AppWorkspace.Shutdown to also cancel kernel subscriptions.
func (w *KernelWorkspace) Shutdown() {
	if w.subscribeCancel != nil {
		w.subscribeCancel()
		w.subscribeWG.Wait()
	}
	w.AppWorkspace.Shutdown()
}
