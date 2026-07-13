package task

import "time"

// UrgencyCoefficients holds the weights of the urgency formula.
// Hardcoded for now; may become configurable later.
type UrgencyCoefficients struct {
	Due        float64 // multiplied by dueFactor (0..1.3)
	PriorityH  float64
	PriorityM  float64
	PriorityL  float64
	TagNext    float64
	Blocking   float64 // task has pending children
	Active     float64 // task is started
	AgePerDay  float64
	AgeCap     float64
	HasProject float64
	Waiting    float64 // negative
}

var DefaultCoefficients = UrgencyCoefficients{
	Due:        12.0,
	PriorityH:  6.0,
	PriorityM:  3.9,
	PriorityL:  1.8,
	TagNext:    15.0,
	Blocking:   2.0,
	Active:     4.0,
	AgePerDay:  0.02,
	AgeCap:     2.0,
	HasProject: 1.0,
	Waiting:    -3.0,
}

// Urgency computes the sort score for a task at a given moment.
// hasPendingChildren must be supplied by the caller (the domain type may
// not have Children loaded).
func Urgency(t *Task, now time.Time, hasPendingChildren bool) float64 {
	c := DefaultCoefficients
	score := 0.0

	score += c.Due * dueFactor(t.Due, now)

	switch t.Priority {
	case PriorityHigh:
		score += c.PriorityH
	case PriorityMed:
		score += c.PriorityM
	case PriorityLow:
		score += c.PriorityL
	}

	if t.HasTag("next") {
		score += c.TagNext
	}
	if hasPendingChildren {
		score += c.Blocking
	}
	if t.IsActive() {
		score += c.Active
	}

	age := now.Sub(t.CreatedAt).Hours() / 24 * c.AgePerDay
	if age > c.AgeCap {
		age = c.AgeCap
	}
	if age > 0 {
		score += age
	}

	if t.Project != "" {
		score += c.HasProject
	}
	if t.IsWaiting(now) {
		score += c.Waiting
	}
	return score
}

// dueFactor ramps linearly from 0 at 14 days before due to 1.0 at the due
// date, continuing to 1.3 at 7 days overdue (then capped).
func dueFactor(due *time.Time, now time.Time) float64 {
	if due == nil {
		return 0
	}
	days := due.Sub(now).Hours() / 24 // positive = in the future
	switch {
	case days >= 14:
		return 0
	case days <= -7:
		return 1.3
	case days <= 0: // overdue: 1.0 → 1.3 over 7 days
		return 1.0 + (-days/7)*0.3
	default: // 0..14 days out: 1.0 → 0
		return 1.0 - days/14
	}
}
