package repl

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
)

const maxOverlayRows = 14

type olRow struct {
	t         *task.Task
	lastStack []bool // per-level "is last sibling", for tree connectors
	hasKids   bool
	collapsed bool
}

// flattenTree builds tree-connected overlay rows from a flat task list,
// skipping the subtrees of collapsed nodes (expanded[id]==false).
func flattenTree(tasks []*task.Task, sortMode task.SortMode, expanded map[int64]bool) []olRow {
	roots := store.BuildTree(tasks, time.Now(), sortMode)
	var rows []olRow
	var walk func(ts []*task.Task, trunk []bool, depth int)
	walk = func(ts []*task.Task, trunk []bool, depth int) {
		for i, t := range ts {
			var ls []bool
			if depth > 0 {
				ls = append(append([]bool{}, trunk...), i == len(ts)-1)
			}
			hasKids := len(t.Children) > 0
			collapsed := hasKids && !isExpanded(expanded, t.ID)
			rows = append(rows, olRow{t: t, lastStack: ls, hasKids: hasKids, collapsed: collapsed})
			if hasKids && !collapsed {
				walk(t.Children, ls, depth+1)
			}
		}
	}
	walk(roots, nil, 0)
	return rows
}

func isExpanded(expanded map[int64]bool, id int64) bool {
	v, ok := expanded[id]
	return !ok || v
}

func (m model) cursorTask() *task.Task {
	if m.cursor >= 0 && m.cursor < len(m.listRows) {
		return m.listRows[m.cursor].t
	}
	return nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.flash = "" // any key dismisses the transient confirmation
	switch msg.String() {
	case "esc", "q":
		m.mode = modePrompt
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.listRows)-1 {
			m.cursor++
		}
	case "pgup":
		m.cursor = max(0, m.cursor-maxOverlayRows)
	case "pgdown":
		m.cursor = min(len(m.listRows)-1, m.cursor+maxOverlayRows)
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(m.listRows) - 1
	case "left", "h":
		if t := m.cursorTask(); t != nil {
			m.expanded[t.ID] = false
			m.rebuildList()
		}
	case "right", "l":
		if t := m.cursorTask(); t != nil {
			m.expanded[t.ID] = true
			m.rebuildList()
		}
	case "a":
		if t := m.cursorTask(); t != nil {
			return m.startAddChild(t), nil
		}
	case "n":
		if t := m.cursorTask(); t != nil {
			return m.startAddNote(t, modeList), nil
		}
	case "e":
		if t := m.cursorTask(); t != nil {
			return m.startEdit(t, modeList), nil
		}
	case "enter":
		if t := m.cursorTask(); t != nil {
			return m, m.openDetailCmd(t.ID)
		}
	case "d", " ":
		if t := m.cursorTask(); t != nil {
			return m, m.toggleFromList(t)
		}
	case "x":
		if t := m.cursorTask(); t != nil {
			id := t.ID
			m.flash = m.lang.Tf("status.taskDeleted", id)
			filter := m.listFilter
			s, th := m.store, m.th
			return m, func() tea.Msg {
				if err := s.DeleteTask(id); err != nil {
					return errResult(th, err.Error())
				}
				tasks, err := s.List(filter) // re-query so the row disappears
				if err != nil {
					return errResult(th, err.Error())
				}
				return listRefreshMsg{tasks: tasks}
			}
		}
	}
	return m, nil
}

// toggleFromList completes/reopens a task and refreshes the overlay in place.
func (m model) toggleFromList(t *task.Task) tea.Cmd {
	id, status := t.ID, t.Status
	filter := m.listFilter
	th := m.th
	s := m.store
	return func() tea.Msg {
		switch status {
		case task.StatusPending:
			if _, err := s.CompleteTask(id); err != nil {
				return errResult(th, err.Error())
			}
		case task.StatusDone:
			if err := s.ReopenTask(id); err != nil {
				return errResult(th, err.Error())
			}
		}
		tasks, err := s.List(filter)
		if err != nil {
			return errResult(th, err.Error())
		}
		return listRefreshMsg{tasks: tasks}
	}
}

func (m model) openDetailCmd(id int64) tea.Cmd {
	s, th := m.store, m.th
	return func() tea.Msg {
		t, err := s.GetTask(id)
		if err != nil {
			return errResult(th, err.Error())
		}
		var parent *task.Task
		if t.ParentID != 0 {
			if p, perr := s.GetTask(t.ParentID); perr == nil {
				parent = p
			}
		}
		children, err := s.Children(id)
		if err != nil {
			return errResult(th, err.Error())
		}
		notes, err := s.Notes(id)
		if err != nil {
			return errResult(th, err.Error())
		}
		acts, err := s.TaskActivity(id)
		if err != nil {
			return errResult(th, err.Error())
		}
		return detailLoadedMsg{t: t, parent: parent, children: children, notes: notes, acts: acts}
	}
}

func (m model) viewList() string {
	now := time.Now()
	h := min(len(m.listRows), maxOverlayRows)
	if h < 1 {
		h = 1
	}
	// window around the cursor
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	w := min(m.w, 100)
	var lines []string
	if len(m.listRows) == 0 {
		lines = append(lines, m.th.Dim.Render(m.lang.T("overlay.empty")))
	}
	for i := m.offset; i < len(m.listRows) && i < m.offset+h; i++ {
		r := m.listRows[i]
		lines = append(lines, taskLine(m.th, r, w-2, now, i == m.cursor, m.ascii))
	}
	box := ui.DrawBoxChars(m.th, roundBox(m.ascii), m.listTitle, lines, w, len(lines)+2, true)
	hint := m.th.Dim.Render(m.lang.T("overlay.hint"))
	pos := ""
	if len(m.listRows) > 0 {
		pos = m.th.Dim.Render(fmt.Sprintf("  %d/%d", m.cursor+1, len(m.listRows)))
	}
	out := box + "\n" + hint + pos
	if m.flash != "" {
		out += "\n" + m.th.Accent.Render(m.flash) // flash strings already indent
	}
	return out
}

