// Package repl implements the inline interface: a logo banner, a bordered
// prompt at the bottom, and command output that scrolls into the terminal's
// real scrollback. It reuses the store, the token parser (task.ParseTokens)
// and the shared theme layer (internal/ui).
package repl

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
)

const promptGlyph = "› "

type mode int

const (
	modePrompt mode = iota
	modeList
	modeDetail
	modeNote
	modeAddChild
)

type model struct {
	store *store.Store
	th    ui.Theme
	lang  i18n.Lang
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
	listSort   task.SortMode  // overlay sort (report override or model default)
	listLimit  int            // 0 = no row cap
	expanded   map[int64]bool // subtask fold state; absent = expanded
	cursor     int
	offset     int

	// add-child prompt (create a subtask under the cursor task)
	addParent int64
	addTitle  string
	addInput  textinput.Model

	// detail overlay
	detail      *task.Task
	detailLines []string
	detailVP    viewport.Model

	// note prompt
	noteTarget int64
	noteTitle  string
	noteInput  textinput.Model
	noteReturn mode // where to return when the note prompt closes

	pendingEcho string

	transcript []string // test seam: every emitted scrollback block
}

func newModel(s *store.Store, opts Options) model {
	in := textinput.New()
	in.Prompt = promptGlyph
	in.Placeholder = opts.Lang.T("prompt.placeholder")
	in.CharLimit = 500
	in.Cursor.SetMode(cursor.CursorStatic)
	in.Focus()

	return model{
		store:    s,
		th:       ui.NewTheme(opts.ThemeName, opts.ASCII),
		lang:     opts.Lang,
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
		// keep the input inside the prompt box, which is capped at width 100;
		// otherwise a wider terminal overflows the box and misaligns its border.
		m.input.Width = max(10, min(m.w, 100)-8)
		return m, nil

	case cachesMsg:
		m.projects, m.tags = msg.projects, msg.tags
		return m, nil

	case errMsg:
		return m, m.emit(m.th.StatusErr.Render("x " + msg.err.Error()))

	case resultMsg:
		cmd := m.emit(msg.lines...)
		if msg.reload {
			return m, tea.Batch(cmd, m.loadCachesCmd())
		}
		return m, cmd

	case openListMsg:
		m.listTitle = msg.title
		m.listTasks = msg.tasks
		m.listFilter = msg.filter
		m.listSort = msg.sort
		m.listLimit = msg.limit
		m.cursor, m.offset = 0, 0
		m.rebuildList()
		m.cursor = 0
		m.mode = modeList
		return m, m.echoOnly()

	case listRefreshMsg:
		m.listTasks = msg.tasks
		m.rebuildList()
		return m, m.loadCachesCmd()

	case openNoteMsg:
		// note <id> from the prompt: capture returns to the prompt.
		return m.beginNote(msg.id, msg.title, modePrompt), m.echoOnly()

	case detailLoadedMsg:
		m.detail = msg.t
		w := min(m.w, 100)
		m.detailLines = detailBlock(m.th, m.lang, msg.t, msg.parent, msg.children, msg.notes, msg.acts, w-4, m.ascii)
		h := min(len(m.detailLines), maxOverlayRows)
		if h < 1 {
			h = 1
		}
		m.detailVP = viewport.New(w-2, h)
		m.detailVP.SetContent(strings.Join(m.detailLines, "\n"))
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
		case modeAddChild:
			return m.updateAddChild(msg)
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
		ret := m.noteReturn
		m.mode = ret
		if ret == modePrompt {
			return m, m.emit(m.th.Dim.Render(m.lang.T("note.cancelled")))
		}
		return m, nil // returning to an overlay: no scrollback noise
	case "enter":
		body := strings.TrimSpace(m.noteInput.Value())
		ret := m.noteReturn
		id := m.noteTarget
		m.mode = ret
		if body == "" {
			if ret == modePrompt {
				return m, m.emit(m.th.Dim.Render(m.lang.T("note.empty")))
			}
			return m, nil
		}
		if ret == modeDetail {
			// add the note, then reload the detail so it shows immediately.
			s, th := m.store, m.th
			load := m.openDetailCmd(id)
			return m, func() tea.Msg {
				if _, err := s.AddNote(id, body); err != nil {
					return errResult(th, err.Error())
				}
				return load()
			}
		}
		return m, m.storeCmd(func() resultMsg {
			if _, err := m.store.AddNote(id, body); err != nil {
				return resultMsg{lines: []string{m.th.StatusErr.Render("x " + err.Error())}}
			}
			return resultMsg{lines: []string{m.th.Accent.Render(m.lang.Tf("status.noteAdded", id))}}
		})
	}
	var cmd tea.Cmd
	m.noteInput, cmd = m.noteInput.Update(msg)
	return m, cmd
}

