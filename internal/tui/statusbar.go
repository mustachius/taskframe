package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderStatusChips draws the status row, lipgloss-README style: a primary
// chip with the task count, the transient status message beside it, and
// right-aligned chips for the active context, language and sort order.
func (a *App) renderStatusChips() string {
	th := a.th
	if a.searching {
		return padRow(th.Status.Render(" "+a.search.View()), a.w, th.Status)
	}

	count := " " + strings.TrimSpace(a.lang.Tf("app.taskCount", a.list.Count())) + " "

	msg := ""
	if a.filter.Text != "" {
		msg = a.lang.Tf("app.searchInfo", a.filter.Text)
	}
	if a.status != "" {
		msg = " " + a.status + msg
	}
	msgStyle := th.Status
	if a.statusErr {
		msgStyle = th.StatusErr
	}

	type chip struct {
		text  string
		style lipgloss.Style
	}
	var rights []chip
	if a.activeCtx != "" {
		rights = append(rights, chip{" @" + a.activeCtx + " ", th.Chip})
	}
	rights = append(rights,
		chip{" " + langTag(a.lang) + " ", th.ChipAlt},
		chip{" " + a.lang.T("sort."+string(a.sortMode)) + " ", th.ChipAlt},
	)
	rightW := 0
	for _, c := range rights {
		rightW += len([]rune(c.text)) + 1 // 1-cell breathing gap per chip
	}

	// widths measured on the plain strings; the message yields when tight
	avail := a.w - len([]rune(count)) - rightW
	if avail < 0 {
		avail = 0
	}
	msg = truncRunes(msg, avail)
	gap := a.w - len([]rune(count)) - len([]rune(msg)) - rightW
	if gap < 0 {
		gap = 0
	}

	var b strings.Builder
	b.WriteString(th.Chip.Render(count))
	b.WriteString(msgStyle.Render(msg))
	b.WriteString(th.Status.Render(strings.Repeat(" ", gap)))
	for _, c := range rights {
		b.WriteString(th.Status.Render(" "))
		b.WriteString(c.style.Render(c.text))
	}
	return b.String()
}
