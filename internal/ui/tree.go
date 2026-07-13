package ui

import "strings"

// TreePrefix renders the box-drawing connector prefix for a node in a tree,
// shared by the REPL overlay and the CLI list so both draw identical branches.
// lastStack holds, for each level from the node's first non-root ancestor down
// to the node itself, whether that level is the last child among its siblings;
// it is empty for top-level (root) nodes, which get no prefix. The last entry
// becomes a branch glyph (├─/└─); earlier entries become trunk (│ / blank).
// It returns plain text (no ANSI) — the CLI relies on that.
func TreePrefix(lastStack []bool, ascii bool) string {
	if len(lastStack) == 0 {
		return ""
	}
	var b strings.Builder
	for i, last := range lastStack {
		leaf := i == len(lastStack)-1
		switch {
		case leaf && last && ascii:
			b.WriteString("`- ")
		case leaf && last:
			b.WriteString("└─ ")
		case leaf && ascii:
			b.WriteString("|- ")
		case leaf:
			b.WriteString("├─ ")
		case last: // ancestor was the last child: empty trunk
			b.WriteString("   ")
		case ascii:
			b.WriteString("|  ")
		default:
			b.WriteString("│  ")
		}
	}
	return b.String()
}
