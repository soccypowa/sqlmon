package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/Soccypowa/sqlmon/Internal/db"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	// Lines consumed by chrome outside the viewport:
	//   1  header bar
	//   1  thin separator line
	//   1  footer bar
	//   2  viewport border (top + bottom)
	chromeHeight    = 6 // Borde vara 5 enligt exemplet
	refreshInterval = 5 * time.Second
)

// fetchDoneMsg is sent by fetchSessionsCmd when the query completes
// (successfully or not). Update() pattern-matches on this type.
type fetchDoneMsg struct {
	sessions  []db.Session
	err       error
	fetchedAt time.Time
}

// tickMsg is sent by the auto-refresh ticker every N seconds.
type tickMsg time.Time

// The model holds ALL state for the application.
// Bubble Tea re-renders the whole screen whenever the model changes,
// so anything the View() needs must live here.

type Model struct {
	width  int
	height int
	//data
	client    *db.Client
	sessions  []db.Session
	fetchedAt time.Time
	//stateflags
	loading  bool           // true while a query is in flight
	err      error          // last fetch error (nil = healthy)
	viewPort viewport.Model // scrollable content pane
}

// NewModel constructs the initial model.
func Newmodel(client *db.Client) Model {
	vp := viewport.New(0, 0) // real size set on first WindowSizeMsg

	return Model{
		client:   client,
		loading:  true,
		viewPort: vp,
	}
}

// Init() runs once at startup. It returns an optional tea.Cmd —
// a function that produces a Msg. Returning nil means "do nothing".

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchSessionCmd(m.client),
		tickCmd(),
	)
}

// fetchSessionsCmd returns a tea.Cmd that queries the DB in the
// background and delivers a fetchDoneMsg when it finishes.
//
// KEY IDEA: a tea.Cmd is just a func() tea.Msg. Bubble Tea runs it
// in a goroutine. When it returns, the Msg is sent into Update().
// The DB call can take 2 seconds and the UI stays fully responsive.

func fetchSessionCmd(client *db.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessions, err := client.GetSessions(ctx)
		return fetchDoneMsg{
			sessions:  sessions,
			err:       err,
			fetchedAt: time.Now(),
		}
	}
}

// tickCmd fires a tickMsg after refreshInterval.
// Update() re-arms it each time, creating a repeating timer.
//
// WHY NOT time.Ticker?
//   A time.Ticker runs on its own goroutine and would require a
//   channel + select. tea.Tick integrates neatly with the Msg loop.

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update() is called every time a Msg arrives (keyboard event,
// window resize, timer tick, network response, …).

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	//	Windows resize
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Resize the viewport to fill the available space.
		// The panel border adds 2 chars of horizontal padding on each
		// side, so we subtract 4 from width; chromeHeight accounts for
		// the rows consumed by header, separator, and footer.
		m.viewPort.Width = m.width - 4
		m.viewPort.Height = m.height - chromeHeight
		m.viewPort.SetContent(m.buildContent())

	// query result arrived
	case fetchDoneMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.sessions = msg.sessions
			m.fetchedAt = msg.fetchedAt
		}
		m.viewPort.SetContent(m.buildContent())

	// Refresher ticker fired
	case tickMsg:
		m.loading = true
		m.viewPort.SetContent(m.buildContent())
		cmds = append(cmds, fetchSessionCmd(m.client), tickCmd())

	// When a key is pressed
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r": //manual refresh
			m.loading = true
			m.viewPort.SetContent(m.buildContent())
			cmds = append(cmds, fetchSessionCmd(m.client))
		}
	}

	// Forward all other messages to the viewport so it can handle
	// scroll keys (↑ ↓ PgUp PgDn g G) itself.
	var vpCmd tea.Cmd
	m.viewPort, vpCmd = m.viewPort.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// Splitting "build the content string" from "render the frame"
// keeps View() clean and lets Update() refresh content without
// duplicating layout logic.
func (m Model) buildContent() string {
	// Error state
	if m.err != nil {
		return styleErrText.Render(fmt.Sprintf("\n   ✗  Error: %v\n\n  Press [r] to retry.", m.err))
	}
	// Loading state - first load, no rows yet
	if m.loading && len(m.sessions) == 0 {
		return styleLoading.Render("\n  ⟳  Fetching sessions…")
	}

	content := buildTableHeader(m.viewPort.Width)

	for _, s := range m.sessions {
		content += buildRow(s, m.viewPort.Width)
	}

	if len(m.sessions) == 0 {
		content += styleDim.Render("  (no user sessions)\n")
	}
	return content
}

// Column layout: fixed widths for all but the last (Query) column.
// We compute query width dynamically from the available viewport width.
var colWidths = []int{6, 11, 16, 14, 18, 9}

func buildTableHeader(viewWidth int) string {
	headers := []string{"SPID", "Status", "Login", "Database", "Program", "Wait ms"}
	row := ""
	totalFixed := 0

	for i, h := range headers {
		row += styleHeaderCell.Render(padRight(h, colWidths[i])) + " "
		totalFixed += colWidths[i] + 1
	}
	qw := viewWidth - totalFixed - 2
	if qw > 0 {
		row += styleHeader.Render(padRight("Query", qw))
	}
	row += "\n"
	row += styleSeparator.Render(repeatString("-", viewWidth-2)) + "\n"
	return row
}

func buildRow(s db.Session, viewWidth int) string {
	spidStr := padRight(fmt.Sprintf("%d", s.SPID), colWidths[0])
	statusStr := padRight(s.Status, colWidths[1])

	var statusStyled string
	switch s.Status {
	case "running":
		statusStyled = styleGood.Render(statusStr)
	case "suspended":
		statusStyled = styleWarn.Render(statusStr)
	default:
		statusStyled = styleDim.Render(statusStr)
	}

	var spidStyled string
	if s.BlockedBy > 0 {
		spidStyled = styleBad.Render(padRight(fmt.Sprintf("%d!", s.SPID), colWidths[0]))
	} else {
		spidStyled = styleCell.Render(spidStr)
	}

	totalFixed := 0
	for _, w := range colWidths {
		totalFixed += w + 1
	}
	qw := viewWidth - totalFixed - 2
	if qw < 0 {
		qw = 0
	}

	row := spidStyled + " " +
		statusStyled + " " +
		styleCell.Render(padRight(truncate(s.LoginName, colWidths[2]), colWidths[2])) + " " +
		styleDim.Render(padRight(truncate(s.DatabaseName, colWidths[3]), colWidths[3])) + " " +
		styleDim.Render(padRight(truncate(s.ProgramName, colWidths[4]), colWidths[4])) + " " +
		styleCell.Render(padRight(fmt.Sprintf("%d", s.WaitTimeMs), colWidths[5])) + " "

	if qw > 0 {
		row += styleDim.Render(truncate(cleanQuery(s.QueryText), qw))
	}

	return row + "\n"
}

// String helpers
func padRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "-"
}

func repeatString(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

// cleanQuery collapses whitespace/newlines so SQL text fits on one line.
func cleanQuery(s string) string {
	out := ""
	prevSpace := false

	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			if !prevSpace {
				out += " "
			}
			prevSpace = true
		} else {
			out += string(r)
			prevSpace = false
		}
	}
	return out
}

//var _ = fmt.SprintF
