// Package tui implements the Norton Commander-style terminal interface.
package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
)

// Modal is a dialog that captures all input while open.
type Modal interface {
	Update(msg tea.Msg) (Modal, tea.Cmd)
	View(th Theme, w, h int) string
}

type focusArea int

const (
	focusSidebar focusArea = iota
	focusList
)

const sidebarWidth = 26

// Options configures the TUI at startup (resolved in main.go).
type Options struct {
	ThemeName string
	ASCII     bool
	SortMode  task.SortMode
	Lang      i18n.Lang
}

type App struct {
	store *store.Store
	th    Theme
	lang  i18n.Lang
	ascii bool

	sortMode task.SortMode

	w, h  int
	focus focusArea

	sidebar Sidebar
	list    TaskList
	modal   Modal

	filter    task.Filter
	activeTab int    // index into tabDefs
	activeCtx string // name of the active context ("" = none)
	search    textinput.Model
	searching bool

	overdueCount int // tints the Overdue tab label

	status    string
	statusErr bool

	pendingDelete int64
	reduceMotion  bool // skip scroll animation (set in tests)
}

func Run(s *store.Store, opts Options) error {
	app := newApp(s, opts)
	// cell-motion mouse: click selects, wheel scrolls. It disables the
	// terminal's native text selection while the TUI runs (Shift+drag still
	// selects in Windows Terminal) — standard trade-off for full-screen TUIs.
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func newApp(s *store.Store, opts Options) *App {
	// normalize once so i18n.Next and the header language tag always see a
	// valid code (tests construct Options{} with a zero Lang)
	opts.Lang = i18n.Normalize(string(opts.Lang))
	search := textinput.New()
	search.Prompt = opts.Lang.T("app.searchPrompt")
	search.CharLimit = 100
	search.Width = 40
	search.Cursor.SetMode(cursor.CursorStatic)
	return &App{
		store:    s,
		th:       NewTheme(opts.ThemeName, opts.ASCII),
		lang:     opts.Lang,
		ascii:    opts.ASCII,
		sortMode: task.NormalizeSortMode(string(opts.SortMode)),
		focus:    focusList,
		list:     NewTaskList(),
		search:   search,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(a.loadTasksCmd(), a.loadProjectsCmd())
}

// --- commands (all store access happens here) ---

func (a *App) loadTasksCmd() tea.Cmd {
	base := a.filter
	return func() tea.Msg {
		// fold the active context in, REPL-style: the sidebar's own scalars
		// win over the context's, tags union (see model.applyContext)
		cf, _, err := a.store.ContextFilter(time.Now())
		if err != nil {
			return errMsg{err}
		}
		f := cf.Merge(base)
		if f.Status == "" && !f.IncludeAll && !f.WaitingOnly {
			f.HideWaiting = true
		}
		tasks, err := a.store.List(f)
		if err != nil {
			return errMsg{err}
		}
		return tasksLoadedMsg{tasks}
	}
}

func (a *App) loadProjectsCmd() tea.Cmd {
	return func() tea.Msg {
		counts, err := a.store.ProjectStatusCounts()
		if err != nil {
			return errMsg{err}
		}
		total := 0
		for _, c := range counts {
			total += c.Pending
		}
		tags, err := a.store.AllTags()
		if err != nil {
			return errMsg{err}
		}
		now := time.Now()
		cf, activeCtx, err := a.store.ContextFilter(now)
		if err != nil {
			return errMsg{err}
		}
		ctxDefs, err := a.store.Contexts()
		if err != nil {
			return errMsg{err}
		}
		// counts mirror what each row shows when selected (pending + waiting
		// hidden), so numbers always match the list
		count := func(f task.Filter) int {
			if f.Status == "" && !f.WaitingOnly { // same rule as loadTasksCmd
				f.HideWaiting = true
			}
			ts, err := a.store.List(f)
			if err != nil {
				return 0
			}
			return len(ts)
		}
		// context-aware counts fold the active context in, like the list does;
		// project and tag counts stay global (they answer "how many exist",
		// not "how many under this context")
		countCtx := func(f task.Filter) int { return count(cf.Merge(f)) }
		var ctxs []ctxEntry
		names := make([]string, 0, len(ctxDefs))
		for n := range ctxDefs {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			c := 0
			if _, f, text, perr := task.ParseTokens(strings.Fields(ctxDefs[n]), now); perr == nil {
				f.Text = text
				c = count(f)
			}
			ctxs = append(ctxs, ctxEntry{name: n, count: c})
		}
		return projectsLoadedMsg{data: sidebarData{
			counts:    counts,
			tags:      tags,
			total:     total,
			overdue:   countCtx(task.Filter{DueBefore: &now}),
			done:      countCtx(task.Filter{Status: task.StatusDone}),
			del:       countCtx(task.Filter{Status: task.StatusDeleted}),
			contexts:  ctxs,
			activeCtx: activeCtx,
		}}
	}
}

func (a *App) openDetailCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		t, err := a.store.GetTask(id)
		if err != nil {
			return errMsg{err}
		}
		children, err := a.store.Children(id)
		if err != nil {
			return errMsg{err}
		}
		notes, err := a.store.Notes(id)
		if err != nil {
			return errMsg{err}
		}
		acts, err := a.store.TaskActivity(id)
		if err != nil {
			return errMsg{err}
		}
		return detailLoadedMsg{t: t, children: children, notes: notes, acts: acts}
	}
}

