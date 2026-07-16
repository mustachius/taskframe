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

// blendHex interpolates two "#rrggbb" colors at t in [0,1].
func blendHex(fr, fg, fb, tr, tg, tb int, t float64) lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", lerp(fr, tr, t), lerp(fg, tg, t), lerp(fb, tb, t)))
}

// GradientBlock colors a block of lines top-to-bottom: each line gets a solid
// color interpolated between the hex endpoints from (first line) and to (last
// line). Truecolor output; lipgloss/termenv downsamples on limited terminals.
// Returns the lines unchanged when an endpoint is not "#rrggbb".
func GradientBlock(lines []string, from, to string) []string {
	fr, fg, fb, ok1 := parseHex(from)
	tr, tg, tb, ok2 := parseHex(to)
	if !ok1 || !ok2 || len(lines) == 0 {
		return lines
	}
	out := make([]string, len(lines))
	n := len(lines)
	for i, line := range lines {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		out[i] = lipgloss.NewStyle().Foreground(blendHex(fr, fg, fb, tr, tg, tb, t)).Render(line)
	}
	return out
}

// GradientLine colors a single plain line left-to-right, one interpolated
// color per rune — GradientBlock's horizontal counterpart, for one-row brands.
// Returns s unchanged when an endpoint is not "#rrggbb".
func GradientLine(s, from, to string) string {
	fr, fg, fb, ok1 := parseHex(from)
	tr, tg, tb, ok2 := parseHex(to)
	runes := []rune(s)
	if !ok1 || !ok2 || len(runes) == 0 {
		return s
	}
	n := len(runes)
	var b strings.Builder
	for i, r := range runes {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		b.WriteString(lipgloss.NewStyle().Foreground(blendHex(fr, fg, fb, tr, tg, tb, t)).Render(string(r)))
	}
	return b.String()
}
