package ui

import "github.com/charmbracelet/x/ansi"

// StripANSI removes every escape sequence, leaving the plain visible text.
// The TUI uses it to re-style live content as a dimmed modal backdrop.
func StripANSI(s string) string { return ansi.Strip(s) }
