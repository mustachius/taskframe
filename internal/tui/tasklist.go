package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
)

type listRow struct {
	t       *task.Task
	depth   int
	hasKids bool
}

// TaskList is the right panel: a sortable, expandable task tree.
type TaskList struct {
	roots    []*task.Task
	rows     []listRow
	cursor   int
	offset   int
	expanded map[int64]bool // default true; false = collapsed
	total    int
	sortMode task.SortMode
}

func NewTaskList() TaskList {
	return TaskList{expanded: map[int64]bool{}, sortMode: task.SortUrgency}
}

func (l *TaskList) SetSortMode(m task.SortMode) { l.sortMode = m }

func (l *TaskList) SetTasks(tasks []*task.Task) {
	prev := l.CursorID()
	l.total = len(tasks)
	l.roots = store.BuildTree(tasks, time.Now(), l.sortMode)
	l.rebuild()
	l.cursor = 0
	for i, r := range l.rows {
		if r.t.ID == prev {
			l.cursor = i
			break
		}
	}
}

func (l *TaskList) rebuild() {
	l.rows = l.rows[:0]
	var walk func(ts []*task.Task, depth int)
	walk = func(ts []*task.Task, depth int) {
		for _, t := range ts {
			l.rows = append(l.rows, listRow{t, depth, len(t.Children) > 0})
			if len(t.Children) > 0 && l.isExpanded(t.ID) {
				walk(t.Children, depth+1)
			}
		}
	}
	walk(l.roots, 0)
	if l.cursor >= len(l.rows) {
		l.cursor = len(l.rows) - 1
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
}

func (l *TaskList) isExpanded(id int64) bool {
	v, ok := l.expanded[id]
	return !ok || v
}

func (l *TaskList) CursorTask() *task.Task {
	if l.cursor < len(l.rows) {
		return l.rows[l.cursor].t
	}
	return nil
}

func (l *TaskList) CursorID() int64 {
	if t := l.CursorTask(); t != nil {
		return t.ID
	}
	return 0
}

func (l *TaskList) Move(delta int) {
	l.cursor += delta
	if l.cursor < 0 {
		l.cursor = 0
	}
	if l.cursor >= len(l.rows) {
		l.cursor = len(l.rows) - 1
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
}

func (l *TaskList) Home() { l.cursor = 0 }
func (l *TaskList) End() {
	if len(l.rows) > 0 {
		l.cursor = len(l.rows) - 1
	}
}

func (l *TaskList) Collapse() {
	if r := l.CursorTask(); r != nil {
		l.expanded[r.ID] = false
		l.rebuild()
	}
}

func (l *TaskList) Expand() {
	if r := l.CursorTask(); r != nil {
		l.expanded[r.ID] = true
		l.rebuild()
	}
}

func (l *TaskList) Count() int { return l.total }

// Lines renders visible rows for the given inner size.
func (l *TaskList) Lines(th Theme, w, h int, focused bool) []string {
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+h {
		l.offset = l.cursor - h + 1
	}
	now := time.Now()
	var lines []string
	for i := l.offset; i < len(l.rows) && len(lines) < h; i++ {
		lines = append(lines, l.renderRow(th, l.rows[i], w, now, focused && i == l.cursor))
	}
	if len(l.rows) == 0 {
		lines = append(lines, th.Dim.Render(" nenhuma tarefa — F2 para adicionar"))
	}
	return lines
}

func (l *TaskList) renderRow(th Theme, r listRow, w int, now time.Time, isCursor bool) string {
	t := r.t

	mark := "[ ]"
	switch t.Status {
	case task.StatusDone:
		mark = "[x]"
	case task.StatusDeleted:
		mark = "[-]"
	}
	if t.Status == task.StatusPending && t.IsActive() {
		mark = "[▶]"
	}

	arrow := " "
	if r.hasKids {
		if l.isExpanded(t.ID) {
			arrow = "▾"
		} else {
			arrow = "▸"
		}
	}

	due := "     "
	if t.Due != nil {
		due = t.Due.Format("02/01")
	}

	pri := " "
	if t.Priority != task.PriorityNone {
		pri = string(t.Priority)
	}

	indent := strings.Repeat("  ", r.depth)
	title := t.Title
	for _, tag := range t.Tags {
		title += " +" + tag
	}
	if t.Recur != "" {
		title += " ↻"
	}

	head := fmt.Sprintf(" %s%3d %s %s %s  ", arrow, t.ID, mark, pri, due)
	body := truncRunes(indent+title, w-len([]rune(head)))
	plain := head + body

	if isCursor {
		return th.Cursor.Render(padRowPlain(plain, w))
	}

	// segment styling: due date red when overdue, priority H highlighted
	headStyle := th.Text
	bodyStyle := th.Text
	switch {
	case t.Status == task.StatusDone:
		headStyle, bodyStyle = th.Dim, th.Done
	case t.Status == task.StatusDeleted:
		headStyle, bodyStyle = th.Dim, th.Dim
	case t.IsOverdue(now):
		headStyle = th.Overdue
	case t.Priority == task.PriorityHigh:
		headStyle = th.PrioHi
	}
	return headStyle.Render(head) + bodyStyle.Render(body)
}
