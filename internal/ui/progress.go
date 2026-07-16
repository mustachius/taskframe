package ui

import "strings"

// progressFilled returns how many of width cells are filled for frac in [0,1].
func progressFilled(frac float64, width int) int {
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
	return filled
}

// ProgressGlyphs returns the unstyled bar runes for frac in [0,1] ("███░░░",
// or "###---" under ascii) — for callers that must apply a single style to the
// whole row (e.g. a cursor line) and cannot nest ANSI sequences inside it.
func ProgressGlyphs(frac float64, width int, ascii bool) string {
	if width <= 0 {
		return ""
	}
	full, empty := "█", "░"
	if ascii {
		full, empty = "#", "-"
	}
	filled := progressFilled(frac, width)
	return strings.Repeat(full, filled) + strings.Repeat(empty, width-filled)
}

// ProgressBar renders a width-cell bar for frac in [0,1] using the theme's
// accent for the filled part and dim for the remainder. Width-1 block glyphs so
// it aligns everywhere; ascii themes still read fine (▓/░ are box-drawing).
func ProgressBar(frac float64, width int, th Theme) string {
	if width <= 0 {
		return ""
	}
	full, empty := "█", "░"
	if th.ASCII() {
		full, empty = "#", "-"
	}
	filled := progressFilled(frac, width)
	return th.Accent.Render(strings.Repeat(full, filled)) +
		th.Dim.Render(strings.Repeat(empty, width-filled))
}
