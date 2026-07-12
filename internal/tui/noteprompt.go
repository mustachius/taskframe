package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// NotePrompt is a one-line input for adding a note to a task.
type NotePrompt struct {
	taskID    int64
	taskTitle string
	input     textinput.Model
}

func NewNotePrompt(taskID int64, taskTitle string) *NotePrompt {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 500
	ti.Width = 50
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.Focus()
	return &NotePrompt{taskID: taskID, taskTitle: taskTitle, input: ti}
}

func (n *NotePrompt) Update(msg tea.Msg) (Modal, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return n, nil
	}
	switch keyMsg.String() {
	case "esc":
		return n, func() tea.Msg { return modalCancelMsg{} }
	case "enter":
		body := strings.TrimSpace(n.input.Value())
		if body == "" {
			return n, func() tea.Msg { return modalCancelMsg{} }
		}
		id := n.taskID
		return n, func() tea.Msg { return noteSubmittedMsg{taskID: id, body: body} }
	}
	var cmd tea.Cmd
	n.input, cmd = n.input.Update(msg)
	return n, cmd
}

func (n *NotePrompt) View(th Theme, w, h int) string {
	lines := []string{
		"",
		" " + th.Dim.Render(truncRunes(n.taskTitle, 50)),
		" " + th.Text.Render(n.input.View()),
		"",
		" " + th.Dim.Render("Enter salva · Esc cancela"),
	}
	bw := 58
	if bw > w-4 {
		bw = w - 4
	}
	return drawBox(th, "Nova nota", lines, bw, len(lines)+3, true)
}
