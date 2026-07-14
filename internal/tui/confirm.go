package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvsaga/taskframe/internal/i18n"
)

// Confirm is a yes/no dialog.
type Confirm struct {
	lang    i18n.Lang
	title   string
	message string
}

func NewConfirm(lang i18n.Lang, title, message string) *Confirm {
	return &Confirm{lang: lang, title: title, message: message}
}

func (c *Confirm) Update(msg tea.Msg) (Modal, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	switch keyMsg.String() {
	case "y", "s", "enter":
		return c, func() tea.Msg { return confirmResultMsg{ok: true} }
	case "n", "esc":
		return c, func() tea.Msg { return confirmResultMsg{ok: false} }
	}
	return c, nil
}

func (c *Confirm) View(th Theme, w, h int) string {
	lines := []string{
		"",
		" " + th.Text.Render(c.message),
		"",
		" " + th.TitleFocus.Render(c.lang.T("confirm.yes")) + th.Text.Render("   ") + th.Dim.Render(c.lang.T("confirm.no")),
	}
	bw := len([]rune(c.message)) + 6
	if bw < 30 {
		bw = 30
	}
	if bw > w-4 {
		bw = w - 4
	}
	return drawBox(th, c.title, lines, bw, len(lines)+3, true)
}
