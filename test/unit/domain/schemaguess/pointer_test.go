package schemaguess_test

import (
	"testing"

	"github.com/Olian04/guesschema/internal/domain/schemaguess"
)

func TestJoinPointer(t *testing.T) {
	t.Parallel()
	if g, e := schemaguess.JoinPointer("", "a"), "/a"; g != e {
		t.Fatalf("root join: got %q want %q", g, e)
	}
	if g, e := schemaguess.JoinPointer("/a", "b/c"), "/a/b~1c"; g != e {
		t.Fatalf("escape /: got %q want %q", g, e)
	}
	if g, e := schemaguess.JoinPointer("/a", "b~c"), "/a/b~0c"; g != e {
		t.Fatalf("escape ~: got %q want %q", g, e)
	}
}

func TestSplitPointer(t *testing.T) {
	t.Parallel()
	if g := schemaguess.SplitPointer("/a/b~1c"); len(g) != 2 || g[0] != "a" || g[1] != "b/c" {
		t.Fatalf("got %#v", g)
	}
}
