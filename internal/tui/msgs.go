package tui

import "github.com/jvsaga/taskframe/internal/task"

type tasksLoadedMsg struct{ tasks []*task.Task }

type projectsLoadedMsg struct {
	counts map[string]int
	total  int
	done   int
	del    int
}

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
