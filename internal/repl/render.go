package repl

import (
	"fmt"
	"time"

	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
)

// foldGlyph returns the expand/collapse indicator for a row with subtasks.
func foldGlyph(r olRow, ascii bool) string {
	if !r.hasKids {
		return ""
	}
	switch {
	case r.collapsed && ascii:
		return "+ "
	case r.collapsed:
		return "▸ "
	case ascii:
		return "- "
	default:
		return "▾ "
	}
}

// taskLine formats one task as a single themed row for the overlay/echoes,
// with tree connectors and a fold indicator for nodes that have subtasks.
func taskLine(th ui.Theme, r olRow, w int, now time.Time, selected, ascii bool) string {
	t := r.t
	mark := "[ ]"
	switch t.Status {
	case task.StatusDone:
		mark = "[x]"
	case task.StatusDeleted:
		mark = "[-]"
	}
	if t.Status == task.StatusPending && t.IsActive() {
		mark = "[>]"
	}
	pri := " "
	if t.Priority != task.PriorityNone {
		pri = string(t.Priority)
	}
	due := "     "
	if t.Due != nil {
		due = t.Due.Format("02/01")
	}
	title := foldGlyph(r, ascii) + t.Title
	for _, tag := range t.Tags {
		title += " +" + tag
	}
	if t.Recur != "" {
		title += " ~"
	}
	if t.Status == task.StatusPending && t.IsActive() {
		title += " ·" + ui.FormatElapsed(now.Sub(*t.Start))
	}

	head := fmt.Sprintf(" %4d %s %s %s  ", t.ID, mark, pri, due)
	body := ui.TruncRunes(ui.TreePrefix(r.lastStack, ascii)+title, w-len([]rune(head)))

	if selected {
		return th.Cursor.Render(ui.PadRowPlain(head+body, w))
	}
	headStyle, bodyStyle := th.Text, th.Text
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

// plainTaskLine is the unstyled one-liner used in scrollback echoes.
func plainTaskLine(t *task.Task, now time.Time) string {
	mark := "[ ]"
	switch t.Status {
	case task.StatusDone:
		mark = "[x]"
	case task.StatusDeleted:
		mark = "[-]"
	}
	pri := "-"
	if t.Priority != task.PriorityNone {
		pri = string(t.Priority)
	}
	due := ""
	if t.Due != nil {
		due = t.Due.Format("02/01")
		if t.IsOverdue(now) {
			due += "!"
		}
	}
	s := fmt.Sprintf("  %d %s %s %-7s %s", t.ID, mark, pri, due, t.Title)
	if t.Project != "" {
		s += "  (" + t.Project + ")"
	}
	return s
}
