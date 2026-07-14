package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
)

// Move is the F6 dialog: change a task's project and/or parent.
type Move struct {
	lang      i18n.Lang
	taskID    int64
	taskTitle string
	project   textinput.Model
	parent    textinput.Model
	focus     int
	errText   string
}

func NewMove(lang i18n.Lang, taskID int64, taskTitle, project string, parentID int64) *Move {
	mk := func(v string) textinput.Model {
		ti := textinput.New()
		ti.Prompt = ""
		ti.CharLimit = 100
		ti.Width = 30
		ti.Cursor.SetMode(cursor.CursorStatic)
		ti.SetValue(v)
		return ti
	}
	parentStr := ""
	if parentID != 0 {
		parentStr = strconv.FormatInt(parentID, 10)
	}
	m := &Move{lang: lang, taskID: taskID, taskTitle: taskTitle, project: mk(project), parent: mk(parentStr)}
	m.project.Focus()
	return m
}

func (m *Move) Update(msg tea.Msg) (Modal, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		return m, func() tea.Msg { return modalCancelMsg{} }
	case "tab", "down", "shift+tab", "up":
		if m.focus == 0 {
			m.focus = 1
			m.project.Blur()
			m.parent.Focus()
		} else {
			m.focus = 0
			m.parent.Blur()
			m.project.Focus()
		}
		return m, nil
	case "enter":
		var parentID int64
		if v := strings.TrimSpace(m.parent.Value()); v != "" {
			id, err := strconv.ParseInt(v, 10, 64)
			if err != nil || id <= 0 {
				m.errText = m.lang.Tf("move.err.parentInvalid", v)
				return m, nil
			}
			if id == m.taskID {
				m.errText = m.lang.T("move.err.selfParent")
				return m, nil
			}
			parentID = id
		}
		taskID := m.taskID
		project := strings.TrimSpace(m.project.Value())
		return m, func() tea.Msg {
			return moveSubmittedMsg{taskID: taskID, project: project, parentID: parentID}
		}
	}
	var cmd tea.Cmd
	if m.focus == 0 {
		m.project, cmd = m.project.Update(msg)
	} else {
		m.parent, cmd = m.parent.Update(msg)
	}
	return m, cmd
}

func (m *Move) View(th Theme, w, h int) string {
	field := func(idx int, label string, ti textinput.Model) string {
		style := th.Text
		if m.focus == idx {
			style = th.TitleFocus
		}
		return " " + style.Render(padRowPlain(label, 12)) + th.Text.Render(ti.View())
	}
	lines := []string{
		"",
		" " + th.Dim.Render(truncRunes(fmt.Sprintf("%d — %s", m.taskID, m.taskTitle), 44)),
		"",
		field(0, m.lang.T("move.field.project"), m.project),
		field(1, m.lang.T("move.field.parent"), m.parent),
		" " + strings.Repeat(" ", 12) + th.Dim.Render(m.lang.T("move.hint.root")),
		"",
	}
	if m.errText != "" {
		lines = append(lines, " "+th.Overdue.Render("x "+m.errText))
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, " "+th.Dim.Render(m.lang.T("move.footer")))

	bw := 52
	if bw > w-4 {
		bw = w - 4
	}
	return drawBox(th, m.lang.T("move.title"), lines, bw, len(lines)+3, true)
}
