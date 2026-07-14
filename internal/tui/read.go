package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/task"
)

// Read is a full-screen, Glow-style view of a task's notes rendered as Markdown.
type Read struct {
	lang  i18n.Lang
	ascii bool
	t     *task.Task
	notes []task.Note
	sc    scroller
}

func NewRead(lang i18n.Lang, ascii, reduceMotion bool, t *task.Task, notes []task.Note) *Read {
	return &Read{lang: lang, ascii: ascii, t: t, notes: notes, sc: newScroller(reduceMotion)}
}

func (r *Read) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "esc", "q", "R":
			return r, func() tea.Msg { return modalCancelMsg{} }
		}
		return r, r.sc.onKey(m)
	case frameMsg:
		return r, r.sc.onFrame()
	}
	return r, nil
}

// markdown assembles the task as a Markdown document: the title as an H1 and
// each note as a timestamped section.
func (r *Read) markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", r.t.Title)
	if len(r.notes) == 0 {
		b.WriteString(r.lang.T("read.noNotes") + "\n")
		return b.String()
	}
	for _, n := range r.notes {
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", n.CreatedAt.Format("02/01/2006 15:04"), n.Body)
	}
	return b.String()
}

func (r *Read) View(th Theme, w, h int) string {
	bw := w - 8
	if bw > 84 {
		bw = 84
	}
	bh := h - 4
	rendered := strings.TrimRight(renderMarkdown(r.markdown(), bw-4, mdStyle(th, r.ascii)), "\n")
	r.sc.setSize(bw-2, bh-2)
	r.sc.setContent(rendered)
	visible := strings.Split(r.sc.view(), "\n")
	box := drawBox(th, r.lang.Tf("detail.titleTui", r.t.ID), visible, bw, bh, true)
	return box + "\n " + th.Dim.Render(r.lang.T("detail.footerTui"))
}
