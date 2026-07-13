// Package repl implements the inline, Claude-Code-style interface: a logo
// banner, a bordered prompt at the bottom, and command output that scrolls
// into the terminal's real scrollback. It reuses the store, the token parser
// (task.ParseTokens) and the shared theme layer (internal/ui).
package repl

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
	"github.com/jvsaga/taskframe/internal/ui"
)

const promptGlyph = "› "

type mode int

const (
	modePrompt mode = iota
	modeList
	modeDetail
	modeNote
)

type model struct {
	store *store.Store
	th    ui.Theme
	ascii bool
	sort  task.SortMode

	w, h int
	mode mode

	input textinput.Model

	hist     history
	projects []string // completion cache: aggregated dotted project paths
	tags     []string
	compHint string // transient completion candidates, shown under the prompt

	// list overlay
	listTitle  string
	listTasks  []*task.Task // flat set backing the overlay (for tree rebuilds)
	listRows   []olRow
	listFilter task.Filter
	expanded   map[int64]bool // subtask fold state; absent = expanded
	cursor     int
	offset     int

	// detail overlay
	detail       *task.Task
	detailLines  []string
	detailScroll int

	// note prompt
	noteTarget int64
	noteTitle  string
	noteInput  textinput.Model

	pendingEcho string

	transcript []string // test seam: every emitted scrollback block
}

func newModel(s *store.Store, opts Options) model {
	in := textinput.New()
	in.Prompt = promptGlyph
	in.Placeholder = "add tarefa… · list · /help"
	in.CharLimit = 500
	in.Cursor.SetMode(cursor.CursorStatic)
	in.Focus()

	return model{
		store:    s,
		th:       ui.NewTheme(opts.ThemeName, opts.ASCII),
		ascii:    opts.ASCII,
		sort:     task.NormalizeSortMode(string(opts.SortMode)),
		w:        80,
		h:        24,
		mode:     modePrompt,
		input:    in,
		expanded: map[int64]bool{},
	}
}

func (m model) Init() tea.Cmd {
	return m.loadCachesCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.input.Width = max(10, m.w-8)
		return m, nil

	case cachesMsg:
		m.projects, m.tags = msg.projects, msg.tags
		return m, nil

	case errMsg:
		return m, m.emit(m.th.StatusErr.Render("✗ " + msg.err.Error()))

	case resultMsg:
		cmd := m.emit(msg.lines...)
		if msg.reload {
			return m, tea.Batch(cmd, m.loadCachesCmd())
		}
		return m, cmd

	case openListMsg:
		m.listTitle = msg.title
		m.listTasks = msg.tasks
		m.listRows = flattenTree(msg.tasks, m.sort, m.expanded)
		m.listFilter = msg.filter
		m.cursor, m.offset = 0, 0
		m.mode = modeList
		return m, m.echoOnly()

	case listRefreshMsg:
		m.listTasks = msg.tasks
		m.listRows = flattenTree(msg.tasks, m.sort, m.expanded)
		if m.cursor >= len(m.listRows) {
			m.cursor = len(m.listRows) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, m.loadCachesCmd()

	case openNoteMsg:
		m.noteTarget = msg.id
		m.noteTitle = msg.title
		ni := textinput.New()
		ni.Prompt = "nota› "
		ni.CharLimit = 500
		ni.Cursor.SetMode(cursor.CursorStatic)
		ni.Width = max(10, m.w-10)
		ni.Focus()
		m.noteInput = ni
		m.mode = modeNote
		return m, m.echoOnly()

	case detailLoadedMsg:
		m.detail = msg.t
		m.detailLines = detailBlock(m.th, msg.t, msg.parent, msg.children, msg.notes, msg.acts, m.w-6)
		m.detailScroll = 0
		m.mode = modeDetail
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeDetail:
			return m.updateDetail(msg)
		case modeNote:
			return m.updateNote(msg)
		default:
			return m.updatePrompt(msg)
		}
	}
	return m, nil
}

