package task

import "testing"

func TestReports(t *testing.T) {
	for _, name := range []string{"next", "overdue", "today", "week", "waiting"} {
		r, ok := LookupReport(name)
		if !ok {
			t.Fatalf("report %q missing", name)
		}
		if r.Name != name {
			t.Fatalf("report %q has Name=%q", name, r.Name)
		}
		if r.Build == nil {
			t.Fatalf("report %q has nil Build", name)
		}
	}
	if _, ok := LookupReport("nope"); ok {
		t.Fatal("unknown report should not resolve")
	}

	now := base
	if f := reports["next"].Build(now); !f.HideWaiting || f.DueBefore != nil {
		t.Fatalf("next filter unexpected: %+v", f)
	}
	if f := reports["overdue"].Build(now); f.DueBefore == nil || !f.DueBefore.Equal(now) {
		t.Fatalf("overdue should be DueBefore=now: %+v", f)
	}
	if f := reports["today"].Build(now); f.DueBefore == nil || !f.DueBefore.Equal(EndOfDay(now)) {
		t.Fatalf("today should be DueBefore=EndOfDay: %+v", f)
	}
	if f := reports["waiting"].Build(now); !f.WaitingOnly {
		t.Fatalf("waiting should set WaitingOnly: %+v", f)
	}
}

func TestFilterMerge(t *testing.T) {
	rep := reports["next"].Build(base) // HideWaiting
	extra := Filter{Project: "work", Tags: []string{"a"}, ExcludeTags: []string{"b"}, IncludeAll: true}
	m := rep.Merge(extra)
	if m.Project != "work" {
		t.Fatalf("merge should take extra Project, got %q", m.Project)
	}
	if !m.HideWaiting {
		t.Fatal("merge should keep base HideWaiting")
	}
	if !m.IncludeAll {
		t.Fatal("merge should OR IncludeAll")
	}
	if len(m.Tags) != 1 || len(m.ExcludeTags) != 1 {
		t.Fatalf("merge should union tag slices: %+v", m)
	}
	// empty scalar in extra must not clobber base
	base2 := Filter{Project: "casa"}
	if got := base2.Merge(Filter{}).Project; got != "casa" {
		t.Fatalf("empty merge clobbered project: %q", got)
	}
}
