package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/Soccypowa/sqlmon/Internal/db"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Tab int

const (
	TabProcesses Tab = iota
	TabWaits
	TabExpensive
	tabCount // Always last used for wrap-around arithemtic
)

func (t Tab) String() string {
	return [...]string{"Processes", "Wait Stats", "Expensive Queries"}[t]
}

const (
	// Lines consumed by chrome outside the viewport:
	//   1  header bar
	//   1  thin separator line
	//   1  footer bar
	//   2  viewport border (top + bottom)
	chromeHeight = 10 // Borde vara 9 enligt exemplet
)

// fetchDoneMsg is sent by fetchSessionsCmd when the query completes
// (successfully or not). Update() pattern-matches on this type.
type fetchDoneMsg struct {
	sessions  []db.Session
	waits     []db.WaitStat
	queries   []db.ExpensiveQuery
	metrics   db.ServerMetrics
	err       error
	fetchedAt time.Time
}

// tickMsg is sent by the auto-refresh ticker every N seconds.
type tickMsg time.Time

// The model holds ALL state for the application.
// Bubble Tea re-renders the whole screen whenever the model changes,
// so anything the View() needs must live here.

type Model struct {
	width     int
	height    int
	viewPort  viewport.Model // scrollable content pane
	activeTab Tab
	//data
	sessions []db.Session
	waits    []db.WaitStat
	queries  []db.ExpensiveQuery
	metrics  db.ServerMetrics
	//config
	client     *db.Client
	serverName string
	interval   time.Duration
	//stateflags
	fetchedAt time.Time
	loading   bool  // true while a query is in flight
	err       error // last fetch error (nil = healthy)
}

// NewModel constructs the initial model.
func Newmodel(client *db.Client, serverName string, interval time.Duration) Model {
	return Model{
		client:     client,
		serverName: serverName,
		interval:   interval,
		loading:    true,
		viewPort:   viewport.New(0, 0),
	}
}

// Init() runs once at startup. It returns an optional tea.Cmd —
// a function that produces a Msg. Returning nil means "do nothing".

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchAllCmd(m.client),
		tickCmd(m.interval),
	)
}

// KEY IDEA: a tea.Cmd is just a func() tea.Msg. Bubble Tea runs it
// in a goroutine. When it returns, the Msg is sent into Update().
// The DB call can take 2 seconds and the UI stays fully responsive.

// fetchAllCmd — now calls four queries, still in one goroutine.
//
// TRADE-OFF DISCUSSION (as promised in Lesson 4):
//
// Sequential (what we do):
//   Simple, one Msg, atomic UI update. Fine when queries take <100 ms
//   each — which is true for these DMV queries on a healthy server.
//
// Parallel alternative (what you'd do if one query was slow):
//
//   type sessResult struct { data []db.Session; err error }
//   type metricsResult struct { data db.ServerMetrics; err error }
//
//   func fetchAllCmd(client *db.Client) tea.Cmd {
//       return func() tea.Msg {
//           sessCh    := make(chan sessResult, 1)
//           metricsCh := make(chan metricsResult, 1)
//
//           go func() { s, err := client.GetSessions(ctx); sessCh <- sessResult{s, err} }()
//           go func() { m, err := client.GetMetrics(ctx);  metricsCh <- metricsResult{m, err} }()
//
//           sr := <-sessCh
//           mr := <-metricsCh
//           return fetchDoneMsg{sessions: sr.data, metrics: mr.data, ...}
//       }
//   }
//
// For this app, sequential is the right default. Add parallelism only
// when profiling shows a specific query is the bottleneck.

