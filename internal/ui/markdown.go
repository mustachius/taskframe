package ui

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/glamour"
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
		style := glamour.WithAutoStyle()
		if ascii {
			style = glamour.WithStandardStyle("notty")
		}
		var err error
		r, err = glamour.NewTermRenderer(style, glamour.WithWordWrap(width))
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
