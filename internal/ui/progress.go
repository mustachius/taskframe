package ui

import "strings"

// ProgressBar renders a width-cell bar for frac in [0,1] using the theme's
// accent for the filled part and dim for the remainder. Width-1 block glyphs so
// it aligns everywhere; ascii themes still read fine (▓/░ are box-drawing).
func ProgressBar(frac float64, width int, th Theme) string {
	if width <= 0 {
		return ""
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	full, empty := "█", "░"
	if th.Box == asciiBox {
		full, empty = "#", "-"
	}
	return th.Accent.Render(strings.Repeat(full, filled)) +
		th.Dim.Render(strings.Repeat(empty, width-filled))
}
