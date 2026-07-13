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
	Start       *time.Time // set while the task is active (started); nil = idle

	// Children is populated when loading the task tree; never serialized.
	Children []*Task `json:"-"`
}

// IsActive reports whether the task is currently started.
func (t *Task) IsActive() bool { return t.Start != nil }

func (t *Task) HasTag(tag string) bool {
	for _, tg := range t.Tags {
		if tg == tag {
			return true
		}
	}
	return false
}

// MergeTags returns existing tags plus add, minus remove — deduplicated and in
// first-seen order. Used by edit so +tag/-tag amend the set instead of
// overwriting it.
func MergeTags(existing, add, remove []string) []string {
	rm := make(map[string]bool, len(remove))
	for _, t := range remove {
		rm[t] = true
	}
	seen := make(map[string]bool, len(existing)+len(add))
	out := make([]string, 0, len(existing)+len(add))
	for _, t := range existing {
		if !rm[t] && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, t := range add {
		if !rm[t] && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
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
