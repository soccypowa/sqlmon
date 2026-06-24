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
	chromeHeight = 8 // Borde vara 9 enligt exemplet
	hScrollStep  = 8
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
	// process tab navigation
	selectedRow int
	hScroll     int
	showPopup   bool
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
			// Cleanup selected row in case the session count srank
			if m.selectedRow <= len(msg.sessions) && len(m.sessions) > 0 {
				m.selectedRow = len(msg.sessions) - 1
			}
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
		// When the popup is open we only hande close keys
		if m.showPopup {
			switch msg.String() {
			case "esc", "enter", "q", "ctrl+c":
				m.showPopup = false
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r": //manual refresh
			m.loading = true
			m.viewPort.SetContent(m.buildContent())
			cmds = append(cmds, fetchAllCmd(m.client))
		case "tab":
			m.activeTab = (m.activeTab + 1) % tabCount
			m.selectedRow = 0
			m.viewPort.GotoTop()
			m.viewPort.SetContent(m.buildContent())
		case "shift+tab":
			m.activeTab = (m.activeTab + tabCount - 1) % tabCount
			m.selectedRow = 0
			m.viewPort.GotoTop()
			m.viewPort.SetContent(m.buildContent())
		case "1":
			m.setTab(TabProcesses)
		case "2":
			m.setTab(TabWaits)
		case "3":
			m.setTab(TabExpensive)
		case "up":
			switch m.activeTab {
			case TabProcesses:
				if m.selectedRow > 0 {
					m.selectedRow--
					m.ensureVisible()
					m.viewPort.SetContent(m.buildContent())
				}
			case TabExpensive:
				if m.selectedRow > 0 {
					m.selectedRow--
					m.ensureVisible()
					m.viewPort.SetContent(m.buildContent())
				}
			}
		case "down":
			switch m.activeTab {
			case TabProcesses:
				if m.selectedRow < len(m.sessions)-1 {
					m.selectedRow++
					m.ensureVisible()
					m.viewPort.SetContent(m.buildContent())
				}
			case TabExpensive:
				if m.selectedRow < len(m.queries)-1 {
					m.selectedRow++
					m.ensureVisible()
					m.viewPort.SetContent(m.buildContent())
				}
			}
		case "enter":
			switch m.activeTab {
			case TabProcesses:
				if len(m.sessions) > 0 {
					m.showPopup = true
				}
			case TabExpensive:
				if len(m.queries) > 0 {
					m.showPopup = true
				}
			}
		case "right":
			if m.activeTab == TabProcesses {
				m.hScroll += hScrollStep
				m.viewPort.SetContent(m.buildContent())
			}
		case "left":
			if m.activeTab == TabProcesses {
				m.hScroll -= hScrollStep
				if m.hScroll < 0 {
					m.hScroll = 0
				}
				m.viewPort.SetContent(m.buildContent())
			}
		case "home":
			m.hScroll = 0
			m.viewPort.SetContent(m.buildContent())
		}
	}

	if !m.showPopup {
		// Forward all other messages to the viewport so it can handle
		// scroll keys (↑ ↓ PgUp PgDn g G) itself.
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

// ensureVisible adjusts the viewport's YOffset so that selectedRow
// (plus the 2-line header above the data rows) is within the visible
// window.
//
// The viewport content layout is:
//   line 0: column headers
//   line 1: separator rule
//   lines 2+: one data row per session
//
// So session[i] is at content line (i + 2).

func (m Model) ensureVisible() {
	const headerlines = 2
	targetLine := m.selectedRow + headerlines

	if targetLine < m.viewPort.YOffset {
		m.viewPort.YOffset = targetLine
	} else if targetLine >= m.viewPort.YOffset+m.viewPort.Height {
		m.viewPort.YOffset = targetLine - m.viewPort.Height + 1
	}
}

func (m *Model) setTab(t Tab) {
	m.activeTab = t
	m.selectedRow = 0
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
		return buildProcessTable(m.sessions, m.viewPort.Width, m.selectedRow, m.hScroll)
	case TabWaits:
		return buildWaitsTable(m.waits, m.viewPort.Width)
	case TabExpensive:
		return buildExpensiveTable(m.queries, m.viewPort.Width, m.selectedRow)
	}

	return ""
}

// Column layout: fixed widths for all but the last (Query) column.
// We compute query width dynamically from the available viewport width.
// HORIZONTAL SCROLL IMPLEMENTATION
//
// Each row is built as a full-width ANSI-escaped string. We can't
// simply index into it with s[hScroll:] because ANSI escape sequences
// are invisible bytes — slicing mid-escape corrupts the output.
//
// Instead we work with the *visible* content before styling:
//   1. Build the plain-text cell values (no ANSI).
//   2. Apply hScroll as a character offset to the plain cells.
//   3. Render the (now-offset) cells with Lipgloss.
//
// This means we lose the colour on partially-scrolled cells at the
// left edge, but the layout is always correct. The alternative
// (ANSI-aware string slicing) adds significant complexity for
// marginal benefit.

// Full column definitions for the Processes table.
// id, label, width

type col struct {
	label string
	width int
}

var processCols = []col{
	{"SPID", 6},
	{"Status", 11},
	{"Login", 30},
	{"Host", 14},
	{"Program", 20},
	{"Database", 14},
	{"Wait Type", 16},
	{"Wait ms", 9},
	{"CPU ms", 9},
	{"Elapsed ms", 11},
	{"Blk by", 7},
	{"Query", 50},
}

