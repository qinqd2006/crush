package common

import (
	"fmt"
	"image"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/qinqd2006/crush/crush-agent/pkg/config"
	"github.com/qinqd2006/crush/kernel/pkg"
	"github.com/qinqd2006/crush/ui/pkg/styles"
	"github.com/qinqd2006/crush/ui/pkg/util"
)

// KernelEventMsg is a generic wrapper for kernel events that can be sent as tea.Msg.
type KernelEventMsg[T any] struct {
	Event kernel.Event[T]
}

// MaxAttachmentSize defines the maximum allowed size for file attachments (5 MB).
const MaxAttachmentSize = int64(5 * 1024 * 1024)

// AllowedImageTypes defines the permitted image file types.
var AllowedImageTypes = []string{".jpg", ".jpeg", ".png"}

// Common defines common UI options and configurations.
type Common struct {
	Workspace       kernel.Workspace
	KernelWorkspace kernel.Workspace
	Styles          *styles.Styles
}

// Config returns the configuration associated with this [Common] instance.
func (c *Common) Config() *config.Config {
	return c.Workspace.EngineConfig().(*config.Config)
}

// Kernel returns the underlying kernel.Kernel instance, enabling kernel-based
// access for code that needs to work with kernel types. Returns nil if the
// workspace does not expose a kernel.
//
// Deprecated: This method is kept for backward compatibility. New code should
// use kernel.Workspace methods directly.
// Kernel returns the underlying kernel.Kernel instance.
// For WorkspaceAdapter, this returns nil since it doesn't expose a kernel.
// Use KernelWorkspace for the kernel interface instead.
func (c *Common) Kernel() kernel.Kernel {
	return nil
}

// KernelConfig returns the kernel configuration provider. This provides access
// to the configuration through the kernel interface. For full config access,
// use Config() which returns the engine config.
func (c *Common) KernelConfig() kernel.ConfigProvider {
	return c.Workspace.Config()
}

// DefaultCommon returns the default common UI configurations.
func DefaultCommon(ws kernel.Workspace, kw kernel.Workspace) *Common {
	s := styles.DefaultStyles()
	return &Common{
		Workspace:       ws,
		KernelWorkspace: kw,
		Styles:          &s,
	}
}

// CenterRect returns a new [Rectangle] centered within the given area with the
// specified width and height.
func CenterRect(area uv.Rectangle, width, height int) uv.Rectangle {
	centerX := area.Min.X + area.Dx()/2
	centerY := area.Min.Y + area.Dy()/2
	minX := centerX - width/2
	minY := centerY - height/2
	maxX := minX + width
	maxY := minY + height
	return image.Rect(minX, minY, maxX, maxY)
}

// BottomLeftRect returns a new [Rectangle] positioned at the bottom-left within the given area with the
// specified width and height.
func BottomLeftRect(area uv.Rectangle, width, height int) uv.Rectangle {
	minX := area.Min.X
	maxX := minX + width
	maxY := area.Max.Y
	minY := maxY - height
	return image.Rect(minX, minY, maxX, maxY)
}

// IsFileTooBig checks if the file at the given path exceeds the specified size
// limit.
func IsFileTooBig(filePath string, sizeLimit int64) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("error getting file info: %w", err)
	}

	if fileInfo.Size() > sizeLimit {
		return true, nil
	}

	return false, nil
}

// CopyToClipboard copies the given text to the clipboard using both OSC 52
// (terminal escape sequence) and native clipboard for maximum compatibility.
// Returns a command that reports success to the user with the given message.
func CopyToClipboard(text, successMessage string) tea.Cmd {
	return CopyToClipboardWithCallback(text, successMessage, nil)
}

// CopyToClipboardWithCallback copies text to clipboard and executes a callback
// before showing the success message.
// This is useful when you need to perform additional actions like clearing UI state.
func CopyToClipboardWithCallback(text, successMessage string, callback tea.Cmd) tea.Cmd {
	return tea.Sequence(
		tea.SetClipboard(text),
		func() tea.Msg {
			_ = clipboard.WriteAll(text)
			return nil
		},
		callback,
		util.ReportInfo(successMessage),
	)
}
