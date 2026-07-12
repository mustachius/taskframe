// Package task contains the pure domain model for taskframe.
// It imports nothing from the rest of the project.
package task

import (
	"strings"
	"time"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusDone    Status = "done"
	StatusDeleted Status = "deleted"
)

type Priority string

const (
	PriorityNone Priority = ""
	PriorityLow  Priority = "L"
	PriorityMed  Priority = "M"
	PriorityHigh Priority = "H"
)

type Task struct {
	ID          int64
	ParentID    int64 // 0 = top-level
	Title       string
	Project     string // dotted path: "work.api.auth"
	Priority    Priority
	Status      Status
	Tags        []string
	Due         *time.Time
	Wait        *time.Time
	Scheduled   *time.Time
	Recur       string // "", "daily", "weekly", "monthly", "3d", ...
	CreatedAt   time.Time
	ModifiedAt  time.Time
	CompletedAt *time.Time

	// Children is populated when loading the task tree.
	Children []*Task
}

func (t *Task) HasTag(tag string) bool {
	for _, tg := range t.Tags {
		if tg == tag {
			return true
		}
	}
	return false
}

// IsWaiting reports whether the task is hidden until a future wait date.
func (t *Task) IsWaiting(now time.Time) bool {
	return t.Wait != nil && t.Wait.After(now)
}

func (t *Task) IsOverdue(now time.Time) bool {
	return t.Due != nil && t.Status == StatusPending && t.Due.Before(now)
}

// ProjectParts splits a dotted project path into its segments.
func ProjectParts(project string) []string {
	if project == "" {
		return nil
	}
	return strings.Split(project, ".")
}

type Note struct {
	ID        int64
	TaskID    int64
	Body      string
	CreatedAt time.Time
}

type Activity struct {
	ID     int64
	OpID   string
	TaskID int64
	TS     time.Time
	Kind   string // create, modify, done, delete, note, undo
	Field  string
	OldVal string
	NewVal string
}
