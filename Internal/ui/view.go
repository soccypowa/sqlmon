package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View() converts the model into a string. Bubble Tea prints that
// string to the terminal on every update. Think of it as a pure
// function: same model → same output, no side effects.

func (m Model) View() string {
	// If terminal has not reported the size do nothing
	if m.width == 0 {
		return ""
	}

	panelInner := m.renderTabBar() + "\n" + m.viewPort.View()
	panel := stylePanel.Width(m.width - 2).Render(panelInner)

	return strings.Join([]string{
		m.renderHeader(),
		m.renderMetricsBar(),
		renderSeparator(m.width),
		panel,
		m.renderFooter(),
	}, "\n")

}

// KEY IDEA: lipgloss.Width() measures the *visible* width of a
// string, stripping ANSI escape codes. Always use it instead of
// len() when working with styled strings.

func (m Model) renderHeader() string {
	left := styleTitle.Render(" ⬡ SQL SERVER ACTIVITY MONITOR")

	if m.serverName != "" {
		left += styleSubtitle.Render(" " + m.serverName)
	}

	// Live status indicator
	var statusIndicator string
	if m.loading {
		statusIndicator = styleLoading.Render("  ⟳ refreshing…")
	} else if m.err != nil {
		statusIndicator = styleErrText.Render("  ✗ error")
	} else if !m.fetchedAt.IsZero() {
		ts := m.fetchedAt.Format("15:04:05")
		statusIndicator = styleDim.Render("  ✓ updated " + ts)
	}

	left += statusIndicator
	right := styleSubtitle.Render("sqlmon  ")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return styleHeader.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right,
	)
}

// Renders a single line of key health counters:
//
//   CPU  42% [████████░░░░]  │  Sessions 38  │  Active 5  │  Blocked 0  │  PLE 4821s  │  Batch/s 1204
//
// DESIGN DECISIONS
//
//   1. One line, not a box.
//      The metrics bar is scanned quickly — a dense single line is
//      faster to read than a bordered panel. It sits between the
//      header and the data panel so it's always visible regardless
//      of which tab is active or how far the user has scrolled.
//
//   2. Colour thresholds encode the "normal / watch / act" states.
//      Green  → healthy, no action needed
//      Yellow → worth watching, may need attention
//      Red    → act now — threshold functions in styles.go
//
//   3. The CPU bar gives magnitude at a glance without reading a number.
//      The % label is there for precision; the bar is for fast scanning.

func (m Model) renderMetricsBar() string {
	mx := m.metrics

	sep := styleMetricSep.Render("  |  ")

	cpuNum := cpuStyle(mx.SQLCPUPct).Render(fmt.Sprintf("%3d%%", mx.SQLCPUPct))
	cpuBarS := cpuBar(mx.SQLCPUPct, 12)
	cpuPart := styleMetricsLabel.Render("cpu") + cpuNum + " [" + cpuBarS + "]"

	sessPart := styleMetricsLabel.Render("Sessions ") +
		styleCell.Render(fmt.Sprintf("%d", mx.TotalSessions))

	activPart := styleMetricsLabel.Render("Active ") +
		styleCell.Render(fmt.Sprintf("%d", mx.ActiveRequests))

	blockedPart := styleMetricsLabel.Render("Blocked ") +
		blockStyle(mx.BlockedSessions).Render(fmt.Sprintf("%d", mx.BlockedSessions))

	plePart := styleMetricsLabel.Render("PLE ") +
		pleStyle(mx.PageLifeExp).Render(fmt.Sprintf("%ds", mx.PageLifeExp))

	batchPart := styleMetricsLabel.Render("Batch/s ") +
		styleCell.Render(fmt.Sprintf("%.0f", mx.BatchReqPerSec))

	content := " " + cpuPart +
		sep + sessPart +
		sep + activPart +
		sep + blockedPart +
		sep + plePart +
		sep + batchPart

	return styleMetricBar.Width(m.width).Render(content)
}

// Renders the three tab labels using active/inactive styles.
// The active tab uses an inverted colour scheme (dark text on cyan)
// so it stands out clearly.

func (m Model) renderTabBar() string {
	labels := []string{
		"[1] Processes",
		"[2] Wait Stats",
		"[3] Expensive Queries",
	}
	var rendered []string
	for i, label := range labels {
		if Tab(i) == m.activeTab {
			rendered = append(rendered, styleActiveTab.Render(label))
		} else {
			rendered = append(rendered, styleInactiveTab.Render(label))
		}
	}

	// Underline separator below the tabs
	tabRow := strings.Join(rendered, "")
	underLine := styleSeparator.Render(strings.Repeat("-", m.width-4))
	return tabRow + "\n" + underLine
}

func renderSeparator(width int) string {
	return styleSeparator.Render(strings.Repeat("-", width))
}

func (m Model) renderFooter() string {
	help := fmt.Sprintf("  [q] quit   [r] refresh   [tab/1-3] switch   [↑↓] scroll   auto: %s", m.interval.String())
	//size := fmt.Sprintf("terminal: %d x %d", m.width, m.height)

	var info string
	switch m.activeTab {
	case TabProcesses:
		blocked := 0
		for _, s := range m.sessions {
			if s.BlockedBy > 0 {
				blocked++
			}
		}
		if blocked > 0 {
			info = fmt.Sprintf("sessions %d %s ",
				len(m.sessions),
				styleBad.Render(fmt.Sprintf("blocked: %d", blocked)))
		} else {
			info = fmt.Sprintf("sessions: %d ", len(m.sessions))
		}
	case TabWaits:
		info = fmt.Sprintf("wait types: %d ", len(m.waits))
	case TabExpensive:
		info = fmt.Sprintf("cached plans: %d ", len(m.queries))
	}

	gap := m.width - lipgloss.Width(help) - lipgloss.Width(info)
	if gap < 0 {
		gap = 0
	}
	return styleFooter.Width(m.width).Render(help + strings.Repeat(" ", gap) + info)
}