func (a *App) openReadCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		t, err := a.store.GetTask(id)
		if err != nil {
			return errMsg{err}
		}
		notes, err := a.store.Notes(id)
		if err != nil {
			return errMsg{err}
		}
		return readLoadedMsg{t: t, notes: notes}
	}
}

func (a *App) reload() tea.Cmd {
	return tea.Batch(a.loadTasksCmd(), a.loadProjectsCmd())
}

// --- update ---

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
		return a, nil

	case tea.MouseMsg:
		return a.handleMouse(msg)

	case errMsg:
		a.status, a.statusErr = msg.err.Error(), true
		return a, nil

	case statusMsg:
		// every mutation ends in a statusMsg — refresh both panels
		a.status, a.statusErr = string(msg), false
		return a, a.reload()

	case tasksLoadedMsg:
		a.list.SetTasks(msg.tasks)
		return a, nil

	case projectsLoadedMsg:
		a.sidebar.SetCounts(a.lang, msg.data)
		a.activeCtx = msg.data.activeCtx
		a.overdueCount = msg.data.overdue
		return a, nil

	case detailLoadedMsg:
		a.modal = NewDetail(a.lang, a.ascii, a.reduceMotion, msg.t, msg.children, msg.notes, msg.acts)
		return a, nil

	case readLoadedMsg:
		a.modal = NewRead(a.lang, a.ascii, a.reduceMotion, msg.t, msg.notes)
		return a, nil

	case formSubmittedMsg:
		a.modal = nil
		t := msg.t
		return a, func() tea.Msg {
			var err error
			var what string
			if msg.edit {
				err = a.store.UpdateTask(&t)
				what = a.lang.Tf("app.taskUpdated", t.ID)
			} else {
				err = a.store.AddTask(&t)
				what = a.lang.Tf("app.taskCreated", t.ID)
			}
			if err != nil {
				return errMsg{err}
			}
			return statusMsg(what)
		}

	case noteSubmittedMsg:
		a.modal = nil
		return a, func() tea.Msg {
			if _, err := a.store.AddNote(msg.taskID, msg.body); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.noteAdded", msg.taskID))
		}

	case moveSubmittedMsg:
		a.modal = nil
		return a, func() tea.Msg {
			t, err := a.store.GetTask(msg.taskID) // fresh copy, never the list's pointer
			if err != nil {
				return errMsg{err}
			}
			if msg.parentID != 0 {
				if err := a.store.CheckMoveCycle(msg.taskID, msg.parentID); err != nil {
					return errMsg{err}
				}
			}
			t.Project = msg.project
			t.ParentID = msg.parentID
			if err := a.store.UpdateTask(t); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.taskMoved", msg.taskID))
		}

	case confirmResultMsg:
		a.modal = nil
		id := a.pendingDelete
		a.pendingDelete = 0
		if !msg.ok || id == 0 {
			return a, nil
		}
		return a, func() tea.Msg {
			if err := a.store.DeleteTask(id); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.taskDeleted", id))
		}

	case modalCancelMsg:
		a.modal = nil
		return a, nil
	}

	if a.modal != nil {
		var cmd tea.Cmd
		a.modal, cmd = a.modal.Update(msg)
		return a, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return a.handleKey(keyMsg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if a.searching {
		switch key {
		case "enter":
			a.searching = false
			a.filter.Text = strings.TrimSpace(a.search.Value())
			return a, a.loadTasksCmd()
		case "esc":
			a.searching = false
			a.search.SetValue("")
			a.filter.Text = ""
			return a, a.loadTasksCmd()
		}
		var cmd tea.Cmd
		a.search, cmd = a.search.Update(msg)
		return a, cmd
	}

	switch key {
	case "ctrl+c", "q", "f10":
		return a, tea.Quit

	case "f1", "?":
		a.modal = &Help{lang: a.lang}
		return a, nil

	case "tab":
		if a.focus == focusSidebar {
			a.focus = focusList
		} else {
			a.focus = focusSidebar
		}
		return a, nil

	case "up", "k":
		if a.focus == focusSidebar {
			a.sidebar.Move(-1)
			return a.applyFilters()
		}
		a.list.Move(-1)
		return a, nil

	case "down", "j":
		if a.focus == focusSidebar {
			a.sidebar.Move(1)
			return a.applyFilters()
		}
		a.list.Move(1)
		return a, nil

	case "[":
		return a.setTab(a.activeTab - 1)
	case "]":
		return a.setTab(a.activeTab + 1)
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if i := int(key[0] - '1'); i < len(tabDefs()) {
			return a.setTab(i)
		}
		return a, nil

	case "pgup":
		a.list.Move(-10)
		return a, nil
	case "pgdown":
		a.list.Move(10)
		return a, nil
	case "home", "g":
		a.list.Home()
		return a, nil
	case "end", "G":
		a.list.End()
		return a, nil

	case "left", "h":
		if a.focus == focusList {
			a.list.Collapse()
		}
		return a, nil
	case "right", "l":
		if a.focus == focusList {
			a.list.Expand()
		}
		return a, nil

	case "enter":
		if a.focus == focusSidebar {
			if name, ok := a.sidebar.CurrentContext(); ok {
				return a, a.toggleContextCmd(name)
			}
			a.focus = focusList
			return a, nil
		}
		if t := a.list.CursorTask(); t != nil {
			return a, a.openDetailCmd(t.ID)
		}
		return a, nil

	case "f3", "v":
		if t := a.list.CursorTask(); t != nil {
			return a, a.openDetailCmd(t.ID)
		}
		return a, nil

	case "R":
		if t := a.list.CursorTask(); t != nil {
			return a, a.openReadCmd(t.ID)
		}
		return a, nil

	case "f2", "a":
		a.modal = NewForm(a.lang, nil, 0, a.sidebar.CurrentProject())
		return a, nil

	case "s":
		if t := a.list.CursorTask(); t != nil {
			a.modal = NewForm(a.lang, nil, t.ID, t.Project)
		}
		return a, nil

	case "f6", "m":
		if t := a.list.CursorTask(); t != nil {
			a.modal = NewMove(a.lang, t.ID, t.Title, t.Project, t.ParentID)
		}
		return a, nil

	case "f4", "e":
		if t := a.list.CursorTask(); t != nil {
			a.modal = NewForm(a.lang, t, 0, "")
		}
		return a, nil

	case "f5", "n":
		if t := a.list.CursorTask(); t != nil {
			a.modal = NewNotePrompt(a.lang, t.ID, t.Title)
		}
		return a, nil

	case "f9", "d", " ":
		return a.toggleDone()

	case "S":
		return a.toggleStart()

	case "f8", "x", "delete":
		if t := a.list.CursorTask(); t != nil {
			a.pendingDelete = t.ID
			a.modal = NewConfirm(a.lang, a.lang.T("confirm.deleteTitle"), a.lang.Tf("confirm.deleteMsg", t.ID, truncRunes(t.Title, 40)))
		}
		return a, nil

	case "f7", "/":
		a.searching = true
		a.search.SetValue(a.filter.Text)
		a.search.Focus()
		return a, nil

	case "u":
		return a, func() tea.Msg {
			desc, err := a.store.Undo()
			if err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.undone", desc))
		}

	case "U":
		return a, func() tea.Msg {
			desc, err := a.store.Redo()
			if err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.redone", desc))
		}

	case "t":
		next := NextTheme(a.th.Name)
		a.th = NewTheme(next, a.ascii)
		return a, func() tea.Msg {
			if err := a.store.SetSetting("theme", next); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.theme", next))
		}

	case "L":
		// live toggle, mirroring the REPL's /lang: most strings re-localize on
		// the next render; the search prompt is cached and must be reassigned,
		// and the statusMsg's reload() rebuilds the cached sidebar labels
		a.lang = i18n.Next(a.lang)
		a.search.Prompt = a.lang.T("app.searchPrompt")
		next := a.lang
		return a, func() tea.Msg {
			if err := a.store.SetLanguage(string(next)); err != nil {
				return errMsg{err}
			}
			return statusMsg(next.Tf("app.lang", string(next)))
		}

	case "o":
		a.sortMode = a.sortMode.Next()
		a.list.SetSortMode(a.sortMode)
		mode := a.sortMode
		// the statusMsg triggers reload(), which re-sorts the list
		return a, func() tea.Msg {
			if err := a.store.SetSetting("sort", string(mode)); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.sort", a.lang.T("sort."+string(mode))))
		}

	case "r":
		return a, a.reload()
	}
	return a, nil
}

// applyFilters recomputes the list filter: active tab report ⊕ sidebar
// selection. Sidebar scalars win, tags union — the same Merge semantics the
// active context gets in loadTasksCmd (context < tab < sidebar).
func (a *App) applyFilters() (tea.Model, tea.Cmd) {
	f := a.tabFilter(time.Now()).Merge(a.sidebar.Filter())
	f.Text = a.filter.Text
	a.filter = f
	a.list.SetLimit(a.tabLimit())
	return a, a.loadTasksCmd()
}

func (a *App) toggleDone() (tea.Model, tea.Cmd) {
	t := a.list.CursorTask()
	if t == nil {
		return a, nil
	}
	id := t.ID
	status := t.Status
	return a, func() tea.Msg {
		switch status {
		case task.StatusPending:
			next, err := a.store.CompleteTask(id)
			if err != nil {
				return errMsg{err}
			}
			if next != nil {
				return statusMsg(a.lang.Tf("app.taskDoneRecur", id, next.ID, next.Due.Format("02/01")))
			}
			return statusMsg(a.lang.Tf("app.taskDone", id))
		case task.StatusDone:
			if err := a.store.ReopenTask(id); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.taskReopened", id))
		}
		return statusMsg(a.lang.T("app.taskDeletedRestore"))
	}
}

// toggleContextCmd activates the named context, or clears it when it is
// already the active one. Contexts live in settings — deliberately not
// undoable — and the statusMsg reload re-renders everything context-aware.
func (a *App) toggleContextCmd(name string) tea.Cmd {
	return func() tea.Msg {
		active, err := a.store.ActiveContext()
		if err != nil {
			return errMsg{err}
		}
		if active == name {
			if err := a.store.SetActiveContext(""); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.T("app.ctxCleared"))
		}
		if err := a.store.SetActiveContext(name); err != nil {
			return errMsg{err}
		}
		return statusMsg(a.lang.Tf("app.ctxActive", name))
	}
}

