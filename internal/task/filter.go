package task

import "time"

// Filter describes which tasks to list. Zero value = all pending tasks.
type Filter struct {
	Project     string   // prefix match on dotted path ("work" matches "work.api")
	Tags        []string // required tags (AND)
	ExcludeTags []string // tags that must be absent (-tag)
	Status      Status   // "" = pending only; "all" is expressed via IncludeAll
	IncludeAll  bool     // include done and deleted
	DueBefore   *time.Time
	Text        string // substring match on title and notes
	HideWaiting bool   // hide tasks with wait date in the future
	WaitingOnly bool   // only tasks with wait date in the future (overrides HideWaiting)
	ActiveOnly  bool   // only started tasks (start IS NOT NULL)
	NoContext   bool   // read-time directive: ignore the active context
}

// Merge overlays other onto f: scalar fields from other win when set, slice and
// boolean fields are unioned. Used to combine a report/context filter (f) with
// the extra tokens a user typed (other).
func (f Filter) Merge(other Filter) Filter {
	if other.Project != "" {
		f.Project = other.Project
	}
	if other.Status != "" {
		f.Status = other.Status
	}
	if other.DueBefore != nil {
		f.DueBefore = other.DueBefore
	}
	if other.Text != "" {
		f.Text = other.Text
	}
	f.Tags = append(f.Tags, other.Tags...)
	f.ExcludeTags = append(f.ExcludeTags, other.ExcludeTags...)
	f.IncludeAll = f.IncludeAll || other.IncludeAll
	f.HideWaiting = f.HideWaiting || other.HideWaiting
	f.WaitingOnly = f.WaitingOnly || other.WaitingOnly
	f.ActiveOnly = f.ActiveOnly || other.ActiveOnly
	f.NoContext = f.NoContext || other.NoContext
	return f
}
