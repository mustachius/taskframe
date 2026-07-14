package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/task"
)

// Detail shows a task's fields, notes and full activity log.
type Detail struct {
	lang   i18n.Lang
	t      *task.Task
	notes  []task.Note
	acts   []task.Activity
	scroll int
}

func NewDetail(lang i18n.Lang, t *task.Task, notes []task.Note, acts []task.Activity) *Detail {
	return &Detail{lang: lang, t: t, notes: notes, acts: acts}
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
	add(" " + label(d.lang.T("lbl.status")) + val(string(t.Status)))
	if t.Project != "" {
		add(" " + label(d.lang.T("lbl.project")) + val(t.Project))
	}
	if len(t.Tags) > 0 {
		add(" " + label(d.lang.T("lbl.tags")) + val("+"+strings.Join(t.Tags, " +")))
	}
	if t.Priority != task.PriorityNone {
		add(" " + label(d.lang.T("lbl.priority")) + val(string(t.Priority)))
	}
	if t.Due != nil {
		add(" " + label(d.lang.T("lbl.due")) + val(t.Due.Format("02/01/2006")))
	}
	if t.Wait != nil {
		add(" " + label(d.lang.T("lbl.waitUntil")) + val(t.Wait.Format("02/01/2006")))
	}
	if t.Scheduled != nil {
		add(" " + label(d.lang.T("lbl.scheduled")) + val(t.Scheduled.Format("02/01/2006")))
	}
	if t.Recur != "" {
		add(" " + label(d.lang.T("lbl.recurrence")) + val(t.Recur))
	}
	add(" " + label(d.lang.T("lbl.created")) + val(t.CreatedAt.Format("02/01/2006 15:04")))
	if t.CompletedAt != nil {
		add(" " + label(d.lang.T("lbl.completed")) + val(t.CompletedAt.Format("02/01/2006 15:04")))
	}

	if len(d.notes) > 0 {
		add("")
		add(" " + th.TitleFocus.Render(d.lang.T("detail.notes")))
		for _, n := range d.notes {
			add(" " + th.Dim.Render(n.CreatedAt.Format("02/01 15:04")+" ") +
				th.Text.Render(truncRunes(n.Body, w-20)))
		}
	}

	add("")
	add(" " + th.TitleFocus.Render(d.lang.T("detail.history")))
	for _, a := range d.acts {
		add(" " + th.Dim.Render(a.TS.Format("02/01 15:04")+" ") + th.Text.Render(truncRunes(actDesc(d.lang, a), w-20)))
	}
	add("")
	add(" " + th.Dim.Render(d.lang.T("detail.footerTui")))

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
	return drawBox(th, d.lang.Tf("detail.titleTui", t.ID), visible, bw, bh, true)
}

func actDesc(lang i18n.Lang, a task.Activity) string {
	switch a.Kind {
	case "create":
		return lang.T("act.created") + a.NewVal
	case "done":
		return lang.T("act.done")
	case "delete":
		return lang.T("act.deleted")
	case "note":
		return lang.T("act.note") + a.NewVal
	case "modify":
		if a.OldVal == "" {
			return lang.Tf("act.setTo", a.Field, a.NewVal)
		}
		return lang.Tf("act.changed", a.Field, a.OldVal, a.NewVal)
	}
	return a.Kind
}
