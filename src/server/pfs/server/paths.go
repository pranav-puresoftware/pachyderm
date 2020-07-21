package server

import (
	"strings"
)

func pathIsChild(parent, child string) bool {
	parent = strings.TrimLeft(parent, "/")
	child = strings.TrimLeft(child, "/")
	if !strings.HasPrefix(child, parent) {
		return false
	}
	rel := child[len(parent):]
	rel = strings.Trim(rel, "/")
	return !strings.Contains(rel, "/")
}
