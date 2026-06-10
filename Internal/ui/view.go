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

	return strings.Join([]string{
		m.renderHeader(),
		renderSeparator(m.width),
		m.renderViewport(),
		m.renderFooter(),
	}, "\n")

}

// KEY IDEA: lipgloss.Width() measures the *visible* width of a
// string, stripping ANSI escape codes. Always use it instead of
// len() when working with styled strings.

func (m Model) renderHeader() string {
	left := styleTitle.Render(" ⬡ SQL SERVER ACTIVITY MONITOR")

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

	left = left + statusIndicator
	right := styleSubtitle.Render("Lesson 2 style layout")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return styleHeader.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right,
	)
}

func renderSeparator(width int) string {
	return styleSeparator.Render(strings.Repeat("-", width))
}

// The viewport bubble handles all scroll logic internally.
// We just wrap it with the panel border style.
//
// KEY IDEA: stylePanel.Render(m.viewport.View()) layers the rounded
// border on top of whatever string the viewport returns. This is the
// core Lipgloss composition pattern — styles wrap other content.

func (m Model) renderViewport() string {
	// The scroll percentage shown in the top-right corner of the border.
	pct := fmt.Sprintf("%3.f%%", m.viewPort.ScrollPercent()*100)
	pctStyled := styleDim.Render(pct)

	return stylePanel.Width(m.width - 2).Render(
		m.viewPort.View() + "\n" + strings.Repeat(" ", m.viewPort.Width-lipgloss.Width(pctStyled)) + pctStyled,
	)
}

func (m Model) renderFooter() string {
	help := " [q] quit [↑↓ / PgUp PgDn] scroll [g/G] top/bottom"
	size := fmt.Sprintf("terminal: %d x %d", m.width, m.height)

	gap := m.width - lipgloss.Width(help) - lipgloss.Width(size)
	if gap < 0 {
		gap = 0
	}
	content := help + strings.Repeat(" ", gap) + size
	return styleFooter.Width(m.width).Render(content)
}
