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

	// Build normal screen
	panelInner := m.renderTabBar() + "\n" + m.viewPort.View()
	panel := stylePanel.Width(m.width - 2).Render(panelInner)

	screen := strings.Join([]string{
		m.renderHeader(),
		m.renderMetricsBar(),
		panel,
		m.renderFooter(),
	}, "\n")

	// If popup is open
	if m.showPopup && len(m.sessions) > 0 {
		screen = m.renderPopup(screen)
	}

	return screen
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

// ------------------------------------------------------------------
// Popup overlay
// ------------------------------------------------------------------
//
// HOW lipgloss.Place() WORKS
//
//   lipgloss.Place(
//       totalWidth, totalHeight,   // the "canvas" size
//       lipgloss.Center,           // horizontal alignment within canvas
//       lipgloss.Center,           // vertical alignment within canvas
//       popupString,               // the content to place
//       lipgloss.WithWhitespaceForeground(colorDimGray), // fill colour
//   )
//
// Place() returns a string of exactly totalWidth × totalHeight that
// has the popup centred and everything else filled with spaces (or
// whatever whitespace style you provide). When printed, those spaces
// overwrite the screen content behind the popup, creating the
// "modal overlay" effect without needing a real layering system.

func (m Model) renderPopup(behind string) string {
	var title, details, rawQuery string

	switch m.activeTab {
	case TabProcesses:
		s := m.sessions[m.selectedRow]

		title = stylePopupTitle.Render(fmt.Sprintf("SPID %d  ·  %s  ·  %s", s.SPID, s.Status, s.DatabaseName))
		type field struct{ label, value string }
		fields := []field{
			{"Login", s.LoginName},
			{"Host", s.HostName},
			{"Program", s.ProgramName},
			{"Database", s.DatabaseName},
			{"Wait Type", s.WaitType},
			{"Wait Ms", fmt.Sprintf("%d", s.WaitTimeMs)},
			{"CPU Ms", fmt.Sprintf("%d", s.CPUTime)},
			{"Elapsed Ms", fmt.Sprintf("%d", s.ElapsedMs)},
			{"Blocked by", blockedByLabel(s.BlockedBy)},
		}
		labelW := 12 // fixed label column width
		for _, f := range fields {
			if f.value == "" {
				continue
			}
			label := styleDim.Render(padRight(f.label, labelW))
			val := styleCell.Render(f.value)
			details += label + " " + val + "\n"
		}

		rawQuery = cleanQuery(s.QueryText)

	case TabExpensive:
		q := m.queries[m.selectedRow]
		title = fmt.Sprintf("Rank #%d  ·  %d executions  ·  avg %.1f ms CPU", m.selectedRow+1, q.ExecutionCount, q.AvgCPUMs)
		type field struct{ label, value string }
		fields := []field{
			{"Total CPU ms", fmt.Sprintf("%d", q.TotalCPUMs)},
			{"Total elapsed", fmt.Sprintf("%d ms", q.TotalElapsedMs)},
			{"Avg CPU ms", fmt.Sprintf("%.1f", q.AvgCPUMs)},
			{"Executions", fmt.Sprintf("%d", q.ExecutionCount)},
			{"Last run", q.LastExecution.Format("2006-01-02 15:04:05")},
		}
		labelW := 14
		for _, f := range fields {
			label := styleDim.Render(padRight(f.label, labelW))
			val := styleCell.Render(f.value)
			details += label + " " + val + "\n"

		}
		rawQuery = cleanQuery(q.QueryText)
	}

	// Popup width: 70% of terminal, clamped to 40-120 chars.
	popW := m.width * 70 / 100
	if popW < 40 {
		popW = 40
	}
	if popW > 120 {
		popW = 120
	}
	// Inner width available for text (subtract border + padding).
	innerW := popW - 6 // 2 border + 2*2 padding + 2 cursor margin

	devider := styleSeparator.Render(repeatString("-", innerW))

	if rawQuery == "" {
		rawQuery = "(no query text available)"
	}
	wrapped := wordWrap(rawQuery, innerW)
	queryString := stylePopupContent.Render(wrapped)

	hint := stylePopUpHint.Render("[Esc] or [Enter] to close")

	body := strings.Join([]string{title, details, devider, queryString, hint}, "\n")

	popup := stylePopupBox.Width(popW).Render(body)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popup)
}

func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	words := strings.Fields(s)
	line := ""
	result := ""
	for _, w := range words {
		if line == "" {
			line = w
		} else if len(line)+1+len(w) <= width {
			line += " " + w
		} else {
			result += line + "\n"
			line = w
		}
	}
	if line != "" {
		result += line
	}
	return result
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

func (m Model) renderFooter() string {
	var help string
	switch m.activeTab {
	case TabProcesses:
		help = fmt.Sprintf("  [q] quit   [r] refresh   [tab/1-3] switch   [↑↓] select  [Enter] detail   [←→] hscroll   auto: %s", m.interval.String())
	case TabExpensive:
		help = fmt.Sprintf("  [q] quit   [r] refresh   [tab/1-3] switch   [↑↓] select  [Enter] detail  auto: %s", m.interval.String())
	default:
		help = fmt.Sprintf("  [q] quit   [r] refresh   [tab/1-3] switch   [↑↓] scroll    auto: %s", m.interval.String())
	}
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
