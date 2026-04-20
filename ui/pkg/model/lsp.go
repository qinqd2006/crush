package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/qinqd2006/crush/kernel/pkg"
	"github.com/qinqd2006/crush/ui/pkg/common"
	"github.com/qinqd2006/crush/ui/pkg/styles"
)

// lspInfo renders the LSP status section showing active LSP clients and their
// diagnostic counts.
func (m *UI) lspInfo(width, maxItems int, isSection bool) string {
	t := m.com.Styles

	states := slices.SortedFunc(maps.Values(m.lspStates), func(a, b kernel.LSPClientInfo) int {
		return strings.Compare(a.Name, b.Name)
	})

	title := t.ResourceGroupTitle.Render("LSPs")
	if isSection {
		title = common.Section(t, title, width)
	}
	list := t.ResourceAdditionalText.Render("None")
	if len(states) > 0 {
		list = lspList(t, states, width, maxItems)
	}

	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}

// lspDiagnostics formats diagnostic counts with appropriate icons and colors.
func lspDiagnostics(t *styles.Styles, diagnostics kernel.DiagnosticCounts) string {
	var errs []string
	if diagnostics.Error > 0 {
		errs = append(errs, t.LSP.ErrorDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPErrorIcon, diagnostics.Error)))
	}
	if diagnostics.Warning > 0 {
		errs = append(errs, t.LSP.WarningDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPWarningIcon, diagnostics.Warning)))
	}
	if diagnostics.Hint > 0 {
		errs = append(errs, t.LSP.HintDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPHintIcon, diagnostics.Hint)))
	}
	if diagnostics.Information > 0 {
		errs = append(errs, t.LSP.InfoDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPInfoIcon, diagnostics.Information)))
	}
	return strings.Join(errs, " ")
}

// lspList renders a list of LSP clients with their status and diagnostics,
// truncating to maxItems if needed.
func lspList(t *styles.Styles, lsps []kernel.LSPClientInfo, width, maxItems int) string {
	if maxItems <= 0 {
		return ""
	}
	var renderedLsps []string
	for _, l := range lsps {
		var icon string
		title := t.ResourceName.Render(l.Name)
		var description string
		var diagnostics string
		switch l.State {
		case kernel.LSPStateUnstarted:
			icon = t.ResourceOfflineIcon.String()
			description = t.ResourceStatus.Render("unstarted")
		case kernel.LSPStateStarting:
			icon = t.ResourceBusyIcon.String()
			description = t.ResourceStatus.Render("starting...")
		case kernel.LSPStateRunning:
			icon = t.ResourceOnlineIcon.String()
			diagnostics = lspDiagnostics(t, l.Diagnostics)
		case kernel.LSPStateError:
			icon = t.ResourceErrorIcon.String()
			description = t.ResourceStatus.Render("error")
			if l.Error != nil {
				description = t.ResourceStatus.Render(fmt.Sprintf("error: %s", l.Error.Error()))
			}
		default:
			icon = t.ResourceOfflineIcon.String()
			description = t.ResourceStatus.Render("unknown")
		}
		renderedLsps = append(renderedLsps, common.Status(t, common.StatusOpts{
			Icon:         icon,
			Title:        title,
			Description:  description,
			ExtraContent: diagnostics,
		}, width))
	}

	if len(renderedLsps) > maxItems {
		visibleItems := renderedLsps[:maxItems-1]
		remaining := len(renderedLsps) - maxItems
		visibleItems = append(visibleItems, t.ResourceAdditionalText.Render(fmt.Sprintf("…and %d more", remaining)))
		return lipgloss.JoinVertical(lipgloss.Left, visibleItems...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, renderedLsps...)
}
