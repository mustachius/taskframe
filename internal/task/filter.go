package task

import "time"

// Filter describes which tasks to list. Zero value = all pending tasks.
type Filter struct {
	Project     string // prefix match on dotted path ("work" matches "work.api")
	Tags        []string
	Status      Status // "" = pending only; "all" is expressed via IncludeAll
	IncludeAll  bool   // include done and deleted
	DueBefore   *time.Time
	Text        string // substring match on title and notes
	HideWaiting bool   // hide tasks with wait date in the future
}
