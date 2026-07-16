package tui

import "github.com/mustachius/taskframe/internal/task"

type tasksLoadedMsg struct{ tasks []*task.Task }

// ctxEntry is one defined context and how many tasks its filter matches.
type ctxEntry struct {
	name  string
	count int
}

// sidebarData carries everything the sidebar needs in one message.
type sidebarData struct {
	counts    map[string]int // pending per exact project string
	tags      map[string]int // pending per tag
	total     int
	today     int
	overdue   int
	week      int
	active    int
	next      int
	waiting   int
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
