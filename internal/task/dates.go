package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDate parses human-friendly date expressions into a time anchored at
// end-of-day local time (so "due:today" stays due all day):
//
//	today, tomorrow, yesterday
//	mon..sun / monday..sunday  (next occurrence, not today)
//	eow (friday), eom (last day of month)
//	3d, 2w, 1m, 1y  (relative offsets)
//	2026-07-15, 15/07, 15/07/2026
func ParseDate(s string, now time.Time) (time.Time, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	eod := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
	}

	switch s {
	case "today", "now":
		return eod(now), nil
	case "tomorrow", "tom":
		return eod(now.AddDate(0, 0, 1)), nil
	case "yesterday":
		return eod(now.AddDate(0, 0, -1)), nil
	case "eow":
		return eod(nextWeekday(now, time.Friday)), nil
	case "eom":
		firstNext := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 1, 0)
		return eod(firstNext.AddDate(0, 0, -1)), nil
	}

	if wd, ok := weekdays[s]; ok {
		return eod(nextWeekday(now, wd)), nil
	}

	// relative offsets: 3d, 2w, 1m, 1y
	if len(s) >= 2 {
		if n, err := strconv.Atoi(s[:len(s)-1]); err == nil {
			switch s[len(s)-1] {
			case 'd':
				return eod(now.AddDate(0, 0, n)), nil
			case 'w':
				return eod(now.AddDate(0, 0, n*7)), nil
			case 'm':
				return eod(now.AddDate(0, n, 0)), nil
			case 'y':
				return eod(now.AddDate(n, 0, 0)), nil
			}
		}
	}

	for _, layout := range []string{"2006-01-02", "02/01/2006", "02/01"} {
		if t, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			if layout == "02/01" { // no year given: assume current, roll to next if past
				t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
				if eod(t).Before(now) {
					t = t.AddDate(1, 0, 0)
				}
			}
			return eod(t), nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date: %q", s)
}

var weekdays = map[string]time.Weekday{
	"mon": time.Monday, "monday": time.Monday, "seg": time.Monday,
	"tue": time.Tuesday, "tuesday": time.Tuesday, "ter": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday, "qua": time.Wednesday,
	"thu": time.Thursday, "thursday": time.Thursday, "qui": time.Thursday,
	"fri": time.Friday, "friday": time.Friday, "sex": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday, "sab": time.Saturday,
	"sun": time.Sunday, "sunday": time.Sunday, "dom": time.Sunday,
}

// nextWeekday returns the next occurrence of wd strictly after today.
func nextWeekday(now time.Time, wd time.Weekday) time.Time {
	days := (int(wd) - int(now.Weekday()) + 7) % 7
	if days == 0 {
		days = 7
	}
	return now.AddDate(0, 0, days)
}

// NextRecurrence computes the next due date for a recurring task.
// recur accepts: daily, weekly, monthly, yearly, or Nd/Nw/Nm/Ny.
func NextRecurrence(recur string, from time.Time) (time.Time, error) {
	switch strings.ToLower(recur) {
	case "daily":
		return from.AddDate(0, 0, 1), nil
	case "weekly":
		return from.AddDate(0, 0, 7), nil
	case "monthly":
		return from.AddDate(0, 1, 0), nil
	case "yearly":
		return from.AddDate(1, 0, 0), nil
	}
	if len(recur) >= 2 {
		if n, err := strconv.Atoi(recur[:len(recur)-1]); err == nil && n > 0 {
			switch recur[len(recur)-1] {
			case 'd':
				return from.AddDate(0, 0, n), nil
			case 'w':
				return from.AddDate(0, 0, n*7), nil
			case 'm':
				return from.AddDate(0, n, 0), nil
			case 'y':
				return from.AddDate(n, 0, 0), nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized recurrence: %q", recur)
}
