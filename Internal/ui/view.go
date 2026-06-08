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
		renderHeader(m.width),
		renderSeparator(m.width),
		renderViewport(m),
		renderFooter(m),
	}, "\n")

}

// KEY IDEA: lipgloss.Width() measures the *visible* width of a
// string, stripping ANSI escape codes. Always use it instead of
// len() when working with styled strings.

func renderHeader(width int) string {
	left := styleTitle.Render(" ⬡ SQL SERVER ACTIVITY MONITOR")
	right := styleSubtitle.Render("Lesson 2 style layout")

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	content := left + strings.Repeat(" ", gap) + right

	return styleHeader.Width(width).Render(content)
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

func renderViewport(m Model) string {
	// The scroll percentage shown in the top-right corner of the border.
	pct := fmt.Sprintf("%3.f%%", m.viewPort.ScrollPercent()*100)
	pctStyled := styleDim.Render(pct)

	return stylePanel.Width(m.width - 2).Render(
		m.viewPort.View() + "\n" + strings.Repeat(" ", m.viewPort.Width-lipgloss.Width(pctStyled)) + pctStyled,
	)
}

func renderFooter(m Model) string {
	help := " [q] quit [↑↓ / PgUp PgDn] scroll [g/G] top/bottom"
	size := fmt.Sprintf("terminal: %d x %d", m.width, m.height)

	gap := m.width - lipgloss.Width(help) - lipgloss.Width(size)
	if gap < 0 {
		gap = 0
	}
	content := help + strings.Repeat(" ", gap) + size
	return styleFooter.Width(m.width).Render(content)
}
