package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/ui"
)

// wheelStep is how many list/viewport lines one wheel notch scrolls.
const wheelStep = 3

// headerHeight mirrors renderHeader's branching without rendering — the art
// heights are structural, and gradient-coloring six lines per wheel event
// would be waste. TestHeaderHeightMatchesRender pins the two in sync.
func (a *App) headerHeight() int {
	if a.w >= headerFullMinW && a.h >= headerFullMinH {
		if a.ascii {
			return len(ui.WordmarkASCII)
		}
		return len(ui.WordmarkShadow)
	}
	return 1
}

// contentTop is the Y of the first panel content row: header + tab band +
// panel top border. Render, hit-testing and the tests all go through it so
// the geometry can never drift apart.
func (a *App) contentTop() int {
	return a.headerHeight() + tabsHeight + 1
}

// modalScroller is implemented by modals that can wheel-scroll (Detail, Read).
type modalScroller interface{ scrollBy(delta int) tea.Cmd }

// handleMouse routes wheel and left-click events. Click coordinates map to
// panel rows using the offsets from the last render — safe because View()
// always follows Update, so the on-screen mapping is exactly offset+row.
func (a *App) handleMouse(m tea.MouseMsg) (tea.Model, tea.Cmd) {
	if a.w < 60 || a.h < 12 {
		return a, nil // View short-circuits to the "window small" screen
	}

	// MouseMsg is a named copy of MouseEvent; methods live on the latter
	ev := tea.MouseEvent(m)

	if a.modal != nil {
		if ev.IsWheel() {
			delta := wheelStep
			if m.Button == tea.MouseButtonWheelUp {
				delta = -wheelStep
			}
			if ms, ok := a.modal.(modalScroller); ok {
				return a, ms.scrollBy(delta)
			}
		}
		return a, nil // clicks/other buttons never reach modals
	}

	if ev.IsWheel() {
		delta := 1
		if m.Button == tea.MouseButtonWheelUp {
			delta = -1
		}
		if m.X < sidebarWidth {
			a.sidebar.Move(delta)
			return a.applyFilters()
		}
		a.list.Move(delta * wheelStep)
		return a, nil
	}

	if m.Action != tea.MouseActionPress || m.Button != tea.MouseButtonLeft {
		return a, nil
	}
	hdr := a.headerHeight()

	// tab band: clicking a tab activates it (same spans the renderer used)
	if m.Y >= hdr && m.Y < hdr+tabsHeight {
		spans, _, _ := tabLayout(tabLabels(a.lang), a.activeTab, a.w)
		for _, sp := range spans {
			if m.X >= sp.x0 && m.X < sp.x1 {
				return a.setTab(sp.idx)
			}
		}
		return a, nil
	}

	panelH := a.h - 2 - hdr - tabsHeight // same formula as View()
	row := m.Y - a.contentTop()          // content row inside the panel
	if row < 0 || row > panelH-3 {
		return a, nil // header, tabs, borders, status or fkey bar
	}

	if m.X < sidebarWidth {
		if a.sidebar.SetCursor(a.sidebar.offset + row) {
			a.focus = focusSidebar
			// clicking a context row only selects it — Enter stays the toggle
			return a.applyFilters()
		}
		return a, nil // separator/section header or blank area
	}

	idx := a.list.offset + row
	if idx >= len(a.list.rows) {
		return a, nil // click below the last row must not jump the cursor
	}
	if idx == a.list.cursor && a.focus == focusList {
		// NC touch: clicking the already-selected row opens the detail
		return a, a.openDetailCmd(a.list.rows[idx].t.ID)
	}
	a.list.MoveTo(idx)
	a.focus = focusList
	return a, nil
}
