package tui

import "github.com/jvsaga/taskframe/internal/task"

type tasksLoadedMsg struct{ tasks []*task.Task }

// sidebarData carries everything the sidebar needs in one message.
type sidebarData struct {
	counts  map[string]int // pending per exact project string
	tags    map[string]int // pending per tag
	total   int
	today   int
	overdue int
	week    int
	waiting int
	done    int
	del     int
}

type projectsLoadedMsg struct{ data sidebarData }

type detailLoadedMsg struct {
	t     *task.Task
	notes []task.Note
	acts  []task.Activity
}

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