// toggleStart flips the active (started) state of the task under the cursor.
func (a *App) toggleStart() (tea.Model, tea.Cmd) {
	t := a.list.CursorTask()
	if t == nil || t.Status != task.StatusPending {
		return a, nil
	}
	id, active := t.ID, t.IsActive()
	return a, func() tea.Msg {
		if active {
			if err := a.store.StopTask(id); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.taskStopped", id))
		}
		if err := a.store.StartTask(id); err != nil {
			return errMsg{err}
		}
		return statusMsg(a.lang.Tf("app.taskStarted", id))
	}
}

// --- view ---

func (a *App) View() string {
	if a.w < 60 || a.h < 12 {
		return a.lang.T("app.windowSmall")
	}

	// the header and the tab band shrink the content area; the frame always
	// spans exactly a.h rows
	header := a.renderHeader()
	hdr := strings.Join(header, "\n") + "\n"
	tabs := strings.Join(a.renderTabBand(), "\n") + "\n"
	contentH := a.contentHeight()

	main := a.renderMain(contentH)
	if a.modal != nil {
		content := a.modal.View(a.th, a.w, contentH+1)
		main = overlayModal(a.th, main, content, a.w)
	}

	var b strings.Builder
	b.WriteString(hdr)
	b.WriteString(tabs)
	for _, ln := range main {
		b.WriteString(ln + "\n")
	}
	b.WriteString(a.frameTrailer())
	return b.String()
}

