package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	// Lines consumed by chrome outside the viewport:
	//   1  header bar
	//   1  thin separator line
	//   1  footer bar
	//   2  viewport border (top + bottom)
	chromeHeight = 6 // Borde vara 5 enligt exemplet
)

// The model holds ALL state for the application.
// Bubble Tea re-renders the whole screen whenever the model changes,
// so anything the View() needs must live here.

type Model struct {
	width    int
	height   int
	viewPort viewport.Model // scrollable content pane
}

// NewModel constructs the initial model.
func Newmodel() Model {
	vp := viewport.New(0, 0) // real size set on first WindowSizeMsg
	vp.SetContent(placeholderContent())
	return Model{viewPort: vp}
}

// Init() runs once at startup. It returns an optional tea.Cmd —
// a function that produces a Msg. Returning nil means "do nothing".

func (m Model) Init() tea.Cmd {
	return nil
}

// Update() is called every time a Msg arrives (keyboard event,
// window resize, timer tick, network response, …).

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	//Everytime the window resizes
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Resize the viewport to fill the available space.
		// The panel border adds 2 chars of horizontal padding on each
		// side, so we subtract 4 from width; chromeHeight accounts for
		// the rows consumed by header, separator, and footer.
		m.viewPort.Width = m.width - 4
		m.viewPort.Height = m.height - chromeHeight

	// When a key is presse
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	// Forward all other messages to the viewport so it can handle
	// scroll keys (↑ ↓ PgUp PgDn g G) itself.
	var vpCmd tea.Cmd
	m.viewPort, vpCmd = m.viewPort.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func placeholderContent() string {
	lines := "  SPID   Status      Login            Database        Wait Type\n"
	lines += "  ─────  ──────────  ───────────────  ──────────────  ────────────\n"
	statuses := []string{"running", "suspended", "sleeping", "sleeping", "suspended"}
	logins := []string{"sa", "appuser", "reportuser", "appuser", "svc_etl"}
	dbs := []string{"master", "AppDB", "ReportDB", "AppDB", "DWH"}
	waits := []string{"", "LCK_M_S", "", "", "ASYNC_NETWORK_IO"}
	for i := 1; i <= 30; i++ {
		idx := i % len(statuses)
		lines += "  " + padRight(itoa(50+i), 7) +
			padRight(statuses[idx], 12) +
			padRight(logins[idx], 17) +
			padRight(dbs[idx], 16) +
			waits[idx] + "\n"
	}
	return lines
}

func padRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 30
	}
	return digits
}
