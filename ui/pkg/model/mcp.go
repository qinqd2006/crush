package model

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/qinqd2006/crush/infra/pkg/mcp"
	"github.com/qinqd2006/crush/kernel/pkg"
	"github.com/qinqd2006/crush/ui/pkg/common"
	"github.com/qinqd2006/crush/ui/pkg/styles"
)

// mcpInfo renders the MCP status section showing active MCP clients and their
// tool/prompt counts.
func (m *UI) mcpInfo(width, maxItems int, isSection bool) string {
	var mcps []kernel.MCPClientInfo
	t := m.com.Styles

	for _, mcpServer := range m.com.KernelConfig().ListMCPServers() {
		if state, ok := m.mcpStates[mcpServer.Name]; ok {
			mcps = append(mcps, state)
		}
	}

	title := t.ResourceGroupTitle.Render("MCPs")
	if isSection {
		title = common.Section(t, title, width)
	}
	list := t.ResourceAdditionalText.Render("None")
	if len(mcps) > 0 {
		list = mcpList(t, mcps, width, maxItems)
	}

	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}

// mcpCounts formats tool, prompt, and resource counts for display.
func mcpCounts(t *styles.Styles, client kernel.MCPClientInfo) string {
	var parts []string
	if client.ToolCount > 0 {
		parts = append(parts, t.Subtle.Render(fmt.Sprintf("%d tools", client.ToolCount)))
	}
	if client.PromptsCount > 0 {
		parts = append(parts, t.Subtle.Render(fmt.Sprintf("%d prompts", client.PromptsCount)))
	}
	if client.ResourcesCount > 0 {
		parts = append(parts, t.Subtle.Render(fmt.Sprintf("%d resources", client.ResourcesCount)))
	}
	return strings.Join(parts, " ")
}

// mcpList renders a list of MCP clients with their status and counts,
// truncating to maxItems if needed.
func mcpList(t *styles.Styles, mcps []kernel.MCPClientInfo, width, maxItems int) string {
	if maxItems <= 0 {
		return ""
	}
	var renderedMcps []string

	for _, m := range mcps {
		var icon string
		title := m.Name
		// Show "Docker MCP" instead of the config name for Docker MCP.
		if m.Name == mcp.DockerMCPName {
			title = "Docker MCP"
		}
		title = t.ResourceName.Render(title)
		var description string
		var extraContent string

		switch m.State {
		case kernel.MCPStateStarting:
			icon = t.ResourceBusyIcon.String()
			description = t.ResourceStatus.Render("starting...")
		case kernel.MCPStateConnected:
			icon = t.ResourceOnlineIcon.String()
			extraContent = mcpCounts(t, m)
		case kernel.MCPStateError:
			icon = t.ResourceErrorIcon.String()
			description = t.ResourceStatus.Render("error")
			if m.Error != nil {
				description = t.ResourceStatus.Render(fmt.Sprintf("error: %s", m.Error.Error()))
			}
		case kernel.MCPStateDisabled:
			icon = t.ResourceOfflineIcon.Foreground(t.Muted.GetBackground()).String()
			description = t.ResourceStatus.Render("disabled")
		default:
			icon = t.ResourceOfflineIcon.String()
		}

		renderedMcps = append(renderedMcps, common.Status(t, common.StatusOpts{
			Icon:         icon,
			Title:        title,
			Description:  description,
			ExtraContent: extraContent,
		}, width))
	}

	if len(renderedMcps) > maxItems {
		visibleItems := renderedMcps[:maxItems-1]
		remaining := len(renderedMcps) - maxItems
		visibleItems = append(visibleItems, t.ResourceAdditionalText.Render(fmt.Sprintf("…and %d more", remaining)))
		return lipgloss.JoinVertical(lipgloss.Left, visibleItems...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, renderedMcps...)
}
