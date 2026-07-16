package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
)

type sbKind int

// New kinds must be appended, never reordered — itemKey embeds the int value
// and it keeps the selection stable across reloads.
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
	sbActive
	sbNext
	sbContext
)

type sbItem struct {
	kind   sbKind
	label  string
	value  string // dotted project path (sbProject), tag (sbTag) or context name
	depth  int
	count  int
	done   int  // sbProject: completed tasks (feeds the progress bar)
	active bool // sbContext: this is the active context
}

// Sidebar is the left panel: project tree, virtual filters and tags.
type Sidebar struct {
	items  []sbItem
	cursor int
	offset int
}

// SetCounts rebuilds the item list from freshly loaded counts.
func (s *Sidebar) SetCounts(lang i18n.Lang, d sidebarData) {
	prev := s.selectedKey()
	s.items = s.items[:0]
	s.items = append(s.items, sbItem{kind: sbAll, label: lang.T("sb.all"), count: d.total})

	// every node in the dotted hierarchy, with counts aggregated upward
	nodes := map[string]store.ProjectCount{}
	for p, c := range d.counts {
		if p == "" {
			continue
		}
		parts := task.ProjectParts(p)
		for i := range parts {
			prefix := strings.Join(parts[:i+1], ".")
			n := nodes[prefix]
			n.Pending += c.Pending
			n.Done += c.Done
			nodes[prefix] = n
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
			depth: len(parts) - 1, count: nodes[p].Pending, done: nodes[p].Done,
		})
	}

	s.items = append(s.items,
		sbItem{kind: sbSeparator, label: lang.T("sb.sec.filters")},
		sbItem{kind: sbToday, label: lang.T("sb.today"), count: d.today},
		sbItem{kind: sbOverdue, label: lang.T("sb.overdue"), count: d.overdue},
		sbItem{kind: sbWeek, label: lang.T("sb.week"), count: d.week},
		sbItem{kind: sbActive, label: lang.T("sb.active"), count: d.active},
		sbItem{kind: sbNext, label: lang.T("sb.next"), count: d.next},
		sbItem{kind: sbWaiting, label: lang.T("sb.waiting"), count: d.waiting},
	)

	if len(d.tags) > 0 {
		s.items = append(s.items, sbItem{kind: sbSeparator, label: lang.T("sb.sec.tags")})
		tags := make([]string, 0, len(d.tags))
		for t := range d.tags {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		for _, t := range tags {
			s.items = append(s.items, sbItem{kind: sbTag, label: "+" + t, value: t, count: d.tags[t]})
		}
	}

	if len(d.contexts) > 0 {
		s.items = append(s.items, sbItem{kind: sbSeparator, label: lang.T("sb.sec.contexts")})
		for _, c := range d.contexts {
			s.items = append(s.items, sbItem{
				kind: sbContext, label: "@" + c.name, value: c.name,
				count: c.count, active: c.name == d.activeCtx,
			})
		}
	}

	s.items = append(s.items,
		sbItem{kind: sbSeparator, label: lang.T("sb.sec.archive")},
		sbItem{kind: sbDone, label: lang.T("sb.done"), count: d.done},
		sbItem{kind: sbDeleted, label: lang.T("sb.deleted"), count: d.del},
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

// report returns the base filter of a named report (shared source of truth).
func report(name string, now time.Time) task.Filter {
	r, _ := task.LookupReport(name)
	return r.Build(now)
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
		return report("today", now)
	case sbOverdue:
		return report("overdue", now)
	case sbWeek:
		return report("week", now)
	case sbActive:
		return report("active", now)
	case sbNext:
		return report("next", now)
	case sbWaiting:
		return report("waiting", now)
	case sbTag:
		return task.Filter{Tags: []string{it.value}}
	case sbDone:
		return task.Filter{Status: task.StatusDone}
	case sbDeleted:
		return task.Filter{Status: task.StatusDeleted}
	default:
		// sbContext rows included: resting on one filters nothing extra —
		// activation is a deliberate Enter (applySidebar runs on every move,
		// so Filter() must stay free of side effects)
		return task.Filter{}
	}
}

// Title returns a human label for the current selection (list panel title).
func (s *Sidebar) Title(lang i18n.Lang) string {
	if s.cursor >= len(s.items) {
		return lang.T("sb.title.tasks")
	}
	switch it := s.items[s.cursor]; it.kind {
	case sbProject:
		return lang.T("sb.title.of") + it.value
	case sbToday:
		return lang.T("sb.today")
	case sbOverdue:
		return lang.T("sb.overdue")
	case sbWeek:
		return lang.T("sb.title.week")
	case sbActive:
		return lang.T("sb.active")
	case sbNext:
		return lang.T("sb.next")
	case sbWaiting:
		return lang.T("sb.waiting")
	case sbTag:
		return lang.T("sb.title.of") + "+" + it.value
	case sbContext:
		return lang.T("sb.title.of") + "@" + it.value
	case sbDone:
		return lang.T("sb.done")
	case sbDeleted:
		return lang.T("sb.deleted")
	default:
		return lang.T("sb.title.tasks")
	}
}

// Limit returns the row cap of the selected report view (0 = unlimited).
// Reports carry their own display limit (e.g. next = 15); store.List does not
// apply it, so the list panel caps its rows instead.
func (s *Sidebar) Limit() int {
	if s.cursor < len(s.items) && s.items[s.cursor].kind == sbNext {
		if r, ok := task.LookupReport("next"); ok {
			return r.Limit
		}
	}
	return 0
}

// CurrentProject returns the selected project path ("" when not on one).
func (s *Sidebar) CurrentProject() string {
	if s.cursor < len(s.items) && s.items[s.cursor].kind == sbProject {
		return s.items[s.cursor].value
	}
	return ""
}

// SetCursor moves the selection to item i (mouse click); refuses separators
// and out-of-range indexes.
func (s *Sidebar) SetCursor(i int) bool {
	if i < 0 || i >= len(s.items) || s.items[i].kind == sbSeparator {
		return false
	}
	s.cursor = i
	return true
}

// CurrentContext returns the context name under the cursor, if any.
func (s *Sidebar) CurrentContext() (string, bool) {
	if s.cursor < len(s.items) && s.items[s.cursor].kind == sbContext {
		return s.items[s.cursor].value, true
	}
	return "", false
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
			if it.label == "" {
				lines = append(lines, th.Border.Render(strings.Repeat("─", w)))
				continue
			}
			// labeled section header: ─ label ────
			name := truncRunes(it.label, w-4)
			rest := w - len([]rune(name)) - 3
			if rest < 0 {
				rest = 0
			}
			lines = append(lines, th.Border.Render("─ ")+th.Title.Render(name)+
				th.Border.Render(" "+strings.Repeat("─", rest)))
			continue
		}
		count := fmt.Sprintf("%d", it.count)
		indent := strings.Repeat("  ", it.depth)
		name := it.label
		if it.kind == sbContext && it.active {
			name = "● " + name
		}
		// project rows carry a done/total progress bar when the width allows
		// (the label must keep a readable minimum after making room for it)
		barW := 0
		frac := 0.0
		if tot := it.count + it.done; it.kind == sbProject && tot > 0 && w-len(count)-9 >= 8 {
			barW = 6
			frac = float64(it.done) / float64(tot)
		}
		labelMax := w - len(count) - 2
		if barW > 0 {
			labelMax -= barW + 1
		}
		label := truncRunes(indent+name, labelMax)
		gap := w - len([]rune(label)) - len(count) - 1
		if barW > 0 {
			gap -= barW + 1
		}
		if gap < 1 {
			gap = 1
		}
		style := th.Text
		emphasized := false // keep the count in the row color (e.g. overdue red)
		switch it.kind {
		case sbDone, sbDeleted:
			style = th.Dim
			emphasized = true
		case sbOverdue:
			if it.count > 0 {
				style = th.Overdue
				emphasized = true
			}
		case sbTag:
			style = th.Accent
		case sbContext:
			if it.active {
				style = th.Accent
			}
		}
		if i == s.cursor {
			// cursor rows: plain string under one whole-row style — nesting
			// styled segments would break the highlight mid-row
			style = th.Accent
			if focused {
				style = th.Cursor
			}
			row := " " + label + strings.Repeat(" ", gap-1) + count + " "
			if barW > 0 {
				row += ui.ProgressGlyphs(frac, barW, th.ASCII()) + " "
			}
			lines = append(lines, style.Render(truncRunes(row, w)))
			continue
		}
		countStyle := th.Dim
		if emphasized {
			countStyle = style
		}
		out := style.Render(" "+label+strings.Repeat(" ", gap-1)) + countStyle.Render(count+" ")
		if barW > 0 {
			out += ui.ProgressBar(frac, barW, th) + style.Render(" ")
		}
		lines = append(lines, out)
	}
	return lines
}
