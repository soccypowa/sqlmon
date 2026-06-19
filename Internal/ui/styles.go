package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorBg       = lipgloss.Color("#0d1117") // near-black background
	colorBgPanel  = lipgloss.Color("#161b22") // slightly lighter panels
	colorBorder   = lipgloss.Color("#30363d") // subtle border lines
	colorDimGray  = lipgloss.Color("#2d3436") // dark bars (header/footer bg)
	colorGray     = lipgloss.Color("#636e72") // secondary / dim text
	colorWhite    = lipgloss.Color("#dfe6e9") // primary text
	colorCyan     = lipgloss.Color("#00d2d3") // accent / active elements
	colorGreen    = lipgloss.Color("#00ff87") // good / running
	colorYellow   = lipgloss.Color("#ffd32a") // warning
	colorRed      = lipgloss.Color("#ff4757") // error / blocked
	colorSelected = lipgloss.Color("#1c2d40")
)

// Structual styles
var styleHeader = lipgloss.NewStyle().Background(colorDimGray).Foreground(colorWhite)
var styleFooter = lipgloss.NewStyle().Background(colorDimGray).Foreground(colorGray)
var stylePanel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Background(colorBgPanel)

// Text styles
var styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
var styleSubtitle = lipgloss.NewStyle().Foreground(colorGray)
var styleDim = lipgloss.NewStyle().Foreground(colorGray)
var styleSeparator = lipgloss.NewStyle().Foreground(colorBorder)
var styleLoading = lipgloss.NewStyle().Italic(true).Foreground(colorYellow)
var styleErrText = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
var styleHeaderCell = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
var styleCell = lipgloss.NewStyle().Foreground(colorWhite)
var styleSelected = lipgloss.NewStyle().Background(colorSelected)

// Status color styles
var styleGood = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
var styleWarn = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
var styleBad = lipgloss.NewStyle().Bold(true).Foreground(colorRed)

// Tab styles
var styleActiveTab = lipgloss.NewStyle().Bold(true).Foreground(colorBg).Background(colorCyan).Padding(0, 2)
var styleInactiveTab = lipgloss.NewStyle().Foreground(colorGray).Padding(0, 2)

// Metrics bar
var styleMetricsLabel = lipgloss.NewStyle().Foreground(colorGray)
var styleMetricSep = lipgloss.NewStyle().Foreground(colorBorder)
var styleMetricBar = lipgloss.NewStyle().Background(colorDimGray).Padding(0, 1)

// Popup styles
var stylePopupBox = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).
	BorderForeground(colorCyan).Background(colorBgPanel).
	Padding(1, 2)

var stylePopupTitle = lipgloss.NewStyle().Bold(true).Foreground(colorCyan).
	MarginBottom(1)

var stylePopupContent = lipgloss.NewStyle().Foreground(colorWhite)

var stylePopUpHint = lipgloss.NewStyle().Foreground(colorCyan).MarginTop(1)

// Threshold aware helpers
func cpuStyle(pct int) lipgloss.Style {
	switch {
	case pct >= 80:
		return styleBad
	case pct >= 50:
		return styleWarn
	default:
		return styleGood
	}
}

func pleStyle(secs int64) lipgloss.Style {
	switch {
	case secs < 300:
		return styleBad
	case secs < 1000:
		return styleWarn
	default:
		return styleGood
	}
}

func blockStyle(n int) lipgloss.Style {
	if n > 0 {
		return styleBad
	}
	return styleGood
}

// Example for 65%, width 12:
//   ████████░░░░   (8 filled, 4 empty, yellow)

func cpuBar(pct, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := (pct * width) / 100

	var filledColor lipgloss.Color
	switch {
	case pct >= 80:
		filledColor = colorRed
	case pct >= 50:
		filledColor = colorYellow
	default:
		filledColor = colorGreen
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += lipgloss.NewStyle().Foreground(filledColor).Render("█")
		} else {
			bar += lipgloss.NewStyle().Foreground(colorDimGray).Render("░")
		}
	}
	return bar
}