// renderMain builds the live midsection — title row plus the two columns —
// as exactly contentH+1 full-width lines. It is both the normal view and the
// backdrop a modal dims over.
func (a *App) renderMain(contentH int) []string {
	sbLines := a.sidebar.Lines(a.th, sidebarWidth-2, contentH, a.focus == focusSidebar)
	listLines := a.list.Lines(a.th, a.lang, a.w-27, contentH, a.focus == focusList)
	out := make([]string, 0, contentH+1)
	out = append(out, a.titleRow())
	for i := 0; i < contentH; i++ {
		sb, li := "", ""
		if i < len(sbLines) {
			sb = sbLines[i]
		}
		if i < len(listLines) {
			li = listLines[i]
		}
		out = append(out, a.joinColumns(sb, li))
	}
	return out
}

// frameTrailer is the last two frame rows: status chips + key hints.
func (a *App) frameTrailer() string {
	return a.renderStatusChips() + "\n" + renderHint(a.th, a.lang, a.ascii, a.hintState(), a.w)
}

// listTitle combines the active tab and the sidebar selection for the list
// column title: "Today", "Tasks: casa", or "Today · Tasks: casa".
func (a *App) listTitle() string {
	st := a.sidebar.Title(a.lang)
	if a.activeTab == 0 {
		return st
	}
	tl := a.lang.T(tabDefs()[a.activeTab].labelKey)
	if st == a.lang.T("sb.title.tasks") {
		return tl
	}
	return tl + " · " + st
}

