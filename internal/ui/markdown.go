package ui

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	mdMu    sync.Mutex
	mdCache = map[string]*glamour.TermRenderer{}
)

// RenderMarkdown renders md as styled terminal output word-wrapped to width,
// using the given glamour builtin style ("dark", "dracula", "tokyo-night", …)
// or "notty" for no color. Renderers are cached per (style, width) because
// building one is expensive. On any error it degrades to the raw markdown.
//
// This package stays string-only (no task/store imports) to preserve the shared
// visual layer's position at the bottom of the dependency graph.
func RenderMarkdown(md string, width int, style string) string {
	if width < 1 {
		width = 80
	}
	if style == "" {
		style = "dark"
	}
	key := fmt.Sprintf("%s:%d", style, width)

	mdMu.Lock()
	defer mdMu.Unlock()

	r, ok := mdCache[key]
	if !ok {
		// Pin glamour to the same color profile lipgloss uses, so markdown
		// renders consistently with the rest of the UI. Relying on glamour's own
		// auto-detection can fall back to no color and leave literal ** markers.
		prof := lipgloss.ColorProfile()
		if style != "notty" && prof == termenv.Ascii {
			// The rest of the UI is clearly rendering in color, so an Ascii
			// result here is almost certainly a detection miss; use truecolor.
			prof = termenv.TrueColor
		}
		var err error
		r, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle(style),
			glamour.WithColorProfile(prof),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return md
		}
		mdCache[key] = r
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return out
}
