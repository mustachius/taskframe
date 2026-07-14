package task

// SortMode selects the task list ordering. Stored as a plain string so it
// persists directly in the settings table.
type SortMode string

const (
	SortUrgency SortMode = "urgency"
	SortDue     SortMode = "due"
	SortCreated SortMode = "created"
)

// NormalizeSortMode maps unknown values to the default (urgency).
func NormalizeSortMode(s string) SortMode {
	switch SortMode(s) {
	case SortDue, SortCreated:
		return SortMode(s)
	default:
		return SortUrgency
	}
}

func (m SortMode) Next() SortMode {
	switch m {
	case SortUrgency:
		return SortDue
	case SortDue:
		return SortCreated
	default:
		return SortUrgency
	}
}

// Label is the canonical (English) sort-mode name. UI layers localize it via
// the i18n catalog key "sort.<mode>"; this is the fallback / non-UI name.
func (m SortMode) Label() string {
	switch m {
	case SortDue:
		return "due"
	case SortCreated:
		return "created"
	default:
		return "urgency"
	}
}
