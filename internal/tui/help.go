package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
)

// Help is the F1 keybinding reference.
type Help struct{ lang i18n.Lang }

func (hp *Help) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return hp, func() tea.Msg { return modalCancelMsg{} }
	}
	return hp, nil
}

func (hp *Help) View(th Theme, w, h int) string {
	// letters lead each combo: the F-keys still work but left the visual
	rows := [][2]string{
		{"Tab", hp.lang.T("tuihelp.tab.v")},
		{"↑↓ / jk", hp.lang.T("tuihelp.move.v")},
		{"←→ / hl", hp.lang.T("tuihelp.fold.v")},
		{"[ ] · 1-7", hp.lang.T("tuihelp.tabs.v")},
		{"Enter, F3", hp.lang.T("tuihelp.detail.v")},
		{"R", hp.lang.T("tuihelp.read.v")},
		{"a, F2", hp.lang.T("tuihelp.new.v")},
		{"s", hp.lang.T("tuihelp.newsub.v")},
		{"e, F4", hp.lang.T("tuihelp.edit.v")},
		{"n, F5", hp.lang.T("tuihelp.note.v")},
		{"m, F6", hp.lang.T("tuihelp.movedlg.v")},
		{"d, space, F9", hp.lang.T("tuihelp.done.v")},
		{"x, Del, F8", hp.lang.T("tuihelp.del.v")},
		{"/, F7", hp.lang.T("tuihelp.search.v")},
		{"S", hp.lang.T("tuihelp.start.v")},
		{"Enter (@ctx)", hp.lang.T("tuihelp.ctx.v")},
		{"o", hp.lang.T("tuihelp.sort.v")},
		{"t", hp.lang.T("tuihelp.theme.v")},
		{"L", hp.lang.T("tuihelp.lang.v")},
		{"u", hp.lang.T("tuihelp.undo.v")},
		{"U", hp.lang.T("tuihelp.redo.v")},
		{"r", hp.lang.T("tuihelp.reload.v")},
		{"q, F10", hp.lang.T("tuihelp.quit.v")},
	}
	var lines []string
	lines = append(lines, "")
	bw := 64
	if w >= 80 {
		// two columns: the header + tab band shrank the modal canvas, and
		// single-column would overflow it (the overlay clips overflow). The
		// description widths derive from the final box width — DrawBox pads
		// short lines but never truncates long ones.
		bw = 94
		if bw > w-4 {
			bw = w - 4
		}
		avail := bw - 2 - 1 - 14 - 2 - 14 // inner minus lead pad, labels, gutter
		dw1 := avail / 2
		dw2 := avail - dw1
		half := (len(rows) + 1) / 2
		for i := 0; i < half; i++ {
			line := " " + th.TitleFocus.Render(padRowPlain(rows[i][0], 14)) +
				th.Text.Render(padRowPlain(truncRunes(rows[i][1], dw1), dw1))
			if j := i + half; j < len(rows) {
				line += "  " + th.TitleFocus.Render(padRowPlain(rows[j][0], 14)) +
					th.Text.Render(truncRunes(rows[j][1], dw2))
			}
			lines = append(lines, line)
		}
	} else {
		for _, r := range rows {
			lines = append(lines, " "+th.TitleFocus.Render(padRowPlain(r[0], 16))+th.Text.Render(r[1]))
		}
	}
	lines = append(lines, "", " "+th.Dim.Render(hp.lang.T("tuihelp.footer")))

	if bw > w-4 {
		bw = w - 4
	}
	return drawBox(th, hp.lang.T("tuihelp.title"), lines, bw, len(lines)+3, true)
}
