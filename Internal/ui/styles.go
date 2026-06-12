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
var styleBad = lipgloss.NewStyle().Bold(true).Foreground(colorRed)

var styleLoading = lipgloss.NewStyle().Italic(true).Foreground(colorYellow)

var styleErrText = lipgloss.NewStyle().Bold(true).Foreground(colorRed)

var styleHeaderCell = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)

var styleCell = lipgloss.NewStyle().Foreground(colorWhite)

// Tab styles
var styleActiveTab = lipgloss.NewStyle().Bold(true).Foreground(colorBg).Background(colorCyan).Padding(0, 2)
var styleInactiveTab = lipgloss.NewStyle().Foreground(colorGray).Padding(0, 2)
var styleTabBar = lipgloss.NewStyle().Background(colorBgPanel).BorderBottom(true).BorderForeground(colorBorder)
