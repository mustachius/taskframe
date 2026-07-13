package task

import (
	"testing"
	"time"
)

var base = time.Date(2026, 7, 12, 10, 0, 0, 0, time.Local) // a Sunday

func TestParseDate(t *testing.T) {
	cases := []struct {
		in   string
		want time.Time
	}{
		{"today", time.Date(2026, 7, 12, 23, 59, 59, 0, time.Local)},
		{"tomorrow", time.Date(2026, 7, 13, 23, 59, 59, 0, time.Local)},
		{"3d", time.Date(2026, 7, 15, 23, 59, 59, 0, time.Local)},
		{"1w", time.Date(2026, 7, 19, 23, 59, 59, 0, time.Local)},
		{"fri", time.Date(2026, 7, 17, 23, 59, 59, 0, time.Local)},
		{"sex", time.Date(2026, 7, 17, 23, 59, 59, 0, time.Local)},
		{"2026-08-01", time.Date(2026, 8, 1, 23, 59, 59, 0, time.Local)},
		{"01/08", time.Date(2026, 8, 1, 23, 59, 59, 0, time.Local)},
		{"01/06", time.Date(2027, 6, 1, 23, 59, 59, 0, time.Local)}, // past → next year
		{"eom", time.Date(2026, 7, 31, 23, 59, 59, 0, time.Local)},
	}
	for _, c := range cases {
		got, err := ParseDate(c.in, base)
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("%q: got %v, want %v", c.in, got, c.want)
		}
	}
	if _, err := ParseDate("nonsense", base); err == nil {
		t.Error("expected error for nonsense date")
	}
}

func TestUrgencyOrdering(t *testing.T) {
	now := base
	overdue := now.AddDate(0, 0, -2)
	farFuture := now.AddDate(0, 0, 60)

	urgent := &Task{Title: "a", Priority: PriorityHigh, Due: &overdue, CreatedAt: now}
	relaxed := &Task{Title: "b", Due: &farFuture, CreatedAt: now}
	if Urgency(urgent, now, false) <= Urgency(relaxed, now, false) {
		t.Error("overdue high-priority task should outrank far-future task")
	}

	next := &Task{Title: "c", Tags: []string{"next"}, CreatedAt: now}
	plain := &Task{Title: "d", Priority: PriorityHigh, CreatedAt: now}
	if Urgency(next, now, false) <= Urgency(plain, now, false) {
		t.Error("+next tag should pin above priority alone")
	}

	wait := now.AddDate(0, 0, 5)
	waiting := &Task{Title: "e", Wait: &wait, CreatedAt: now}
	if Urgency(waiting, now, false) >= 0 {
		t.Error("waiting task should have negative urgency contribution")
	}
}

func TestNextRecurrence(t *testing.T) {
	from := base
	cases := map[string]time.Time{
		"daily":   from.AddDate(0, 0, 1),
		"weekly":  from.AddDate(0, 0, 7),
		"monthly": from.AddDate(0, 1, 0),
		"3d":      from.AddDate(0, 0, 3),
		"2w":      from.AddDate(0, 0, 14),
	}
	for in, want := range cases {
		got, err := NextRecurrence(in, from)
		if err != nil || !got.Equal(want) {
			t.Errorf("%q: got %v (%v), want %v", in, got, err, want)
		}
	}
	if _, err := NextRecurrence("xx", from); err == nil {
		t.Error("expected error")
	}
}

func TestConfigureUrgency(t *testing.T) {
	defer func() { ActiveCoefficients = DefaultCoefficients }()

	ConfigureUrgency(map[string]float64{"hasProject": 50, "active": 9})
	if ActiveCoefficients.HasProject != 50 || ActiveCoefficients.Active != 9 {
		t.Fatalf("overrides not applied: %+v", ActiveCoefficients)
	}
	// unspecified coefficient keeps its default
	if ActiveCoefficients.Due != DefaultCoefficients.Due {
		t.Fatal("unspecified coefficient should keep default")
	}
	// reconfigure resets from defaults; unknown keys are ignored
	ConfigureUrgency(map[string]float64{"nope": 1})
	if ActiveCoefficients.HasProject != DefaultCoefficients.HasProject {
		t.Fatal("reconfigure should reset from defaults")
	}
}
