package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ParseTokens interprets taskwarrior-style arguments shared by the CLI and
// the REPL: `+tag`, `-tag` (exclude, list only), `pro:x`, `due:x`, `prio:H`,
// `wait:x`, `recur:x`, `sub:n`, `status:x`, and the `all` list modifier.
// Non-token words are joined and returned as free text (title for add, search
// text for list).
func ParseTokens(args []string, now time.Time) (t Task, filter Filter, text string, err error) {
	var words []string
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "+") && len(a) > 1:
			t.Tags = append(t.Tags, a[1:])
			filter.Tags = append(filter.Tags, a[1:])
		case len(a) > 1 && a[0] == '-' && isTagStart(rune(a[1])):
			// exclusion is a filter concept only; never a task attribute
			filter.ExcludeTags = append(filter.ExcludeTags, a[1:])
		case strings.HasPrefix(a, "pro:"), strings.HasPrefix(a, "project:"):
			v := a[strings.Index(a, ":")+1:]
			t.Project = v
			filter.Project = v
		case strings.HasPrefix(a, "due:"):
			d, perr := ParseDate(a[4:], now)
			if perr != nil {
				err = perr
				return
			}
			t.Due = &d
			filter.DueBefore = &d
		case strings.HasPrefix(a, "prio:"), strings.HasPrefix(a, "priority:"):
			v := strings.ToUpper(a[strings.Index(a, ":")+1:])
			if v != "H" && v != "M" && v != "L" && v != "" {
				err = fmt.Errorf("prioridade inválida: %s (use H, M ou L)", v)
				return
			}
			t.Priority = Priority(v)
		case strings.HasPrefix(a, "wait:"):
			d, perr := ParseDate(a[5:], now)
			if perr != nil {
				err = perr
				return
			}
			t.Wait = &d
		case strings.HasPrefix(a, "recur:"):
			if _, rerr := NextRecurrence(a[6:], now); rerr != nil {
				err = rerr
				return
			}
			t.Recur = a[6:]
		case strings.HasPrefix(a, "sub:"):
			id, perr := strconv.ParseInt(a[4:], 10, 64)
			if perr != nil {
				err = fmt.Errorf("sub: espera um id numérico")
				return
			}
			t.ParentID = id
		case strings.HasPrefix(a, "status:"):
			v := strings.ToLower(a[7:])
			switch v {
			case "all":
				filter.IncludeAll = true
			case "pending", "done", "deleted":
				filter.Status = Status(v)
			default:
				err = fmt.Errorf("status inválido: %s (use pending, done, deleted, all)", v)
				return
			}
		case a == "all":
			filter.IncludeAll = true
		case a == "nocontext":
			filter.NoContext = true
		default:
			words = append(words, a)
		}
	}
	text = strings.Join(words, " ")
	return
}

// isTagStart reports whether r can begin a tag name (so "-work" is an exclude
// filter but "-5" or a lone "-" is treated as free text).
func isTagStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
