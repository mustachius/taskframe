package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
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
	limit    int // cap visible rows (report display limit); 0 = unlimited
	sortMode task.SortMode
}

func NewTaskList() TaskList {
	return TaskList{expanded: map[int64]bool{}, sortMode: task.SortUrgency}
}

func (l *TaskList) SetSortMode(m task.SortMode) { l.sortMode = m }

// SetLimit caps the rendered rows (0 = unlimited) and rebuilds.
func (l *TaskList) SetLimit(n int) {
	if l.limit == n {
		return
	}
	l.limit = n
	l.rebuild()
}

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
			if l.limit > 0 && len(l.rows) >= l.limit {
				return
			}
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

// MoveTo puts the cursor on absolute row i, clamped like Move (mouse click).
func (l *TaskList) MoveTo(i int) {
	l.cursor = i
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
func (l *TaskList) Lines(th Theme, lang i18n.Lang, w, h int, focused bool) []string {
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
		lines = append(lines, th.Dim.Render(lang.T("tasklist.empty")))
	}
	return lines
}

func (l *TaskList) renderRow(th Theme, r listRow, w int, now time.Time, isCursor bool) string {
	t := r.t

	mark := statusMark(t, th.ASCII())

	arrow := " "
	if r.hasKids {
		if l.isExpanded(t.ID) {
			arrow = "▾"
			if th.ASCII() {
				arrow = "v"
			}
		} else {
			arrow = "▸"
			if th.ASCII() {
				arrow = ">"
			}
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
	elapsed := ""
	if t.Status == task.StatusPending && t.IsActive() {
		elapsed = " ·" + ui.FormatElapsed(now.Sub(*t.Start))
	}
	title := t.Title
	for _, tag := range t.Tags {
		title += " +" + tag
	}
	if t.Recur != "" {
		title += " ~"
	}
	title += elapsed

	head := fmt.Sprintf(" %s%3d %s %s %s  ", arrow, t.ID, mark, pri, due)

	// the cursor row stays a plain string under one whole-row style — nesting
	// styled segments inside it would cut the highlight mid-row
	if isCursor {
		body := truncRunes(indent+title, w-len([]rune(head)))
		return th.Cursor.Render(padRowPlain(head+body, w))
	}

	switch t.Status {
	case task.StatusDone:
		body := truncRunes(indent+title, w-len([]rune(head)))
		return th.Dim.Render(head) + th.Done.Render(body)
	case task.StatusDeleted:
		body := truncRunes(indent+title, w-len([]rune(head)))
		return th.Dim.Render(head + body)
	}

	// pending rows: segment-level colors — active mark, priority, due, tags
	markStyle := th.Text
	if t.IsActive() {
		markStyle = th.Accent
	}
	priStyle := th.Text
	switch t.Priority {
	case task.PriorityHigh:
		priStyle = th.PrioHi
	case task.PriorityMed:
		priStyle = th.Accent
	}
	dueStyle := th.Text
	if t.Due != nil {
		switch {
		case t.IsOverdue(now):
			dueStyle = th.Overdue
		case !t.Due.After(task.EndOfDay(now)):
			dueStyle = th.Warn
		}
	}
	segs := []seg{
		{fmt.Sprintf(" %s%3d ", arrow, t.ID), th.Dim},
		{mark, markStyle},
		{" ", th.Text},
		{pri, priStyle},
		{" ", th.Text},
		{due, dueStyle},
		{"  " + indent + t.Title, th.Text},
	}
	for _, tag := range t.Tags {
		segs = append(segs, seg{" +" + tag, th.Accent})
	}
	if t.Recur != "" {
		segs = append(segs, seg{" ~", th.Dim})
	}
	if elapsed != "" {
		segs = append(segs, seg{elapsed, th.Accent})
	}
	return renderSegs(segs, w)
}

// statusMark is the 1-cell checklist glyph for a task's state: ○ pending,
// ✓ done, × deleted, ● active (started). ASCII falls back to o/x/-/*
// (* so the active mark cannot collide with the > fold arrow).
func statusMark(t *task.Task, ascii bool) string {
	switch {
	case t.Status == task.StatusDone:
		if ascii {
			return "x"
		}
		return "✓"
	case t.Status == task.StatusDeleted:
		if ascii {
			return "-"
		}
		return "×"
	case t.IsActive():
		if ascii {
			return "*"
		}
		return "●"
	default:
		if ascii {
			return "o"
		}
		return "○"
	}
}

// seg is one independently styled slice of a rendered row.
type seg struct {
	text  string
	style lipgloss.Style
}

// renderSegs styles segments left-to-right, truncating (…) at max cells.
func renderSegs(segs []seg, max int) string {
	var b strings.Builder
	used := 0
	for _, sg := range segs {
		if used >= max {
			break
		}
		txt := truncRunes(sg.text, max-used)
		b.WriteString(sg.style.Render(txt))
		used += len([]rune(txt))
	}
	return b.String()
}
