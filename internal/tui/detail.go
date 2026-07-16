package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
)

// Detail shows a task's fields, subtasks, notes and full activity log.
type Detail struct {
	lang     i18n.Lang
	ascii    bool
	t        *task.Task
	children []*task.Task
	notes    []task.Note
	acts     []task.Activity
	sc       scroller
}

func NewDetail(lang i18n.Lang, ascii, reduceMotion bool, t *task.Task, children []*task.Task, notes []task.Note, acts []task.Activity) *Detail {
	return &Detail{lang: lang, ascii: ascii, t: t, children: children, notes: notes, acts: acts, sc: newScroller(reduceMotion)}
}

func (d *Detail) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "esc", "enter", "f3", "q":
			return d, func() tea.Msg { return modalCancelMsg{} }
		}
		return d, d.sc.onKey(m)
	case frameMsg:
		return d, d.sc.onFrame()
	}
	return d, nil
}

// scrollBy lets the mouse wheel scroll the detail (see handleMouse).
func (d *Detail) scrollBy(delta int) tea.Cmd { return d.sc.scrollBy(delta) }

func (d *Detail) View(th Theme, w, h int) string {
	t := d.t
	label := func(s string) string { return th.Dim.Render(padRowPlain(s, 16)) }
	val := func(s string) string { return th.Text.Render(s) }

	var lines []string
	add := func(s string) { lines = append(lines, s) }

	now := time.Now()
	add("")
	add(" " + th.TitleFocus.Render(truncRunes(t.Title, w-8)))
	add("")
	statusStyle := th.Text
	if t.Status != task.StatusPending {
		statusStyle = th.Dim
	}
	add(" " + label(d.lang.T("lbl.status")) + statusStyle.Render(string(t.Status)))
	if t.Project != "" {
		add(" " + label(d.lang.T("lbl.project")) + th.Accent.Render(t.Project))
	}
	if len(t.Tags) > 0 {
		add(" " + label(d.lang.T("lbl.tags")) + th.Accent.Render("+"+strings.Join(t.Tags, " +")))
	}
	if t.Priority != task.PriorityNone {
		priStyle := th.Text
		switch t.Priority {
		case task.PriorityHigh:
			priStyle = th.PrioHi
		case task.PriorityMed:
			priStyle = th.Accent
		}
		add(" " + label(d.lang.T("lbl.priority")) + priStyle.Render(string(t.Priority)))
	}
	if t.Due != nil {
		dueStyle := th.Text
		switch {
		case t.IsOverdue(now):
			dueStyle = th.Overdue
		case !t.Due.After(task.EndOfDay(now)):
			dueStyle = th.Warn
		}
		add(" " + label(d.lang.T("lbl.due")) + dueStyle.Render(t.Due.Format("02/01/2006")))
	}
	if t.Wait != nil {
		add(" " + label(d.lang.T("lbl.waitUntil")) + val(t.Wait.Format("02/01/2006")))
	}
	if t.Scheduled != nil {
		add(" " + label(d.lang.T("lbl.scheduled")) + val(t.Scheduled.Format("02/01/2006")))
	}
	if t.Recur != "" {
		add(" " + label(d.lang.T("lbl.recurrence")) + th.Dim.Render(t.Recur))
	}
	if t.Start != nil {
		add(" " + label(d.lang.T("lbl.started")) +
			th.Accent.Render(t.Start.Format("02/01/2006 15:04")+" · "+ui.FormatElapsed(now.Sub(*t.Start))))
	}
	add(" " + label(d.lang.T("lbl.created")) + val(t.CreatedAt.Format("02/01/2006 15:04")))
	if t.CompletedAt != nil {
		add(" " + label(d.lang.T("lbl.completed")) + val(t.CompletedAt.Format("02/01/2006 15:04")))
	}

	if len(d.children) > 0 {
		done := 0
		for _, c := range d.children {
			if c.Status == task.StatusDone {
				done++
			}
		}
		add("")
		bar := progressBar(float64(done)/float64(len(d.children)), 12, th)
		add(" " + th.TitleFocus.Render(d.lang.Tf("detail.subtasks", done, len(d.children))) + "  " + bar)
		for _, c := range d.children {
			mark := "[ ]"
			if c.Status == task.StatusDone {
				mark = "[x]"
			}
			add(" " + th.Dim.Render(mark+" ") + th.Text.Render(truncRunes(c.Title, w-20)))
		}
	}

	if len(d.notes) > 0 {
		add("")
		add(" " + th.TitleFocus.Render(d.lang.T("detail.notes")))
		var nb strings.Builder
		for _, n := range d.notes {
			nb.WriteString("**" + n.CreatedAt.Format("02/01 15:04") + "** — " + n.Body + "\n\n")
		}
		mw := w - 10
		if mw > 74 {
			mw = 74
		}
		if mw < 10 {
			mw = 10
		}
		md := renderMarkdown(nb.String(), mw, mdStyle(th, d.ascii))
		for _, ln := range strings.Split(strings.TrimRight(md, "\n"), "\n") {
			add(ln)
		}
	}

	add("")
	add(" " + th.TitleFocus.Render(d.lang.T("detail.history")))
	for _, a := range d.acts {
		add(" " + th.Dim.Render(a.TS.Format("02/01 15:04")+" ") + th.Text.Render(truncRunes(actDesc(d.lang, a), w-20)))
	}

	bw := w - 8
	if bw > 76 {
		bw = 76
	}
	bh := h - 4
	d.sc.setSize(bw-2, bh-2)
	d.sc.setContent(strings.Join(lines, "\n"))
	visible := strings.Split(d.sc.view(), "\n")
	box := drawBox(th, d.lang.Tf("detail.titleTui", t.ID), visible, bw, bh, true)
	return box + "\n " + th.Dim.Render(d.lang.T("detail.footerTui"))
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
		return lang.T("act.note")
	case "modify":
		if a.OldVal == "" {
			return lang.Tf("act.setTo", a.Field, a.NewVal)
		}
		return lang.Tf("act.changed", a.Field, a.OldVal, a.NewVal)
	}
	return a.Kind
}
