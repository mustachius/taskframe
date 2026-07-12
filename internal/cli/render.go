package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
)

// renderList prints a plain-text table, urgency-sorted, tree-indented.
// No ANSI styling: output must be pipe-friendly.
func renderList(tasks []*task.Task) {
	if len(tasks) == 0 {
		fmt.Println("nenhuma tarefa")
		return
	}
	now := time.Now()
	roots := store.BuildTree(tasks, now, task.SortUrgency)

	fmt.Printf("%-4s %-3s %-4s %-10s %-30s %s\n", "ID", "St", "Pri", "Due", "Project", "Title")
	fmt.Println(strings.Repeat("-", 78))
	var walk func(ts []*task.Task, depth int)
	walk = func(ts []*task.Task, depth int) {
		for _, t := range ts {
			indent := strings.Repeat("  ", depth)
			fmt.Printf("%-4d %-3s %-4s %-10s %-30s %s%s%s\n",
				t.ID, statusMark(t.Status), string(t.Priority), dueStr(t.Due, now),
				truncate(t.Project, 30), indent, titlePrefix(depth), tagsSuffix(t))
			walk(t.Children, depth+1)
		}
	}
	walk(roots, 0)
	fmt.Printf("\n%d tarefa(s)\n", len(tasks))
}

func statusMark(s task.Status) string {
	switch s {
	case task.StatusDone:
		return "[x]"
	case task.StatusDeleted:
		return "[-]"
	default:
		return "[ ]"
	}
}

func dueStr(due *time.Time, now time.Time) string {
	if due == nil {
		return ""
	}
	d := due.Format("02/01/2006")
	if due.Before(now) {
		d += "!"
	}
	return d
}

func titlePrefix(depth int) string {
	if depth > 0 {
		return "└─ "
	}
	return ""
}

func tagsSuffix(t *task.Task) string {
	s := t.Title
	for _, tag := range t.Tags {
		s += " +" + tag
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