func fetchAllCmd(client *db.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessions, err := client.GetSessions(ctx)
		if err != nil {
			return fetchDoneMsg{err: err, fetchedAt: time.Now()}
		}

		waits, err := client.GetWaitStats(ctx)
		if err != nil {
			return fetchDoneMsg{err: err, fetchedAt: time.Now()}
		}

		queries, err := client.GetExpensiveQueries(ctx)
		if err != nil {
			return fetchDoneMsg{err: err, fetchedAt: time.Now()}
		}
		metrics, err := client.GetMetrics(ctx)
		if err != nil {
			return fetchDoneMsg{err: err, fetchedAt: time.Now()}
		}

		return fetchDoneMsg{
			sessions:  sessions,
			waits:     waits,
			queries:   queries,
			metrics:   metrics,
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

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
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
			m.waits = msg.waits
			m.queries = msg.queries
			m.metrics = msg.metrics
			m.fetchedAt = msg.fetchedAt
		}
		m.viewPort.SetContent(m.buildContent())

	// Refresher ticker fired
	case tickMsg:
		m.loading = true
		m.viewPort.SetContent(m.buildContent())
		cmds = append(cmds, fetchAllCmd(m.client), tickCmd(m.interval))

	// When a key is pressed
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r": //manual refresh
			m.loading = true
			m.viewPort.SetContent(m.buildContent())
			cmds = append(cmds, fetchAllCmd(m.client))
		case "tab", "right":
			m.activeTab = (m.activeTab + 1) % tabCount
			m.viewPort.GotoTop()
			m.viewPort.SetContent(m.buildContent())
		case "shift+tab", "left":
			m.activeTab = (m.activeTab + tabCount - 1) % tabCount
			m.viewPort.GotoTop()
			m.viewPort.SetContent(m.buildContent())
		case "1":
			m.setTab(TabProcesses)
		case "2":
			m.setTab(TabWaits)
		case "3":
			m.setTab(TabExpensive)
		}
	}

	// Forward all other messages to the viewport so it can handle
	// scroll keys (↑ ↓ PgUp PgDn g G) itself.
	var vpCmd tea.Cmd
	m.viewPort, vpCmd = m.viewPort.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) setTab(t Tab) {
	m.activeTab = t
	m.viewPort.GotoTop()
	m.viewPort.SetContent(m.buildContent())
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

	switch m.activeTab {
	case TabProcesses:
		return buildProcessTable(m.sessions, m.viewPort.Width)
	case TabWaits:
		return buildWaitsTable(m.waits, m.viewPort.Width)
	case TabExpensive:
		return buildExpensiveTable(m.queries, m.viewPort.Width)
	}

	return ""
}

// Column layout: fixed widths for all but the last (Query) column.
// We compute query width dynamically from the available viewport width.
var processColW = []int{6, 11, 16, 14, 18, 10, 9}

func buildProcessTable(sessions []db.Session, vw int) string {
	headers := []string{"SPID", "Status", "Login", "Database", "Program", "Wait Type", "Wait ms"}
	out := tableHeader(headers, processColW, vw) + tableRule(vw)

	if len(sessions) == 0 {
		return out + styleDim.Render(" (no user sessions)\n")
	}
	for _, s := range sessions {
		spidStr := padRight(fmt.Sprintf("%d", s.SPID), processColW[0])
		var spidStyled string
		if s.BlockedBy > 0 {
			spidStyled = styleBad.Render(padRight(fmt.Sprintf("%d!", s.SPID), processColW[0]))
		} else {
			spidStyled = styleCell.Render(spidStr)
		}

		statusStr := padRight(s.Status, processColW[1])
		var statusStyled string
		switch s.Status {
		case "running":
			statusStyled = styleGood.Render(statusStr)
		case "suspended":
			statusStyled = styleWarn.Render(statusStr)
		default:
			statusStyled = styleDim.Render(statusStr)
		}

		waitstr := padRight(truncate(s.WaitType, processColW[5]), processColW[5])
		var waitStyled string
		if s.WaitType != "" {
			waitStyled = styleWarn.Render(waitstr)
		} else {
			waitStyled = styleDim.Render(waitstr)
		}

		fixed := totalWidth(processColW)
		qw := vw - fixed - 2

		row := spidStyled + " " +
			statusStyled + " " +
			styleCell.Render(padRight(truncate(s.LoginName, processColW[2]), processColW[2])) + " " +
			styleDim.Render(padRight(truncate(s.DatabaseName, processColW[3]), processColW[3])) + " " +
			styleDim.Render(padRight(truncate(s.ProgramName, processColW[4]), processColW[4])) + " " +
			waitStyled + " " +
			styleCell.Render(padRight(fmt.Sprintf("%d", s.WaitTimeMs), processColW[6])) + " "

		if qw > 0 {
			row += styleDim.Render(truncate(cleanQuery(s.QueryText), qw))
		}

		out += row + "\n"
	}
	return out
}

