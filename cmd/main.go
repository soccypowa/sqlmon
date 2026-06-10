package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Soccypowa/sqlmon/Internal/db"
	"github.com/Soccypowa/sqlmon/Internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	connStr := flag.String("conn", "", "SQL Server connection string\n"+
		" e.g. sqlserver://sa:password@localhos\n"+
		" or set SQLMON_CONN environment variable")
	flag.Parse()

	//fallback to env variable so connstr is not mandatory in dev
	if *connStr == "" {
		*connStr = os.Getenv("SQLMON_CONN")
	}

	if *connStr == "" {
		fmt.Fprintln(os.Stderr, "Error: provide --conn or set SQLMON_CONN")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, `  go run ./cmd --conn "sqlserver://sa:password@localhost"`)
		fmt.Fprintln(os.Stderr, `  go run ./cmd --conn "sqlserver://myserver?trusted_connection=true"`)
		os.Exit(1)
	}

	// Connect — this is blocking and intentionally runs before the
	// TUI starts. A clear error here is much friendlier than a
	// confusing error inside the running UI.
	fmt.Fprintln(os.Stderr, "connecting...")
	client, err := db.New(*connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCould not connect: %v\n\n", err)
		fmt.Fprintln(os.Stderr, "Things to check:")
		fmt.Fprintln(os.Stderr, "  • SQL Server is running and reachable")
		fmt.Fprintln(os.Stderr, "  • Username / password are correct")
		fmt.Fprintln(os.Stderr, "  • Login has VIEW SERVER STATE permission:")
		fmt.Fprintln(os.Stderr, "      GRANT VIEW SERVER STATE TO [yourlogin];")
		os.Exit(1)
	}
	defer client.Close()

	p := tea.NewProgram(
		ui.Newmodel(client),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// p.Run() starts the event loop and blocks until tea.Quit is sent.
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
