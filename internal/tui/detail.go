package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvsaga/taskframe/internal/task"
)

// Detail shows a task's fields, notes and full activity log.
type Detail struct {
	t      *task.Task
	notes  []task.Note
	acts   []task.Activity
	scroll int
}

func NewDetail(t *task.Task, notes []task.Note, acts []task.Activity) *Detail {
	return &Detail{t: t, notes: notes, acts: acts}
}

func (d *Detail) Update(msg tea.Msg) (Modal, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return d, nil
	}
	switch keyMsg.String() {
	case "esc", "enter", "f3", "q":
		return d, func() tea.Msg { return modalCancelMsg{} }
	case "up", "k":
		if d.scroll > 0 {
			d.scroll--
		}
	case "down", "j":
		d.scroll++
	}
	return d, nil
}

func (d *Detail) View(th Theme, w, h int) string {
	t := d.t
	label := func(s string) string { return th.Dim.Render(padRowPlain(s, 16)) }
	val := func(s string) string { return th.Text.Render(s) }

	var lines []string
	add := func(s string) { lines = append(lines, s) }

	add("")
	add(" " + th.TitleFocus.Render(truncRunes(t.Title, w-8)))
	add("")
	add(" " + label("Status") + val(string(t.Status)))
	if t.Project != "" {
		add(" " + label("Projeto") + val(t.Project))
	}
	if len(t.Tags) > 0 {
		add(" " + label("Tags") + val("+"+strings.Join(t.Tags, " +")))
	}
	if t.Priority != task.PriorityNone {
		add(" " + label("Prioridade") + val(string(t.Priority)))
	}
	if t.Due != nil {
		add(" " + label("Vencimento") + val(t.Due.Format("02/01/2006")))
	}
	if t.Wait != nil {
		add(" " + label("Aguardando até") + val(t.Wait.Format("02/01/2006")))
	}
	if t.Scheduled != nil {
		add(" " + label("Agendada") + val(t.Scheduled.Format("02/01/2006")))
	}
	if t.Recur != "" {
		add(" " + label("Recorrência") + val(t.Recur))
	}
	add(" " + label("Criada em") + val(t.CreatedAt.Format("02/01/2006 15:04")))
	if t.CompletedAt != nil {
		add(" " + label("Concluída em") + val(t.CompletedAt.Format("02/01/2006 15:04")))
	}

	if len(d.notes) > 0 {
		add("")
		add(" " + th.TitleFocus.Render("Notas"))
		for _, n := range d.notes {
			add(" " + th.Dim.Render(n.CreatedAt.Format("02/01 15:04")+" ") +
				th.Text.Render(truncRunes(n.Body, w-20)))
		}
	}

	add("")
	add(" " + th.TitleFocus.Render("Histórico"))
	for _, a := range d.acts {
		add(" " + th.Dim.Render(a.TS.Format("02/01 15:04")+" ") + th.Text.Render(truncRunes(actDesc(a), w-20)))
	}
	add("")
	add(" " + th.Dim.Render("↑↓ rola · Esc fecha"))

	bw := w - 8
	if bw > 76 {
		bw = 76
	}
	bh := h - 4
	inner := bh - 2
	if d.scroll > len(lines)-inner {
		d.scroll = len(lines) - inner
	}
	if d.scroll < 0 {
		d.scroll = 0
	}
	visible := lines[d.scroll:]
	return drawBox(th, fmt.Sprintf("Tarefa %d", t.ID), visible, bw, bh, true)
}

func actDesc(a task.Activity) string {
	switch a.Kind {
	case "create":
		return "criada: " + a.NewVal
	case "done":
		return "concluída"
	case "delete":
		return "deletada"
	case "note":
		return "nota: " + a.NewVal
	case "modify":
		if a.OldVal == "" {
			return fmt.Sprintf("%s definido: %s", a.Field, a.NewVal)
		}
		return fmt.Sprintf("%s: %s → %s", a.Field, a.OldVal, a.NewVal)
	}
	return a.Kind
}