var waitsColW = []int{38, 14, 14}

func buildWaitsTable(waits []db.WaitStat, vw int) string {
	headers := []string{"Wait Type", "Tasks", "Total ms"}
	out := tableHeader(headers, waitsColW, vw) + tableRule(vw)

	if len(waits) == 0 {
		return out + styleDim.Render(" (no wait statistics)\n")
	}

	//FInd the largest waittime
	maxMs := waits[0].WaittimeMs // We have an order by clause in the query
	if maxMs == 0 {
		maxMs = 1
	}

	const barWidth = 14

	for _, w := range waits {
		// proprotional ba: filled = barwidth *thisWait / maxWait
		filled := int(int64(barWidth) * w.WaittimeMs / maxMs)
		bar := ""
		for i := 0; i < barWidth; i++ {
			if i < filled {
				bar += styleWarn.Render("▪")
			} else {
				bar += styleDim.Render("·")
			}
		}
		avgStr := fmt.Sprintf(" avg %.1f ms", w.AvgWaitMs)

		out += styleCell.Render(padRight(truncate(w.WaitType, waitsColW[0]), waitsColW[0])) + " " +
			styleDim.Render(padRight(fmt.Sprintf("%d", w.WaitingTasks), waitsColW[1])) + " " +
			styleDim.Render(padRight(fmt.Sprintf("%d", w.WaittimeMs), waitsColW[2])) + " " +
			bar +
			styleDim.Render(avgStr) + "\n"
	}
	return out
}

var expColW = []int{12, 14, 16, 12, 14}

func buildExpensiveTable(queries []db.ExpensiveQuery, vw int) string {
	headers := []string{"executions", "Total CPU ms", "Total Elapsed ms", "Avg CPU ms", "Last run"}
	out := tableHeader(headers, expColW, vw) + tableRule(vw)

	if len(queries) == 0 {
		return out + styleDim.Render(" (no cached query plans found)\n")
	}

	for _, q := range queries {
		lastRun := q.LastExecution.Format("01-02 15:04:05")

		fixed := totalWidth(expColW)
		qw := vw - fixed - 2

		row := styleCell.Render(padRight(fmt.Sprintf("%d", q.ExecutionCount), expColW[0])) + " " +
			styleWarn.Render(padRight(fmt.Sprintf("%d", q.TotalCPUMs), expColW[1])) + " " +
			styleDim.Render(padRight(fmt.Sprintf("%d", q.TotalElapsedMs), expColW[2])) + " " +
			styleCell.Render(padRight(fmt.Sprintf("%.1f", q.AvgCPUMs), expColW[3])) + " " +
			styleDim.Render(padRight(lastRun, expColW[4])) + " "
		if qw > 0 {
			row += styleDim.Render(truncate(cleanQuery(q.QueryText), qw))
		}
		out += row + "\n"
	}
	return out
}

// table helpers
func tableHeader(headers []string, colW []int, vw int) string {
	row := ""
	fixed := totalWidth(colW)
	lastW := vw - fixed - 2
	for i, h := range headers {
		w := colW[i]
		if i == len(headers)-1 {
			w = lastW
		}
		if w <= 0 {
			break
		}
		row += styleHeaderCell.Render(padRight(h, w)) + " "
	}
	return row + "\n"
}

func tableRule(vw int) string {
	return styleSeparator.Render(repeatString("-", vw-2)) + "\n"
}

func totalWidth(cols []int) int {
	totalWidth := 0
	for _, c := range cols {
		totalWidth += c + 1
	}
	return totalWidth
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
