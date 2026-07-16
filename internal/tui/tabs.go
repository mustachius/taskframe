package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
)

// tabsHeight is the fixed height of the tab band (top, labels, junction line).
const tabsHeight = 3

// tabDef binds a tab to a named report; report "" is the unfiltered All tab.
type tabDef struct{ report, labelKey string }

// tabDefs lists the tab band left to right (same order the sidebar's virtual
// filters had). Keys 1-9 jump by position, so order is user-visible.
func tabDefs() []tabDef {
	return []tabDef{
		{"", "tab.all"},
		{"today", "sb.today"},
		{"overdue", "sb.overdue"},
		{"week", "sb.week"},
		{"active", "sb.active"},
		{"next", "sb.next"},
		{"waiting", "sb.waiting"},
	}
}

func tabLabels(lang i18n.Lang) []string {
	defs := tabDefs()
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = lang.T(d.labelKey)
	}
	return out
}

// tabSpan is the half-open cell range [x0,x1) a visible tab occupies on every
// tab-band row. The same spans drive rendering and mouse hit-testing.
type tabSpan struct{ idx, x0, x1 int }

// tabLayout fits the tab boxes into w columns. Every tab fits at the 80-col
// floor in both languages; below that a window anchored on the active tab
// slides, and clipL/clipR ask for the ‹ › edge markers.
func tabLayout(labels []string, active, w int) (spans []tabSpan, clipL, clipR bool) {
	costs := make([]int, len(labels))
	total := 0
	for i, l := range labels {
		costs[i] = len([]rune(l)) + 4 // │·label·│ plus the two border cells
		total += costs[i]
	}
	avail := w - 2 // one margin cell each side, reused by the clip markers
	start, end := 0, len(labels)
	if total > avail {
		start, end = active, active+1
		used := costs[active]
		for {
			grew := false
			if end < len(labels) && used+costs[end] <= avail {
				used += costs[end]
				end++
				grew = true
			}
			if start > 0 && used+costs[start-1] <= avail {
				start--
				used += costs[start]
				grew = true
			}
			if !grew {
				break
			}
		}
	}
	x := 1
	for i := start; i < end; i++ {
		spans = append(spans, tabSpan{idx: i, x0: x, x1: x + costs[i]})
		x += costs[i]
	}
	return spans, start > 0, end < len(labels)
}

// tabJoints returns the junction runes of the bottom band line for the
// theme's box set: up-left corner, up-right corner and tee.
func tabJoints(box ui.BoxChars) (lu, ru, tee string) {
	switch box.TL {
	case "╭":
		return "┘", "└", "┴"
	case "╔":
		return "╝", "╚", "╩"
	default:
		return "+", "+", "+"
	}
}

// renderTabs draws the 3-line tab band, every line exactly w cells. The
// active tab connects to the content below through a gap in the bottom line;
// alert (-1 = none) tints that tab's label with the overdue color.
func renderTabs(th Theme, labels []string, active, alert, w int) []string {
	spans, clipL, clipR := tabLayout(labels, active, w)
	lu, ru, tee := tabJoints(th.Box)
	h, v := th.Box.H, th.Box.V

	border := func(i int) lipgloss.Style {
		if i == active {
			return th.BorderFocus
		}
		return th.Border
	}
	label := func(i int) lipgloss.Style {
		switch {
		case i == active:
			return th.TitleFocus
		case i == alert:
			return th.Overdue
		default:
			return th.Dim
		}
	}

	var top, mid, bot strings.Builder
	top.WriteString(th.Bg.Render(" "))
	if clipL {
		mid.WriteString(th.Dim.Render("‹"))
	} else {
		mid.WriteString(th.Bg.Render(" "))
	}
	bot.WriteString(th.Border.Render(h))
	for _, sp := range spans {
		inner := sp.x1 - sp.x0 - 2
		top.WriteString(border(sp.idx).Render(th.Box.TL + strings.Repeat(h, inner) + th.Box.TR))
		mid.WriteString(border(sp.idx).Render(v))
		mid.WriteString(label(sp.idx).Render(" " + labels[sp.idx] + " "))
		mid.WriteString(border(sp.idx).Render(v))
		if sp.idx == active {
			bot.WriteString(th.BorderFocus.Render(lu))
			bot.WriteString(th.Bg.Render(strings.Repeat(" ", inner)))
			bot.WriteString(th.BorderFocus.Render(ru))
		} else {
			bot.WriteString(th.Border.Render(tee + strings.Repeat(h, inner) + tee))
		}
	}
	used := 1
	if len(spans) > 0 {
		used = spans[len(spans)-1].x1
	}
	topLine := padRow(top.String(), w, th.Bg)
	botLine := bot.String() + th.Border.Render(strings.Repeat(h, w-used))
	midLine := mid.String()
	if clipR {
		midLine += th.Bg.Render(strings.Repeat(" ", w-used-1)) + th.Dim.Render("›")
	} else {
		midLine = padRow(midLine, w, th.Bg)
	}
	return []string{topLine, midLine, botLine}
}

// renderTabBand renders the band for the app's current state.
func (a *App) renderTabBand() []string {
	alert := -1
	if a.overdueCount > 0 {
		for i, d := range tabDefs() {
			if d.report == "overdue" {
				alert = i
			}
		}
	}
	return renderTabs(a.th, tabLabels(a.lang), a.activeTab, alert, a.w)
}

// tabFilter returns the report filter of the active tab (zero for All).
func (a *App) tabFilter(now time.Time) task.Filter {
	name := tabDefs()[a.activeTab].report
	if name == "" {
		return task.Filter{}
	}
	r, _ := task.LookupReport(name)
	return r.Build(now)
}

// tabLimit returns the active report's display row cap (0 = unlimited).
// Reports carry their own limit (e.g. next = 15); store.List does not apply
// it, so the list panel caps its rows instead.
func (a *App) tabLimit() int {
	name := tabDefs()[a.activeTab].report
	if name == "" {
		return 0
	}
	r, _ := task.LookupReport(name)
	return r.Limit
}

// setTab activates tab i (wrapping) and refreshes the list.
func (a *App) setTab(i int) (tea.Model, tea.Cmd) {
	n := len(tabDefs())
	a.activeTab = ((i % n) + n) % n
	return a.applyFilters()
}
