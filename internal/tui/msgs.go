package tui

import (
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
)

type tasksLoadedMsg struct{ tasks []*task.Task }

// ctxEntry is one defined context and how many tasks its filter matches.
type ctxEntry struct {
	name  string
	count int
}

// sidebarData carries everything the sidebar (and the tab band) needs in one
// message. The per-report counts died with the sidebar's filters section;
// overdue survives because the Overdue tab label lights up on it.
type sidebarData struct {
	counts    map[string]store.ProjectCount // pending/done per exact project string
	tags      map[string]int                // pending per tag
	total     int
	overdue   int
	done      int
	del       int
	contexts  []ctxEntry // sorted by name
	activeCtx string     // active context name ("" = none)
}

type projectsLoadedMsg struct{ data sidebarData }

type detailLoadedMsg struct {
	t        *task.Task
	children []*task.Task
	notes    []task.Note
	acts     []task.Activity
}

type readLoadedMsg struct {
	t     *task.Task
	notes []task.Note
}

// frameMsg drives one step of a spring animation (see scroller).
type frameMsg struct{}

type errMsg struct{ err error }

type statusMsg string

// formSubmittedMsg carries the finished form value; app persists it.
type formSubmittedMsg struct {
	t    task.Task
	edit bool
}

type modalCancelMsg struct{}

type confirmResultMsg struct{ ok bool }

type noteSubmittedMsg struct {
	taskID int64
	body   string
}

type moveSubmittedMsg struct {
	taskID   int64
	project  string
	parentID int64 // 0 = promote to root
}
