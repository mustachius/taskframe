package task

import (
	"fmt"
	"strconv"
	"strings"
)

// maxIDSpecSpan caps how many ids a single range may expand to, so a typo like
// "1-999999" fails loudly instead of allocating a huge slice.
const maxIDSpecSpan = 10000

// ParseIDSpec parses the task-id selectors shared by the CLI and the REPL. Each
// space-separated argument may be a single id ("5"), a comma-separated list
// ("1,5"), or an inclusive range ("1-3"). Ids come back in first-seen order
// with duplicates removed. Empty input is an error (callers require ≥1 id).
func ParseIDSpec(args []string) ([]int64, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("informe pelo menos um id")
	}
	var ids []int64
	seen := map[int64]bool{}
	add := func(id int64) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, a := range args {
		for _, part := range strings.Split(a, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if lo, hi, ok := strings.Cut(part, "-"); ok {
				a1, e1 := strconv.ParseInt(strings.TrimSpace(lo), 10, 64)
				a2, e2 := strconv.ParseInt(strings.TrimSpace(hi), 10, 64)
				if e1 != nil || e2 != nil {
					return nil, fmt.Errorf("range inválido: %s", part)
				}
				if a2 < a1 {
					a1, a2 = a2, a1
				}
				if a2-a1 > maxIDSpecSpan {
					return nil, fmt.Errorf("range grande demais: %s", part)
				}
				for id := a1; id <= a2; id++ {
					add(id)
				}
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("id inválido: %s", part)
			}
			add(id)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("informe pelo menos um id")
	}
	return ids, nil
}