// beginNote opens the note-capture box for task id, remembering where to return
// (modePrompt for the `note` command, modeList/modeDetail for the `n` shortcut).
func (m model) beginNote(id int64, title string, ret mode) model {
	m.noteTarget = id
	m.noteTitle = title
	ni := textinput.New()
	ni.Prompt = m.lang.T("note.promptGlyph")
	ni.CharLimit = 500
	ni.Cursor.SetMode(cursor.CursorStatic)
	ni.Width = max(10, min(m.w, 60)-10)
	ni.Focus()
	m.noteInput = ni
	m.noteReturn = ret
	m.mode = modeNote
	return m
}

// startAddNote opens the note box for t from an overlay (list or detail),
// returning there when done. Mirrors startAddChild.
func (m model) startAddNote(t *task.Task, ret mode) model {
	return m.beginNote(t.ID, t.Title, ret)
}

// startAddChild opens the inline prompt to create a subtask under t, so the
// user never has to type the parent id.
func (m model) startAddChild(t *task.Task) model {
	m.addParent = t.ID
	m.addTitle = t.Title
	ci := textinput.New()
	ci.Prompt = m.lang.T("child.promptGlyph")
	ci.CharLimit = 500
	ci.Cursor.SetMode(cursor.CursorStatic)
	ci.Width = max(10, min(m.w, 60)-10)
	ci.Focus()
	m.addInput = ci
	m.expanded[t.ID] = true // reveal the new child
	m.mode = modeAddChild
	return m
}

func (m model) updateAddChild(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeList
		return m, nil
	case "enter":
		title := strings.TrimSpace(m.addInput.Value())
		parent := m.addParent
		m.mode = modeList
		if title == "" {
			return m, nil
		}
		filter := m.listFilter
		s, th := m.store, m.th
		return m, func() tea.Msg {
			t := task.Task{Title: title, ParentID: parent}
			if err := s.AddTask(&t); err != nil {
				return errResult(th, err.Error())
			}
			tasks, err := s.List(filter)
			if err != nil {
				return errResult(th, err.Error())
			}
			return listRefreshMsg{tasks: tasks}
		}
	}
	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
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
		box := ui.DrawBoxChars(m.th, roundBox(m.ascii), m.lang.T("note.boxTitle")+ui.TruncRunes(m.noteTitle, 30),
			[]string{" " + m.noteInput.View(), m.th.Dim.Render(m.lang.T("note.boxHint"))},
			min(m.w, 60), 4, true)
		return box
	case modeAddChild:
		box := ui.DrawBoxChars(m.th, roundBox(m.ascii), m.lang.T("child.boxTitle")+ui.TruncRunes(m.addTitle, 30),
			[]string{" " + m.addInput.View(), m.th.Dim.Render(m.lang.T("child.boxHint"))},
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
// cursor on the same task when it is still visible. Honors the overlay sort
// override and row limit set by reports.
func (m *model) rebuildList() {
	var id int64
	if t := m.cursorTask(); t != nil {
		id = t.ID
	}
	sortMode := m.sort
	if m.listSort != "" {
		sortMode = m.listSort
	}
	rows := flattenTree(m.listTasks, sortMode, m.expanded)
	if m.listLimit > 0 && len(rows) > m.listLimit {
		rows = rows[:m.listLimit]
	}
	m.listRows = rows
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
