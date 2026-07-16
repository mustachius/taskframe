package ui

import (
	"fmt"
	"time"
)

// FormatElapsed renders a duration since a task was started, compactly:
// "<1m", "37m", "1h25m" (or "2h" on the hour), "2d3h" (or "2d" on the day).
// There is no live ticking anywhere — values refresh whenever the UI
// re-renders (any key, any reload), never on a timer.
func FormatElapsed(d time.Duration) string {
	if d < time.Minute {
		return "<1m" // includes negative values (clock skew)
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) - h*60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) - days*24
	if h == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, h)
}
