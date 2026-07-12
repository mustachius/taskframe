package repl

// history is an in-memory command recall buffer (↑/↓). v1 does not persist.
type history struct {
	items []string
	idx   int
	stash string // in-progress line saved when browsing back
}

func (h *history) push(line string) {
	if n := len(h.items); n > 0 && h.items[n-1] == line {
		h.idx = len(h.items) // dedupe consecutive; reset pointer
		return
	}
	h.items = append(h.items, line)
	h.idx = len(h.items)
}

// prev moves back in history, stashing the current input on first step.
func (h *history) prev(current string) (string, bool) {
	if len(h.items) == 0 {
		return "", false
	}
	if h.idx == len(h.items) {
		h.stash = current
	}
	if h.idx > 0 {
		h.idx--
	}
	return h.items[h.idx], true
}

// next moves forward; past the newest entry it restores the stashed line.
func (h *history) next() (string, bool) {
	if len(h.items) == 0 {
		return "", false
	}
	if h.idx < len(h.items) {
		h.idx++
	}
	if h.idx >= len(h.items) {
		return h.stash, true
	}
	return h.items[h.idx], true
}
