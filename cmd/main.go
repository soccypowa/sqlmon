package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Soccypowa/sqlmon/Internal/db"
	"github.com/Soccypowa/sqlmon/Internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	connStr := flag.String("conn", "", "SQL Server connection string\n"+
		"  sqlserver://user:password@host[:port][/instance]\n"+
		"  sqlserver://host?trusted_connection=true  (Windows auth)")
	interval := flag.Duration("interval", 5*time.Second,
		"How often to refresh data (e.g. 5s, 10s, 30s).")
	flag.Parse()

	if *connStr == "" {
		fmt.Fprintln(os.Stderr, "Error: provide --conn")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, `  sqlmon --conn "sqlserver://sa:password@localhost"`)
		fmt.Fprintln(os.Stderr, `  sqlmon --conn "sqlserver://myserver:1433" --interval 10s`)
		fmt.Fprintln(os.Stderr, `  sqlmon --conn "sqlserver://myserver?trusted_connection=true"`)
		os.Exit(1)
	}

	// Derive servername
	name := extractHost(*connStr)

	// Connect — this is blocking and intentionally runs before the
	// TUI starts. A clear error here is much friendlier than a
	// confusing error inside the running UI.
	fmt.Fprintf(os.Stderr, "connecting to %s...\n", name)
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
		ui.Newmodel(client, name, *interval),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// p.Run() starts the event loop and blocks until tea.Quit is sent.
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func extractHost(connStr string) string {
	s := connStr

	// Strip Scheme
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}

	// Strip user:pass@ - find the last @ before any / or ?
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '@' {
			s = s[i+1:]
			break
		}
		if s[i] == '/' || s[i] == '?' {
			break
		}
	}
	// Strip query string
	if i := strings.IndexByte(s, '?'); i >= 0 {
		s = s[:i]
	}

	if s == "" {
		return "SQL Server"
	}
	return s
}
