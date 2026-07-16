// Package tui implements the Norton Commander-style terminal interface.
package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
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
	activeCtx string // name of the active context ("" = none)
	search    textinput.Model
	searching bool

	status    string
	statusErr bool

	pendingDelete int64
	reduceMotion  bool // skip scroll animation (set in tests)
}

func Run(s *store.Store, opts Options) error {
	app := newApp(s, opts)
	p := tea.NewProgram(app, tea.WithAltScreen())
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
	f := a.filter
	if f.Status == "" && !f.IncludeAll && !f.WaitingOnly {
		f.HideWaiting = true
	}
	return func() tea.Msg {
		tasks, err := a.store.List(f)
		if err != nil {
			return errMsg{err}
		}
		return tasksLoadedMsg{tasks}
	}
}

func (a *App) loadProjectsCmd() tea.Cmd {
	return func() tea.Msg {
		counts, err := a.store.ProjectCounts()
		if err != nil {
			return errMsg{err}
		}
		total := 0
		for _, n := range counts {
			total += n
		}
		tags, err := a.store.AllTags()
		if err != nil {
			return errMsg{err}
		}
		// counts mirror what each virtual filter shows when selected
		// (pending + waiting hidden), so numbers always match the list
		now := time.Now()
		eodToday := task.EndOfDay(now)
		eodWeek := task.EndOfDay(now.AddDate(0, 0, 7))
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
		return projectsLoadedMsg{data: sidebarData{
			counts:  counts,
			tags:    tags,
			total:   total,
			today:   count(task.Filter{DueBefore: &eodToday}),
			overdue: count(task.Filter{DueBefore: &now}),
			week:    count(task.Filter{DueBefore: &eodWeek}),
			waiting: count(task.Filter{WaitingOnly: true}),
			done:    count(task.Filter{Status: task.StatusDone}),
			del:     count(task.Filter{Status: task.StatusDeleted}),
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
			return a.applySidebar()
		}
		a.list.Move(-1)
		return a, nil

	case "down", "j":
		if a.focus == focusSidebar {
			a.sidebar.Move(1)
			return a.applySidebar()
		}
		a.list.Move(1)
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

	case "t":
		next := NextTheme(a.th.Name)
		a.th = NewTheme(next, a.ascii)
		return a, func() tea.Msg {
			if err := a.store.SetSetting("theme", next); err != nil {
				return errMsg{err}
			}
			return statusMsg(a.lang.Tf("app.theme", next))
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

func (a *App) applySidebar() (tea.Model, tea.Cmd) {
	f := a.sidebar.Filter()
	f.Text = a.filter.Text
	a.filter = f
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

// --- view ---

func (a *App) View() string {
	if a.w < 60 || a.h < 12 {
		return a.lang.T("app.windowSmall")
	}

	// the header shrinks the panel area; the frame always spans exactly a.h rows
	header := a.renderHeader()
	hdr := strings.Join(header, "\n") + "\n"
	panelH := a.h - 2 - len(header)
	listW := a.w - sidebarWidth

	if a.modal != nil {
		content := a.modal.View(a.th, a.w, panelH)
		bg := lipglossPlace(a.th, content, a.w, panelH)
		return hdr + bg + "\n" + a.statusLine() + "\n" + renderFKeyBar(a.th, mainKeys(a.lang), a.w)
	}

	sbLines := a.sidebar.Lines(a.th, sidebarWidth-2, panelH-2, a.focus == focusSidebar)
	listLines := a.list.Lines(a.th, a.lang, listW-2, panelH-2, a.focus == focusList)

	left := drawBox(a.th, a.lang.T("panel.projects"), sbLines, sidebarWidth, panelH, a.focus == focusSidebar)
	right := drawBox(a.th, a.sidebar.Title(a.lang), listLines, listW, panelH, a.focus == focusList)

	panels := joinHorizontal(left, right)
	return hdr + panels + "\n" + a.statusLine() + "\n" + renderFKeyBar(a.th, mainKeys(a.lang), a.w)
}

func (a *App) statusLine() string {
	if a.searching {
		return padRow(a.th.Status.Render(" "+a.search.View()), a.w, a.th.Status)
	}
	style := a.th.Status
	if a.statusErr {
		style = a.th.StatusErr
	}
	info := a.lang.Tf("app.taskCount", a.list.Count())
	if a.filter.Text != "" {
		info += a.lang.Tf("app.searchInfo", a.filter.Text)
	}
	left := info
	if a.status != "" {
		left = " " + a.status + " ·" + info
	}
	// right-aligned sort indicator; the left side yields space when tight.
	rightSeg := a.lang.Tf("app.sort", a.lang.T("sort."+string(a.sortMode))) + " "
	avail := a.w - len([]rune(rightSeg))
	if avail < 0 {
		avail = 0
	}
	left = truncRunes(left, avail)
	gap := a.w - len([]rune(left)) - len([]rune(rightSeg))
	if gap < 0 {
		gap = 0
	}
	return style.Render(left) + style.Render(strings.Repeat(" ", gap)) + style.Render(rightSeg)
}

// joinHorizontal glues two multi-line blocks side by side.
func joinHorizontal(left, right string) string {
	ll := strings.Split(left, "\n")
	rl := strings.Split(right, "\n")
	n := len(ll)
	if len(rl) > n {
		n = len(rl)
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i < len(ll) {
			b.WriteString(ll[i])
		}
		if i < len(rl) {
			b.WriteString(rl[i])
		}
		if i < n-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// lipglossPlace centers a modal over a solid blue backdrop.
func lipglossPlace(th Theme, content string, w, h int) string {
	lines := strings.Split(content, "\n")
	ch := len(lines)
	cw := 0
	for _, l := range lines {
		if lw := visibleWidth(l); lw > cw {
			cw = lw
		}
	}
	top := (h - ch) / 2
	if top < 0 {
		top = 0
	}
	left := (w - cw) / 2
	if left < 0 {
		left = 0
	}

	blank := th.Bg.Render(strings.Repeat(" ", w))
	var b strings.Builder
	row := 0
	for ; row < top; row++ {
		b.WriteString(blank + "\n")
	}
	pad := th.Bg.Render(strings.Repeat(" ", left))
	for _, l := range lines {
		if row >= h {
			break
		}
		rest := w - left - visibleWidth(l)
		if rest < 0 {
			rest = 0
		}
		b.WriteString(pad + l + th.Bg.Render(strings.Repeat(" ", rest)) + "\n")
		row++
	}
	for ; row < h; row++ {
		b.WriteString(blank)
		if row < h-1 {
			b.WriteString("\n")
		}
	}
	out := b.String()
	return strings.TrimSuffix(out, "\n")
}
