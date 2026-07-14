package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
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
	vp    viewport.Model
}

func NewRead(lang i18n.Lang, ascii bool, t *task.Task, notes []task.Note) *Read {
	return &Read{lang: lang, ascii: ascii, t: t, notes: notes, vp: viewport.New(0, 0)}
}

func (r *Read) Update(msg tea.Msg) (Modal, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return r, nil
	}
	switch keyMsg.String() {
	case "esc", "q", "R":
		return r, func() tea.Msg { return modalCancelMsg{} }
	}
	var cmd tea.Cmd
	r.vp, cmd = r.vp.Update(msg)
	return r, cmd
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
	rendered := strings.TrimRight(renderMarkdown(r.markdown(), bw-4, r.ascii), "\n")
	r.vp.Width = bw - 2
	r.vp.Height = bh - 2
	r.vp.SetContent(rendered)
	visible := strings.Split(r.vp.View(), "\n")
	box := drawBox(th, r.lang.Tf("detail.titleTui", r.t.ID), visible, bw, bh, true)
	return box + "\n " + th.Dim.Render(r.lang.T("detail.footerTui"))
}
