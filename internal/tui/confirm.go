package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Confirm is a yes/no dialog.
type Confirm struct {
	title   string
	message string
}

func NewConfirm(title, message string) *Confirm {
	return &Confirm{title: title, message: message}
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
		" " + th.TitleFocus.Render("[S]im") + th.Text.Render("   ") + th.Dim.Render("[N]ão (Esc)"),
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
