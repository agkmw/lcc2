package screens

import (
	"strings"
	"testing"
)

// $EDITOR wins, then $VISUAL, then $PAGER, then less -R. Arguments in
// the env value (e.g. "code -w") must survive the split.
func TestResolveViewer(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	t.Setenv("PAGER", "")
	if v := resolveViewer(); strings.Join(v, " ") != "less -R" {
		t.Fatalf("default = %v", v)
	}

	t.Setenv("PAGER", "moar")
	if v := resolveViewer(); strings.Join(v, " ") != "moar" {
		t.Fatalf("pager = %v", v)
	}

	t.Setenv("VISUAL", "code -w")
	if v := resolveViewer(); len(v) != 2 || v[0] != "code" || v[1] != "-w" {
		t.Fatalf("visual = %v", v)
	}

	t.Setenv("EDITOR", "nvim")
	if v := resolveViewer(); v[0] != "nvim" {
		t.Fatalf("editor = %v", v)
	}
}
