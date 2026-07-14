package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// parseHex reads a "#rrggbb" string into 0-255 components.
func parseHex(s string) (r, g, b int, ok bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return 0, 0, 0, false
	}
	return r, g, b, true
}

func lerp(a, b int, t float64) int { return int(float64(a) + (float64(b)-float64(a))*t + 0.5) }

// GradientLine colors s cell by cell along a left-to-right gradient between the
// hex colors from and to. width is the reference span for the ramp (pass the
// widest line so a multi-line block shares vertical color columns). Truecolor
// output; lipgloss/termenv downsamples on limited terminals. Falls back to the
// plain string when an endpoint is not "#rrggbb".
func GradientLine(s, from, to string, width int) string {
	fr, fg, fb, ok1 := parseHex(from)
	tr, tg, tb, ok2 := parseHex(to)
	runes := []rune(s)
	if !ok1 || !ok2 || len(runes) == 0 {
		return s
	}
	if width < 2 {
		width = len(runes)
	}
	var b strings.Builder
	for i, ch := range runes {
		t := 0.0
		if width > 1 {
			t = float64(i) / float64(width-1)
			if t > 1 {
				t = 1
			}
		}
		col := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", lerp(fr, tr, t), lerp(fg, tg, t), lerp(fb, tb, t)))
		b.WriteString(lipgloss.NewStyle().Foreground(col).Render(string(ch)))
	}
	return b.String()
}
