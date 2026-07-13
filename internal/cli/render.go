package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
	"github.com/jvsaga/taskframe/internal/ui"
)

type cliRow struct {
	t         *task.Task
	lastStack []bool
}

// renderList prints a plain-text table, tree-indented, sorted by sortMode.
// limit > 0 caps the number of displayed rows. No ANSI: output is pipe-friendly.
func renderList(tasks []*task.Task, sortMode task.SortMode, limit int) {
	if len(tasks) == 0 {
		fmt.Println("nenhuma tarefa")
		return
	}
	now := time.Now()
	roots := store.BuildTree(tasks, now, sortMode)

	var rows []cliRow
	var walk func(ts []*task.Task, trunk []bool, depth int)
	walk = func(ts []*task.Task, trunk []bool, depth int) {
		for i, t := range ts {
			var ls []bool
			if depth > 0 {
				ls = append(append([]bool{}, trunk...), i == len(ts)-1)
			}
			rows = append(rows, cliRow{t, ls})
			walk(t.Children, ls, depth+1)
		}
	}
	walk(roots, nil, 0)

	truncated := false
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
		truncated = true
	}

	fmt.Printf("%-4s %-3s %-4s %-10s %-30s %s\n", "ID", "St", "Pri", "Due", "Project", "Title")
	fmt.Println(strings.Repeat("-", 78))
	for _, r := range rows {
		fmt.Printf("%-4d %-3s %-4s %-10s %-30s %s%s\n",
			r.t.ID, mark(r.t), string(r.t.Priority), dueStr(r.t.Due, now),
			truncate(r.t.Project, 30), ui.TreePrefix(r.lastStack, false), tagsSuffix(r.t))
	}
	if truncated {
		fmt.Printf("\n%d de %d tarefa(s) (limite %d)\n", len(rows), len(tasks), limit)
	} else {
		fmt.Printf("\n%d tarefa(s)\n", len(rows))
	}
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

// mark is statusMark plus an active indicator for started pending tasks.
func mark(t *task.Task) string {
	if t.Status == task.StatusPending && t.IsActive() {
		return "[>]"
	}
	return statusMark(t.Status)
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