func (m model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.mode = modeList
		return m, nil
	case "n":
		if m.detail != nil {
			return m.startAddNote(m.detail, modeDetail), nil
		}
	case "e":
		if m.detail != nil {
			return m.startEdit(m.detail, modeDetail), nil
		}
	}
	var cmd tea.Cmd
	m.detailVP, cmd = m.detailVP.Update(msg)
	return m, cmd
}

func (m model) viewDetail() string {
	w := min(m.w, 100)
	title := m.lang.T("detail.title")
	if m.detail != nil {
		title = m.lang.Tf("detail.titleN", m.detail.ID)
	}
	visible := strings.Split(m.detailVP.View(), "\n")
	box := ui.DrawBoxChars(m.th, roundBox(m.ascii), title, visible, w, len(visible)+2, true)
	return box + "\n" + m.th.Dim.Render(m.lang.T("detail.footer"))
}

// detailBlock formats a task's fields, parent, subtasks, notes and activity
// for the detail view.
func detailBlock(th ui.Theme, lang i18n.Lang, now time.Time, t, parent *task.Task, children []*task.Task, notes []task.Note, acts []task.Activity, w int, ascii bool) []string {
	label := func(s string) string { return th.Dim.Render(ui.PadRowPlain(s, 16)) }
	val := func(s string) string { return th.Text.Render(s) }
	var lines []string
	add := func(s string) { lines = append(lines, s) }

	add(" " + th.TitleFocus.Render(ui.TruncRunes(t.Title, w-2)))
	add(" " + label(lang.T("lbl.status")) + val(string(t.Status)))
	if t.Start != nil {
		add(" " + label(lang.T("lbl.started")) +
			th.Accent.Render(t.Start.Format("02/01/2006 15:04")+" · "+ui.FormatElapsed(now.Sub(*t.Start))))
	}
	if parent != nil {
		add(" " + label(lang.T("lbl.parent")) + val(ui.TruncRunes(fmt.Sprintf("#%d %s", parent.ID, parent.Title), w-18)))
	}
	if t.Project != "" {
		add(" " + label(lang.T("lbl.project")) + val(t.Project))
	}
	if len(t.Tags) > 0 {
		add(" " + label(lang.T("lbl.tags")) + val("+"+strings.Join(t.Tags, " +")))
	}
	if t.Priority != task.PriorityNone {
		add(" " + label(lang.T("lbl.priority")) + val(string(t.Priority)))
	}
	if t.Due != nil {
		add(" " + label(lang.T("lbl.due")) + val(t.Due.Format("02/01/2006")))
	}
	if t.Wait != nil {
		add(" " + label(lang.T("lbl.waitUntil")) + val(t.Wait.Format("02/01/2006")))
	}
	if t.Recur != "" {
		add(" " + label(lang.T("lbl.recurrence")) + val(t.Recur))
	}
	add(" " + label(lang.T("lbl.created")) + val(t.CreatedAt.Format("02/01/2006 15:04")))
	if t.CompletedAt != nil {
		add(" " + label(lang.T("lbl.completed")) + val(t.CompletedAt.Format("02/01/2006 15:04")))
	}
	if len(children) > 0 {
		done := 0
		for _, c := range children {
			if c.Status == task.StatusDone {
				done++
			}
		}
		bar := ui.ProgressBar(float64(done)/float64(len(children)), 12, th)
		add(" " + th.TitleFocus.Render(lang.Tf("detail.subtasks", done, len(children))) + "  " + bar)
		for _, c := range children {
			mark := "[ ]"
			if c.Status == task.StatusDone {
				mark = "[x]"
			}
			add(" " + th.Dim.Render(fmt.Sprintf("%s %d ", mark, c.ID)) + th.Text.Render(ui.TruncRunes(c.Title, w-16)))
		}
	}
	if len(notes) > 0 {
		add(" " + th.TitleFocus.Render(lang.T("detail.notes")))
		var nb strings.Builder
		for _, n := range notes {
			nb.WriteString("**" + n.CreatedAt.Format("02/01 15:04") + "** — " + n.Body + "\n\n")
		}
		mdStyle := th.MDStyle
		if ascii {
			mdStyle = "notty"
		}
		md := ui.RenderMarkdown(nb.String(), w-2, mdStyle)
		for _, ln := range strings.Split(strings.TrimRight(md, "\n"), "\n") {
			add(ln)
		}
	}
	add(" " + th.TitleFocus.Render(lang.T("detail.history")))
	for _, a := range acts {
		add(" " + th.Dim.Render(a.TS.Format("02/01 15:04")+" ") + th.Text.Render(ui.TruncRunes(actDesc(lang, a), w-16)))
	}
	return lines
}

func actDesc(lang i18n.Lang, a task.Activity) string {
	switch a.Kind {
	case "create":
		return lang.T("act.created") + a.NewVal
	case "done":
		return lang.T("act.done")
	case "delete":
		return lang.T("act.deleted")
	case "note":
		return lang.T("act.note")
	case "modify":
		if a.OldVal == "" {
			return lang.Tf("act.setTo", a.Field, a.NewVal)
		}
		return lang.Tf("act.changed", a.Field, a.OldVal, a.NewVal)
	}
	return a.Kind
}
