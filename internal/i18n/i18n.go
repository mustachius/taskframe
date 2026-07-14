// Package i18n is the presentation-layer message catalog. English is the
// canonical (default) language; pt-br is the optional translation.
//
// It is a leaf package (imports nothing from the project), so ui/repl/tui/cli
// can all use it without breaking the strict layering. Domain errors in
// internal/task and internal/store are intentionally NOT routed through here —
// they stay in English regardless of language.
package i18n

import "fmt"

// Lang is a supported UI language.
type Lang string

const (
	EN Lang = "en"
	PT Lang = "pt-br"
)

// Names lists the languages in toggle order (English first = default).
var Names = []Lang{EN, PT}

// Normalize maps an arbitrary string to a supported Lang, defaulting to EN.
func Normalize(s string) Lang {
	switch Lang(s) {
	case EN, PT:
		return Lang(s)
	default:
		return EN
	}
}

// Next returns the next language in cycle order (flips between the two).
func Next(l Lang) Lang {
	for i, n := range Names {
		if n == l {
			return Names[(i+1)%len(Names)]
		}
	}
	return Names[0]
}

// idx returns the catalog column for the language (0 = en, 1 = pt-br).
func (l Lang) idx() int {
	if l == PT {
		return 1
	}
	return 0
}

// T looks up key in the catalog for the language. A missing translation falls
// back to English; a missing key returns the key itself (so gaps are visible).
func (l Lang) T(key string) string {
	pair, ok := catalog[key]
	if !ok {
		return key
	}
	s := pair[l.idx()]
	if s == "" {
		s = pair[0] // fall back to English
	}
	return s
}

// Tf is T followed by fmt.Sprintf, for messages with format verbs.
func (l Lang) Tf(key string, a ...any) string {
	return fmt.Sprintf(l.T(key), a...)
}
