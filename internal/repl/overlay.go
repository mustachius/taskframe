package repl

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
	"github.com/jvsaga/taskframe/internal/ui"
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
			return m, m.storeCmd(func() resultMsg {
				if err := m.store.DeleteTask(id); err != nil {
					return errResult(m.th, err.Error())
				}
				return resultMsg{lines: []string{m.th.Dim.Render(fmt.Sprintf("  tarefa %d deletada", id))}, reload: true}
			})
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
		lines = append(lines, m.th.Dim.Render(" nenhuma tarefa"))
	}
	for i := m.offset; i < len(m.listRows) && i < m.offset+h; i++ {
		r := m.listRows[i]
		lines = append(lines, taskLine(m.th, r, w-2, now, i == m.cursor, m.ascii))
	}
	box := ui.DrawBoxChars(m.th, roundBox(m.ascii), m.listTitle, lines, w, len(lines)+2, true)
	hint := m.th.Dim.Render("  ↑↓ move · ←→ recolhe · a filho · enter abre · d conclui · x deleta · esc fecha")
	pos := ""
	if len(m.listRows) > 0 {
		pos = m.th.Dim.Render(fmt.Sprintf("  %d/%d", m.cursor+1, len(m.listRows)))
	}
	return box + "\n" + hint + pos
}

func (m model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.mode = modeList
		return m, nil
	case "up", "k":
		if m.detailScroll > 0 {
			m.detailScroll--
		}
	case "down", "j":
		m.detailScroll++
	}
	return m, nil
}

func (m model) viewDetail() string {
	w := min(m.w, 100)
	inner := min(len(m.detailLines), maxOverlayRows)
	if inner < 1 {
		inner = 1
	}
	if m.detailScroll > len(m.detailLines)-inner {
		m.detailScroll = len(m.detailLines) - inner
	}
	if m.detailScroll < 0 {
		m.detailScroll = 0
	}
	end := min(len(m.detailLines), m.detailScroll+inner)
	visible := m.detailLines[m.detailScroll:end]
	title := "tarefa"
	if m.detail != nil {
		title = fmt.Sprintf("tarefa %d", m.detail.ID)
	}
	box := ui.DrawBoxChars(m.th, roundBox(m.ascii), title, visible, w, len(visible)+2, true)
	return box + "\n" + m.th.Dim.Render("  ↑↓ rola · esc volta")
}

// detailBlock formats a task's fields, parent, subtasks, notes and activity
// for the detail view.
func detailBlock(th ui.Theme, t, parent *task.Task, children []*task.Task, notes []task.Note, acts []task.Activity, w int) []string {
	label := func(s string) string { return th.Dim.Render(ui.PadRowPlain(s, 16)) }
	val := func(s string) string { return th.Text.Render(s) }
	var lines []string
	add := func(s string) { lines = append(lines, s) }

	add(" " + th.TitleFocus.Render(ui.TruncRunes(t.Title, w-2)))
	add(" " + label("status") + val(string(t.Status)))
	if t.Start != nil {
		add(" " + label("iniciada") + th.Accent.Render("▶ "+t.Start.Format("02/01/2006 15:04")))
	}
	if parent != nil {
		add(" " + label("pai") + val(ui.TruncRunes(fmt.Sprintf("#%d %s", parent.ID, parent.Title), w-18)))
	}
	if t.Project != "" {
		add(" " + label("projeto") + val(t.Project))
	}
	if len(t.Tags) > 0 {
		add(" " + label("tags") + val("+"+strings.Join(t.Tags, " +")))
	}
	if t.Priority != task.PriorityNone {
		add(" " + label("prioridade") + val(string(t.Priority)))
	}
	if t.Due != nil {
		add(" " + label("vencimento") + val(t.Due.Format("02/01/2006")))
	}
	if t.Wait != nil {
		add(" " + label("aguardar até") + val(t.Wait.Format("02/01/2006")))
	}
	if t.Recur != "" {
		add(" " + label("recorrência") + val(t.Recur))
	}
	add(" " + label("criada") + val(t.CreatedAt.Format("02/01/2006 15:04")))
	if t.CompletedAt != nil {
		add(" " + label("concluída") + val(t.CompletedAt.Format("02/01/2006 15:04")))
	}
	if len(children) > 0 {
		done := 0
		for _, c := range children {
			if c.Status == task.StatusDone {
				done++
			}
		}
		add(" " + th.TitleFocus.Render(fmt.Sprintf("subtarefas %d/%d", done, len(children))))
		for _, c := range children {
			mark := "[ ]"
			if c.Status == task.StatusDone {
				mark = "[x]"
			}
			add(" " + th.Dim.Render(fmt.Sprintf("%s %d ", mark, c.ID)) + th.Text.Render(ui.TruncRunes(c.Title, w-16)))
		}
	}
	if len(notes) > 0 {
		add(" " + th.TitleFocus.Render("notas"))
		for _, n := range notes {
			add(" " + th.Dim.Render(n.CreatedAt.Format("02/01 15:04")+" ") + th.Text.Render(ui.TruncRunes(n.Body, w-16)))
		}
	}
	add(" " + th.TitleFocus.Render("histórico"))
	for _, a := range acts {
		add(" " + th.Dim.Render(a.TS.Format("02/01 15:04")+" ") + th.Text.Render(ui.TruncRunes(actDesc(a), w-16)))
	}
	return lines
}

func actDesc(a task.Activity) string {
	switch a.Kind {
	case "create":
		return "criada: " + a.NewVal
	case "done":
		return "concluída"
	case "delete":
		return "deletada"
	case "note":
		return "nota: " + a.NewVal
	case "modify":
		if a.OldVal == "" {
			return fmt.Sprintf("%s definido: %s", a.Field, a.NewVal)
		}
		return fmt.Sprintf("%s: %s → %s", a.Field, a.OldVal, a.NewVal)
	}
	return a.Kind
}