func buildProcessTable(sessions []db.Session, vw, selctedRow, hScroll int) string {
	totalW := 0
	for _, c := range processCols {
		totalW += c.width + 1
	}

	headerCells := make([]string, len(processCols))
	for i, c := range processCols {
		headerCells[i] = padRight(c.label, c.width)
	}
	headerLine := hCrop(joinCells(headerCells), hScroll, vw-2)
	out := styleHeaderCell.Render(headerLine) + "\n" + tableRule(vw)

	if len(sessions) == 0 {
		return out + styleDim.Render(" (no user sessions)\n")
	}
	for i, s := range sessions {
		cells := []string{
			padRight(spidLabel(s), processCols[0].width),
			padRight(s.Status, processCols[1].width),
			padRight(truncate(s.LoginName, processCols[2].width), processCols[2].width),
			padRight(truncate(s.HostName, processCols[3].width), processCols[3].width),
			padRight(truncate(s.ProgramName, processCols[4].width), processCols[4].width),
			padRight(truncate(s.DatabaseName, processCols[5].width), processCols[5].width),
			padRight(truncate(s.WaitType, processCols[6].width), processCols[6].width),
			padRight(fmt.Sprintf("%d", s.WaitTimeMs), processCols[7].width),
			padRight(fmt.Sprintf("%d", s.CPUTime), processCols[8].width),
			padRight(fmt.Sprintf("%d", s.ElapsedMs), processCols[9].width),
			padRight(blockedByLabel(s.BlockedBy), processCols[10].width),
			truncate(cleanQuery(s.QueryText), processCols[11].width),
		}

		// Apply horizontal scroll: crop the joined plain-text row to
		// [hScroll : hScroll+viewWidth] before adding any ANSI.
		plainRow := joinCells(cells)
		cropedRow := hCrop(plainRow, hScroll, vw-2)

		var styledRow string
		switch {
		case i == selctedRow && s.BlockedBy > 0:
			styledRow = styleBad.Background(colorSelected).Render(cropedRow)
		case i == selctedRow:
			styledRow = styleSelected.Foreground(colorWhite).Render(cropedRow)
		case s.BlockedBy > 0:
			styledRow = styleBad.Render(cropedRow)
		case s.Status == "running":
			styledRow = styleGood.Render(cropedRow)
		case s.Status == "suspended" && s.WaitType != "":
			styledRow = styleWarn.Render(cropedRow)
		default:
			styledRow = styleDim.Render(cropedRow)
		}

		cursor := "  "
		if i == selctedRow {
			cursor = styleTitle.Render("▶ ")
		}
		out += cursor + styledRow + "\n"
	}
	return out
}

func spidLabel(s db.Session) string {
	if s.BlockedBy > 0 {
		return fmt.Sprintf("%d!", s.SPID)
	}
	return fmt.Sprintf("%d", s.SPID)
}

func blockedByLabel(n int) string {
	if n > 0 {
		return fmt.Sprintf("%d", n)
	}
	return ""
}

func joinCells(cells []string) string {
	out := ""
	for i, c := range cells {
		if i > 0 {
			out += " "
		}
		out += c
	}
	return out
}

// hCrop returns the visible slice of a plain-text string starting at
// character offset `skip`, truncated to at most `width` characters.
// If skip ≥ len(s) it returns a blank string of the correct width.
func hCrop(s string, skip, width int) string {
	runes := []rune(s)
	if skip >= len(runes) {
		return repeatString(" ", width)
	}
	runes = runes[skip:]
	if len(runes) > width {
		runes = runes[:width]
	}
	// Pad to exactly width so the row fills the panel.
	for len(runes) < width {
		runes = append(runes, ' ')
	}
	return string(runes)
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

func buildExpensiveTable(queries []db.ExpensiveQuery, vw, selectedRow int) string {
	headers := []string{"executions", "Total CPU ms", "Total Elapsed ms", "Avg CPU ms", "Last run"}
	out := tableHeader(headers, expColW, vw) + tableRule(vw)

	if len(queries) == 0 {
		return out + styleDim.Render(" (no cached query plans found)\n")
	}

	for i, q := range queries {
		fixed := totalWidth(expColW)
		qw := vw - fixed - 2

		row := styleCell.Render(padRight(fmt.Sprintf("%d", q.ExecutionCount), expColW[0])) + " " +
			styleWarn.Render(padRight(fmt.Sprintf("%d", q.TotalCPUMs), expColW[1])) + " " +
			styleDim.Render(padRight(fmt.Sprintf("%d", q.TotalElapsedMs), expColW[2])) + " " +
			styleCell.Render(padRight(fmt.Sprintf("%.1f", q.AvgCPUMs), expColW[3])) + " " +
			styleDim.Render(padRight(q.LastExecution.Format("01-02 15:04:05"), expColW[4])) + " "
		if qw > 0 {
			row += styleDim.Render(truncate(cleanQuery(q.QueryText), qw))
		}
		cursor := "  "
		if i == selectedRow {
			cursor = styleTitle.Render("▶ ")
			row = styleSelected.Foreground(colorWhite).Render(row)
		}
		out += cursor + row + "\n"
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
