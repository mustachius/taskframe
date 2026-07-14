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

// RenderMarkdown renders md as styled terminal output word-wrapped to width. It
// uses glamour's auto (dark/light) style, or the plain "notty" style under
// ascii. Renderers are cached per (width, style) because building one is
// expensive. On any error it degrades to returning the raw markdown.
//
// This package stays string-only (no task/store imports) to preserve the shared
// visual layer's position at the bottom of the dependency graph.
func RenderMarkdown(md string, width int, ascii bool) string {
	if width < 1 {
		width = 80
	}
	key := fmt.Sprintf("%v:%d", ascii, width)

	mdMu.Lock()
	defer mdMu.Unlock()

	r, ok := mdCache[key]
	if !ok {
		// Pin glamour to the same color profile lipgloss uses, so markdown
		// renders (bold, colors) consistently with the rest of the UI. Relying
		// on glamour's own auto-detection can fall back to no color and leave
		// literal ** markers in the output.
		prof := lipgloss.ColorProfile()
		if !ascii && prof == termenv.Ascii {
			// The rest of the UI (gradient, boxes) is clearly rendering in color,
			// so a fallback to Ascii here is almost certainly a detection miss;
			// use truecolor so notes actually render styled.
			prof = termenv.TrueColor
		}
		style := "dark"
		if ascii || prof == termenv.Ascii {
			style = "notty"
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
