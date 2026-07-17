package tui

import (
	"strings"

	"github.com/mustachius/taskframe/internal/i18n"
)

// hintState selects which key legend the bottom line shows.
type hintState int

const (
	hintList hintState = iota
	hintSidebar
	hintSearch
	hintModal
)

// hintState maps the app's input focus to a legend.
func (a *App) hintState() hintState {
	switch {
	case a.modal != nil:
		return hintModal
	case a.searching:
		return hintSearch
	case a.focus == focusSidebar:
		return hintSidebar
	default:
		return hintList
	}
}

// renderHint draws the dim one-line key legend that replaced the F-key bar.
// Every F-key still works — letters are just the advertised spelling.
func renderHint(th Theme, lang i18n.Lang, ascii bool, st hintState, w int) string {
	key := map[hintState]string{
		hintList:    "hint.list",
		hintSidebar: "hint.sidebar",
		hintSearch:  "hint.search",
		hintModal:   "hint.modal",
	}[st]
	s := lang.T(key)
	if ascii {
		s = strings.NewReplacer("·", "|", "↑↓", "^v").Replace(s)
	}
	return padRow(th.Dim.Render(" "+truncRunes(s, w-1)), w, th.Bg)
}
