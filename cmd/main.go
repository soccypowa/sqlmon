package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// The model holds ALL state for the application.
// Bubble Tea re-renders the whole screen whenever the model changes,
// so anything the View() needs must live here.

type model struct {
	width  int
	height int
}

// Init() runs once at startup. It returns an optional tea.Cmd —
// a function that produces a Msg. Returning nil means "do nothing".

func (m model) Init() tea.Cmd {
	return nil
}

// Update() is called every time a Msg arrives (keyboard event,
// window resize, timer tick, network response, …).

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	//Everytime the window resizes
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	// When a key is presse
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View() converts the model into a string. Bubble Tea prints that
// string to the terminal on every update. Think of it as a pure
// function: same model → same output, no side effects.

func (m model) View() string {
	// If terminal has not reported the size do nothing
	if m.width == 0 {
		return ""
	}

	// The screen is a slice of lines that we joing
	var lines []string

	header := centerText("SQL SERVER ACTIVITY MOITOR", m.width)
	lines = append(lines, header)

	lines = append(lines, strings.Repeat("-", m.width))

	// We reserve 2 lines for the header+rule and 1 for the footer.
	bodyHeight := m.height - 3
	if bodyHeight < 0 {
		bodyHeight = 0
	}

	// placeholde
	midLine := bodyHeight / 2
	for i := 0; i < bodyHeight; i++ {
		if i == midLine {
			lines = append(lines, centerText("No data yet - lessons to come", m.width))
		} else {
			lines = append(lines, "")
		}
	}

	footer := fmt.Sprintf(" [q] quit.   terminal: %d x%d", m.width, m.height)
	lines = append(lines, footer)

	return strings.Join(lines, "\n")
}

func centerText(s string, w int) string {
	if len(s) >= w {
		return s
	}

	padding := (w - len(s)) / 2
	return strings.Repeat(" ", padding) + s
}

func main() {
	intialModel := model{}

	// tea.NewProgram wraps the model and wires up the event loop.
	//
	// tea.WithAltScreen() switches to the "alternate screen buffer" —
	// a second screen that the terminal provides for full-screen apps.
	// When the program exits, the original shell output is restored
	// automatically, leaving no messy output in the scroll-back buffer.
	p := tea.NewProgram(intialModel, tea.WithAltScreen())

	// p.Run() starts the event loop and blocks until tea.Quit is sent.
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
