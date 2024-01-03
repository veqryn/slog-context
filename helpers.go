package slogctx

import (
	"log/slog"
)

// groupOrAttrs holds either a group name or a list of slog.Attrs.
// It also holds a reference/link to its parent groupOrAttrs, forming a linked list.
// Courtesy of https://github.com/jba/slog/blob/b5eef75b08965b871bd5214891313b73d5a30432/withsupport/withsupport.go
type groupOrAttrs struct {
	group string        // group name if non-empty
	attrs []slog.Attr   // attrs if non-empty
	next  *groupOrAttrs // parent
}

// WithGroup returns a new groupOrAttrs that includes the given group, and links to the old groupOrAttrs.
// Safe to call on a nil groupOrAttrs.
func (g *groupOrAttrs) WithGroup(name string) *groupOrAttrs {
	// Empty-name groups are inlined as if they didn't exist
	if name == "" {
		return g
	}
	return &groupOrAttrs{
		group: name,
		next:  g,
	}
}

// WithAttrs returns a new groupOrAttrs that includes the given attrs, and links to the old groupOrAttrs.
// Safe to call on a nil groupOrAttrs.
func (g *groupOrAttrs) WithAttrs(attrs []slog.Attr) *groupOrAttrs {
	if len(attrs) == 0 {
		return g
	}
	return &groupOrAttrs{
		attrs: attrs,
		next:  g,
	}
}
