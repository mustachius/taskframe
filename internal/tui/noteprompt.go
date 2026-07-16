package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mustachius/taskframe/internal/i18n"
)

// NotePrompt is a multi-line input for adding a note to a task: enter breaks
// the line (notes render as markdown), ctrl+d saves.
type NotePrompt struct {
	lang      i18n.Lang
	taskID    int64
	taskTitle string
	input     textarea.Model
}

func NewNotePrompt(lang i18n.Lang, taskID int64, taskTitle string) *NotePrompt {
	ta := textarea.New()
	ta.Prompt = "" // the box border already frames the text
	ta.ShowLineNumbers = false
	ta.CharLimit = 500
	ta.SetWidth(50)
	ta.SetHeight(4)
	ta.Cursor.SetMode(cursor.CursorStatic)
	// the default cursor-line background clashes with bg-painting themes
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.Focus()
	return &NotePrompt{lang: lang, taskID: taskID, taskTitle: taskTitle, input: ta}
}

func (n *NotePrompt) Update(msg tea.Msg) (Modal, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return n, nil
	}
	switch keyMsg.String() {
	case "esc":
		return n, func() tea.Msg { return modalCancelMsg{} }
	// ctrl+d saves. It MUST be intercepted here, before input.Update —
	// textarea's default keymap binds ctrl+d to delete-character-forward.
	case "ctrl+d":
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
	}
	for _, l := range strings.Split(n.input.View(), "\n") {
		lines = append(lines, " "+l)
	}
	lines = append(lines,
		"",
		" "+th.Dim.Render(n.lang.T("notePrompt.footer")),
	)
	bw := 58
	if bw > w-4 {
		bw = w - 4
	}
	return drawBox(th, n.lang.T("notePrompt.title"), lines, bw, len(lines)+3, true)
}
