package repl

import (
	"strings"

	"github.com/mustachius/taskframe/internal/ui"
)

// ThemeNamesList returns the valid theme names (for completion).
func ThemeNamesList() []string { return ui.ThemeNames }

var verbs = []string{"add", "list", "done", "del", "note", "edit", "move", "undo", "purge"}
var slashCmds = []string{"/help", "/theme", "/sort", "/lang", "/sync", "/clear", "/quit", "/classic"}
var sortModes = []string{"urgency", "due", "created"}
var syncVerbs = []string{"init", "status", "pull", "push"}

// complete performs Tab-completion on the current input token, replacing it or
// showing candidates in the transient hint line.
func (m *model) complete() {
	val := m.input.Value()
	fields := strings.Fields(val)
	trailingSpace := strings.HasSuffix(val, " ")

	// the token currently being typed
	cur := ""
	if len(fields) > 0 && !trailingSpace {
		cur = fields[len(fields)-1]
	}
	firstToken := len(fields) == 0 || (len(fields) == 1 && !trailingSpace)

	var prev string
	if n := len(fields); n > 0 {
		if trailingSpace {
			prev = fields[n-1]
		} else if n >= 2 {
			prev = fields[n-2]
		}
	}

	var cands []string
	switch {
	case firstToken && strings.HasPrefix(cur, "/"):
		cands = filterPrefix(slashCmds, cur)
	case firstToken:
		cands = filterPrefix(verbs, cur)
	case strings.HasPrefix(cur, "pro:"), strings.HasPrefix(cur, "project:"):
		sub := cur[strings.Index(cur, ":")+1:]
		cands = withPrefix("pro:", filterPrefix(m.projects, sub))
	case strings.HasPrefix(cur, "+"):
		cands = withPrefix("+", filterPrefix(m.tags, cur[1:]))
	case prev == "/theme":
		cands = filterPrefix(ThemeNamesList(), cur)
	case prev == "/sort":
		cands = filterPrefix(sortModes, cur)
	case prev == "/sync":
		cands = filterPrefix(syncVerbs, cur)
	default:
		m.compHint = ""
		return
	}

	if len(cands) == 0 {
		m.compHint = ""
		return
	}

	// rebuild input: everything before the current token + the completion
	base := val
	if !trailingSpace && len(fields) > 0 {
		base = strings.TrimSuffix(val, fields[len(fields)-1])
	}

	if len(cands) == 1 {
		m.input.SetValue(base + cands[0] + " ")
		m.input.CursorEnd()
		m.compHint = ""
		return
	}
	// multiple: complete to the common prefix, list candidates as a hint
	cp := commonPrefix(cands)
	if cp != "" && cp != tokenAtCursor(val, trailingSpace, fields) {
		m.input.SetValue(base + cp)
		m.input.CursorEnd()
	}
	m.compHint = strings.Join(cands, "  ")
}

func tokenAtCursor(val string, trailingSpace bool, fields []string) string {
	if trailingSpace || len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func filterPrefix(cands []string, pfx string) []string {
	var out []string
	for _, c := range cands {
		if strings.HasPrefix(c, pfx) {
			out = append(out, c)
		}
	}
	return out
}

func withPrefix(pfx string, ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = pfx + s
	}
	return out
}

func commonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	p := ss[0]
	for _, s := range ss[1:] {
		for !strings.HasPrefix(s, p) {
			p = p[:len(p)-1]
			if p == "" {
				return ""
			}
		}
	}
	return p
}
