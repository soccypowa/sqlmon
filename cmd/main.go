package main

import (
	"fmt"
	"os"

	"github.com/Soccypowa/sqlmon/Internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(ui.Newmodel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// p.Run() starts the event loop and blocks until tea.Quit is sent.
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
