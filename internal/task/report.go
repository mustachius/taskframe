package task

import (
	"sort"
	"strings"
	"time"
)

// Report is a named, preconfigured view (filter + sort + optional row limit),
// à la Taskwarrior reports. It is the single source of truth for the virtual
// filters, shared by the CLI, the REPL and the TUI sidebar.
type Report struct {
	Name        string
	Description string
	Build       func(now time.Time) Filter
	Sort        SortMode
	Limit       int // 0 = no limit
}

// EndOfDay returns the last second of t's calendar day.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
}

var reports = map[string]Report{
	// Descriptions are the canonical English text; UI layers localize via the
	// i18n catalog key "report.<name>".
	"next": {
		Name: "next", Description: "most urgent pending",
		Build: func(now time.Time) Filter { return Filter{HideWaiting: true} },
		Sort:  SortUrgency, Limit: 15,
	},
	"overdue": {
		Name: "overdue", Description: "overdue",
		Build: func(now time.Time) Filter { n := now; return Filter{DueBefore: &n, HideWaiting: true} },
		Sort:  SortDue,
	},
	"today": {
		Name: "today", Description: "due today",
		Build: func(now time.Time) Filter { d := EndOfDay(now); return Filter{DueBefore: &d, HideWaiting: true} },
		Sort:  SortDue,
	},
	"week": {
		Name: "week", Description: "next 7 days",
		Build: func(now time.Time) Filter {
			d := EndOfDay(now.AddDate(0, 0, 7))
			return Filter{DueBefore: &d, HideWaiting: true}
		},
		Sort: SortDue,
	},
	"waiting": {
		Name: "waiting", Description: "waiting (future wait)",
		Build: func(now time.Time) Filter { return Filter{WaitingOnly: true} },
		Sort:  SortDue,
	},
	"active": {
		Name: "active", Description: "in progress (started)",
		Build: func(now time.Time) Filter { return Filter{ActiveOnly: true} },
		Sort:  SortUrgency,
	},
}

// LookupReport returns the named report (case-insensitive) and whether it exists.
func LookupReport(name string) (Report, bool) {
	r, ok := reports[strings.ToLower(name)]
	return r, ok
}

// ReportNames returns the report names sorted, for help and completion.
func ReportNames() []string {
	names := make([]string, 0, len(reports))
	for n := range reports {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