func (m model) updatePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "ctrl+d":
		return m, tea.Quit
	case "enter":
		line := strings.TrimSpace(m.input.Value())
		if line == "" {
			return m, nil
		}
		m.hist.push(line)
		m.input.SetValue("")
		m.compHint = ""
		m.pendingEcho = promptGlyph + line
		return m.dispatch(line)
	case "up":
		if v, ok := m.hist.prev(m.input.Value()); ok {
			m.input.SetValue(v)
			m.input.CursorEnd()
		}
		return m, nil
	case "down":
		if v, ok := m.hist.next(); ok {
			m.input.SetValue(v)
			m.input.CursorEnd()
		}
		return m, nil
	case "tab":
		m.complete()
		return m, nil
	}
	m.compHint = ""
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateNote(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modePrompt
		return m, m.emit(m.th.Dim.Render("  (nota cancelada)"))
	case "enter":
		body := strings.TrimSpace(m.noteInput.Value())
		m.mode = modePrompt
		if body == "" {
			return m, m.emit(m.th.Dim.Render("  (nota vazia, ignorada)"))
		}
		id := m.noteTarget
		return m, m.storeCmd(func() resultMsg {
			if _, err := m.store.AddNote(id, body); err != nil {
				return resultMsg{lines: []string{m.th.StatusErr.Render("✗ " + err.Error())}}
			}
			return resultMsg{lines: []string{m.th.Accent.Render("  ✓ nota adicionada à tarefa " + itoa(id))}}
		})
	}
	var cmd tea.Cmd
	m.noteInput, cmd = m.noteInput.Update(msg)
	return m, cmd
}

// View renders only the live region pinned at the bottom (never scrollback).
func (m model) View() string {
	switch m.mode {
	case modeList:
		return m.viewList()
	case modeDetail:
		return m.viewDetail()
	case modeNote:
		box := ui.DrawBoxChars(m.th, roundBox(m.ascii), "nota · "+ui.TruncRunes(m.noteTitle, 30),
			[]string{" " + m.noteInput.View(), m.th.Dim.Render(" enter salva · esc cancela")},
			min(m.w, 60), 4, true)
		return box
	default:
		return m.viewPrompt()
	}
}

func (m model) viewPrompt() string {
	w := min(m.w, 100)
	box := ui.DrawBoxChars(m.th, roundBox(m.ascii), "",
		[]string{" " + m.input.View()}, w, 3, true)
	if m.compHint != "" {
		box += "\n" + m.th.Dim.Render("  "+ui.TruncRunes(m.compHint, w-2))
	}
	return box
}

func roundBox(ascii bool) ui.BoxChars {
	if ascii {
		return ui.RoundAsciiBox
	}
	return ui.RoundBox
}

// --- helpers ---

func (m *model) emit(lines ...string) tea.Cmd {
	block := lines
	if m.pendingEcho != "" {
		block = append([]string{m.th.Dim.Render(m.pendingEcho)}, lines...)
		m.pendingEcho = ""
	}
	joined := strings.Join(block, "\n")
	m.transcript = append(m.transcript, joined)
	return tea.Println(joined)
}

// rebuildList re-flattens the overlay tree after a fold change, keeping the
// cursor on the same task when it is still visible.
func (m *model) rebuildList() {
	var id int64
	if t := m.cursorTask(); t != nil {
		id = t.ID
	}
	m.listRows = flattenTree(m.listTasks, m.sort, m.expanded)
	m.cursor = 0
	for i, r := range m.listRows {
		if r.t.ID == id {
			m.cursor = i
			break
		}
	}
}

// echoOnly prints just the typed command (used when the result is an overlay).
func (m *model) echoOnly() tea.Cmd {
	if m.pendingEcho == "" {
		return nil
	}
	line := m.th.Dim.Render(m.pendingEcho)
	m.pendingEcho = ""
	m.transcript = append(m.transcript, line)
	return tea.Println(line)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
