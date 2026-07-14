package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvsaga/taskframe/internal/i18n"
	"github.com/jvsaga/taskframe/internal/task"
)

const (
	fTitle = iota
	fProject
	fTags
	fPrio
	fDue
	fWait
	fScheduled
	fRecur
	fCount
)

var formLabelKeys = [fCount]string{"form.title", "form.project", "form.tags", "form.priority", "form.due", "form.waitUntil", "form.scheduled", "form.recur"}
var formHintKeys = [fCount]string{"", "form.hint.project", "form.hint.tags", "form.hint.priority", "form.hint.due", "form.hint.wait", "form.hint.scheduled", "form.hint.recur"}

// Form is the add/edit task modal.
type Form struct {
	lang     i18n.Lang
	inputs   [fCount]textinput.Model
	focus    int
	original *task.Task // nil = creating
	parentID int64
	errText  string
}

func NewForm(lang i18n.Lang, original *task.Task, parentID int64, defaultProject string) *Form {
	f := &Form{lang: lang, original: original, parentID: parentID}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Prompt = ""
		ti.CharLimit = 200
		ti.Width = 40
		ti.Cursor.SetMode(cursor.CursorStatic)
		f.inputs[i] = ti
	}
	if original != nil {
		f.inputs[fTitle].SetValue(original.Title)
		f.inputs[fProject].SetValue(original.Project)
		f.inputs[fTags].SetValue(strings.Join(original.Tags, " "))
		f.inputs[fPrio].SetValue(string(original.Priority))
		if original.Due != nil {
			f.inputs[fDue].SetValue(original.Due.Format("02/01/2006"))
		}
		if original.Wait != nil {
			f.inputs[fWait].SetValue(original.Wait.Format("02/01/2006"))
		}
		if original.Scheduled != nil {
			f.inputs[fScheduled].SetValue(original.Scheduled.Format("02/01/2006"))
		}
		f.inputs[fRecur].SetValue(original.Recur)
	} else {
		f.inputs[fProject].SetValue(defaultProject)
	}
	f.inputs[fTitle].Focus()
	return f
}

func (f *Form) Update(msg tea.Msg) (Modal, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}
	switch keyMsg.String() {
	case "esc":
		return f, func() tea.Msg { return modalCancelMsg{} }
	case "enter":
		return f.submit()
	case "tab", "down":
		f.setFocus((f.focus + 1) % fCount)
	case "shift+tab", "up":
		f.setFocus((f.focus + fCount - 1) % fCount)
	default:
		var cmd tea.Cmd
		f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
		return f, cmd
	}
	return f, nil
}

func (f *Form) setFocus(i int) {
	f.inputs[f.focus].Blur()
	f.focus = i
	f.inputs[f.focus].Focus()
}

func (f *Form) submit() (Modal, tea.Cmd) {
	title := strings.TrimSpace(f.inputs[fTitle].Value())
	if title == "" {
		f.errText = f.lang.T("form.err.titleReq")
		f.setFocus(fTitle)
		return f, nil
	}
	prio := strings.ToUpper(strings.TrimSpace(f.inputs[fPrio].Value()))
	if prio != "" && prio != "H" && prio != "M" && prio != "L" {
		f.errText = f.lang.T("form.err.priority")
		f.setFocus(fPrio)
		return f, nil
	}

	parseDateField := func(idx int, name string) (*time.Time, bool) {
		v := strings.TrimSpace(f.inputs[idx].Value())
		if v == "" {
			return nil, true
		}
		d, err := task.ParseDate(v, time.Now())
		if err != nil {
			f.errText = f.lang.Tf("form.err.fieldInvalid", name, v)
			f.setFocus(idx)
			return nil, false
		}
		return &d, true
	}
	due, ok := parseDateField(fDue, f.lang.T("form.field.due"))
	if !ok {
		return f, nil
	}
	wait, ok := parseDateField(fWait, f.lang.T("form.field.wait"))
	if !ok {
		return f, nil
	}
	scheduled, ok := parseDateField(fScheduled, f.lang.T("form.field.scheduled"))
	if !ok {
		return f, nil
	}

	recur := strings.TrimSpace(f.inputs[fRecur].Value())
	if recur != "" {
		if _, err := task.NextRecurrence(recur, time.Now()); err != nil {
			f.errText = f.lang.Tf("form.err.recur", recur)
			f.setFocus(fRecur)
			return f, nil
		}
	}

	var t task.Task
	if f.original != nil {
		t = *f.original
		t.Children = nil
	} else {
		t.ParentID = f.parentID
	}
	t.Title = title
	t.Project = strings.TrimSpace(f.inputs[fProject].Value())
	t.Tags = strings.Fields(f.inputs[fTags].Value())
	t.Priority = task.Priority(prio)
	t.Due = due
	t.Wait = wait
	t.Scheduled = scheduled
	t.Recur = recur

	edit := f.original != nil
	return f, func() tea.Msg { return formSubmittedMsg{t: t, edit: edit} }
}

func (f *Form) View(th Theme, w, h int) string {
	title := f.lang.T("form.title.new")
	if f.original != nil {
		title = f.lang.T("form.title.edit")
	} else if f.parentID != 0 {
		title = f.lang.T("form.title.newSub")
	}

	labelW := 12
	var lines []string
	lines = append(lines, "")
	for i := 0; i < fCount; i++ {
		label := padRowPlain(f.lang.T(formLabelKeys[i]), labelW)
		style := th.Text
		if i == f.focus {
			style = th.TitleFocus
		}
		row := " " + style.Render(label) + th.Text.Render(f.inputs[i].View())
		lines = append(lines, row)
		if i == f.focus && formHintKeys[i] != "" {
			lines = append(lines, " "+strings.Repeat(" ", labelW)+th.Dim.Render(f.lang.T(formHintKeys[i])))
		} else {
			lines = append(lines, "")
		}
	}
	if f.errText != "" {
		lines = append(lines, " "+th.Overdue.Render("x "+f.errText))
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, " "+th.Dim.Render(f.lang.T("form.footer")))

	bw := 62
	if bw > w-4 {
		bw = w - 4
	}
	return drawBox(th, title, lines, bw, len(lines)+3, true)
}
