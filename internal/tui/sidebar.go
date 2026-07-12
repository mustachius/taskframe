package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jvsaga/taskframe/internal/task"
)

type sbKind int

const (
	sbAll sbKind = iota
	sbProject
	sbToday
	sbOverdue
	sbWeek
	sbWaiting
	sbTag
	sbDone
	sbDeleted
	sbSeparator
)

type sbItem struct {
	kind  sbKind
	label string
	value string // dotted project path (sbProject) or tag name (sbTag)
	depth int
	count int
}

// Sidebar is the left panel: project tree, virtual filters and tags.
type Sidebar struct {
	items  []sbItem
	cursor int
	offset int
}

// SetCounts rebuilds the item list from freshly loaded counts.
func (s *Sidebar) SetCounts(d sidebarData) {
	prev := s.selectedKey()
	s.items = s.items[:0]
	s.items = append(s.items, sbItem{kind: sbAll, label: "(todas)", count: d.total})

	// every node in the dotted hierarchy, with counts aggregated upward
	nodes := map[string]int{}
	for p, n := range d.counts {
		if p == "" {
			continue
		}
		parts := task.ProjectParts(p)
		for i := range parts {
			prefix := strings.Join(parts[:i+1], ".")
			nodes[prefix] += n
		}
	}
	paths := make([]string, 0, len(nodes))
	for p := range nodes {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		parts := task.ProjectParts(p)
		s.items = append(s.items, sbItem{
			kind: sbProject, label: parts[len(parts)-1], value: p,
			depth: len(parts) - 1, count: nodes[p],
		})
	}

	s.items = append(s.items,
		sbItem{kind: sbSeparator},
		sbItem{kind: sbToday, label: "Hoje", count: d.today},
		sbItem{kind: sbOverdue, label: "Atrasadas", count: d.overdue},
		sbItem{kind: sbWeek, label: "Semana", count: d.week},
		sbItem{kind: sbWaiting, label: "Aguardando", count: d.waiting},
	)

	if len(d.tags) > 0 {
		s.items = append(s.items, sbItem{kind: sbSeparator})
		tags := make([]string, 0, len(d.tags))
		for t := range d.tags {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		for _, t := range tags {
			s.items = append(s.items, sbItem{kind: sbTag, label: "+" + t, value: t, count: d.tags[t]})
		}
	}

	s.items = append(s.items,
		sbItem{kind: sbSeparator},
		sbItem{kind: sbDone, label: "Concluídas", count: d.done},
		sbItem{kind: sbDeleted, label: "Deletadas", count: d.del},
	)

	// keep selection stable across reloads
	s.cursor = 0
	for i, it := range s.items {
		if itemKey(it) == prev {
			s.cursor = i
			break
		}
	}
}

func itemKey(it sbItem) string { return fmt.Sprintf("%d:%s", it.kind, it.value) }

func (s *Sidebar) selectedKey() string {
	if s.cursor < len(s.items) {
		return itemKey(s.items[s.cursor])
	}
	return ""
}

func (s *Sidebar) Move(delta int) {
	n := len(s.items)
	if n == 0 {
		return
	}
	c := s.cursor
	for {
		c += delta
		if c < 0 || c >= n {
			return
		}
		if s.items[c].kind != sbSeparator {
			s.cursor = c
			return
		}
	}
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
}

// Filter returns the task filter for the selected item.
func (s *Sidebar) Filter() task.Filter {
	if s.cursor >= len(s.items) {
		return task.Filter{}
	}
	now := time.Now()
	switch it := s.items[s.cursor]; it.kind {
	case sbProject:
		return task.Filter{Project: it.value}
	case sbToday:
		d := endOfDay(now)
		return task.Filter{DueBefore: &d}
	case sbOverdue:
		return task.Filter{DueBefore: &now}
	case sbWeek:
		d := endOfDay(now.AddDate(0, 0, 7))
		return task.Filter{DueBefore: &d}
	case sbWaiting:
		return task.Filter{WaitingOnly: true}
	case sbTag:
		return task.Filter{Tags: []string{it.value}}
	case sbDone:
		return task.Filter{Status: task.StatusDone}
	case sbDeleted:
		return task.Filter{Status: task.StatusDeleted}
	default:
		return task.Filter{}
	}
}

// Title returns a human label for the current selection (list panel title).
func (s *Sidebar) Title() string {
	if s.cursor >= len(s.items) {
		return "Tarefas"
	}
	switch it := s.items[s.cursor]; it.kind {
	case sbProject:
		return "Tarefas: " + it.value
	case sbToday:
		return "Hoje"
	case sbOverdue:
		return "Atrasadas"
	case sbWeek:
		return "Próximos 7 dias"
	case sbWaiting:
		return "Aguardando"
	case sbTag:
		return "Tarefas: +" + it.value
	case sbDone:
		return "Concluídas"
	case sbDeleted:
		return "Deletadas"
	default:
		return "Tarefas"
	}
}

// CurrentProject returns the selected project path ("" when not on one).
func (s *Sidebar) CurrentProject() string {
	if s.cursor < len(s.items) && s.items[s.cursor].kind == sbProject {
		return s.items[s.cursor].value
	}
	return ""
}

// Lines renders the sidebar content for the given inner size.
func (s *Sidebar) Lines(th Theme, w, h int, focused bool) []string {
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+h {
		s.offset = s.cursor - h + 1
	}
	var lines []string
	for i := s.offset; i < len(s.items) && len(lines) < h; i++ {
		it := s.items[i]
		if it.kind == sbSeparator {
			lines = append(lines, th.Border.Render(strings.Repeat("─", w)))
			continue
		}
		count := fmt.Sprintf("%d", it.count)
		indent := strings.Repeat("  ", it.depth)
		label := truncRunes(indent+it.label, w-len(count)-2)
		gap := w - len([]rune(label)) - len(count) - 1
		if gap < 1 {
			gap = 1
		}
		row := " " + label + strings.Repeat(" ", gap-1) + count + " "
		row = truncRunes(row, w)
		style := th.Text
		switch it.kind {
		case sbDone, sbDeleted:
			style = th.Dim
		case sbOverdue:
			if it.count > 0 {
				style = th.Overdue
			}
		}
		if i == s.cursor && focused {
			style = th.Cursor
		} else if i == s.cursor {
			style = th.Accent
		}
		lines = append(lines, style.Render(row))
	}
	return lines
}