// titleRow labels the two boxless columns; the focused column's label is the
// only focus indicator left, so it renders TitleFocus (bold).
func (a *App) titleRow() string {
	sbStyle, listStyle := a.th.Title, a.th.Title
	if a.focus == focusSidebar {
		sbStyle = a.th.TitleFocus
	} else {
		listStyle = a.th.TitleFocus
	}
	left := sbStyle.Render(" " + truncRunes(a.lang.T("panel.projects"), sidebarWidth-3))
	right := listStyle.Render(" " + truncRunes(a.listTitle(), a.w-28))
	return a.joinColumns(left, right)
}

// joinColumns glues one sidebar row and one list row into a full-width frame
// line: sidebar (24 cells) · pad · │ separator · pad · list (a.w-27 cells).
func (a *App) joinColumns(sb, list string) string {
	pad := a.th.Bg.Render(" ")
	sep := a.th.Border.Render(a.th.Box.V)
	return padRow(sb, sidebarWidth-2, a.th.Bg) + pad + sep + pad + padRow(list, a.w-27, a.th.Bg)
}

// overlayModal centers the modal over the live main area: every backdrop
// line is stripped to plain text and re-rendered dim, and the modal rows are
// spliced in by rune position. Deliberately NOT lipgloss.Place: the v1 Place
// returns oversized content unchanged (no clipping), and tall modals (help
// at small sizes, the note prompt) rely on being clipped here so the frame
// keeps its exact height. Splicing by rune index assumes 1-cell runes, the
// same convention truncRunes lives by (double-width CJK text would drift).
func overlayModal(th Theme, backdrop []string, modal string, w int) []string {
	lines := strings.Split(modal, "\n")
	ch := len(lines)
	cw := 0
	for _, l := range lines {
		if lw := visibleWidth(l); lw > cw {
			cw = lw
		}
	}
	top := (len(backdrop) - ch) / 2
	if top < 0 {
		top = 0
	}
	left := (w - cw) / 2
	if left < 0 {
		left = 0
	}

	out := make([]string, 0, len(backdrop))
	for row, bl := range backdrop {
		plain := []rune(ui.StripANSI(bl))
		for len(plain) < w {
			plain = append(plain, ' ')
		}
		i := row - top
		if i < 0 || i >= ch {
			out = append(out, th.Dim.Render(string(plain[:w])))
			continue
		}
		rest := w - left - visibleWidth(lines[i])
		if rest < 0 {
			rest = 0
		}
		out = append(out, th.Dim.Render(string(plain[:left]))+lines[i]+th.Dim.Render(string(plain[w-rest:w])))
	}
	return out
}
