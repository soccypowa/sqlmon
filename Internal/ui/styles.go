package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorBg      = lipgloss.Color("#0d1117") // near-black background
	colorBgPanel = lipgloss.Color("#161b22") // slightly lighter panels
	colorBorder  = lipgloss.Color("#30363d") // subtle border lines
	colorDimGray = lipgloss.Color("#2d3436") // dark bars (header/footer bg)
	colorGray    = lipgloss.Color("#636e72") // secondary / dim text
	colorWhite   = lipgloss.Color("#dfe6e9") // primary text
	colorCyan    = lipgloss.Color("#00d2d3") // accent / active elements
	colorGreen   = lipgloss.Color("#00ff87") // good / running
	colorYellow  = lipgloss.Color("#ffd32a") // warning
	colorRed     = lipgloss.Color("#ff4757") // error / blocked
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

// Status color styles
var styleGood = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
var styleWarn = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
var StyleCrit = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
