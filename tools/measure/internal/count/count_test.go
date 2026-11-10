package count_test

import (
	"testing"

	"github.com/JaimeStill/spike-graph/tools/measure/internal/count"
)

// measure counts src as a file, failing the test on a parse error.
func measure(t *testing.T, src string) count.Counts {
	t.Helper()
	c, err := count.File("src.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Code lines leave out blank lines, comment-only lines of either form, and
// the import block, and keep a line that ends in a comment and every line
// of a raw string.
func TestFile_Lines(t *testing.T) {
	src := `// Package p is a doc comment.
package p

import (
	"fmt"
	"strings"
)

/*
A block comment.
*/
var s = ` + "`" + `one
two` + "`" + ` // trailing

func f() {
	// inside
	fmt.Println(strings.ToUpper(s))
}
`
	c := measure(t, src)
	if c.Raw != 18 {
		t.Errorf("Raw = %d, want 18", c.Raw)
	}
	// package, var s (two lines), func f, Println, closing brace.
	if c.Code != 6 {
		t.Errorf("Code = %d, want 6", c.Code)
	}
}

// A last line without a newline is still a raw line, and a single import
// is out of the code lines too.
func TestFile_LinesUnterminated(t *testing.T) {
	c := measure(t, "package p\n\nimport \"fmt\"\n\nvar _ = fmt.Sprint")
	if c.Raw != 5 || c.Code != 2 {
		t.Errorf("Raw, Code = %d, %d, want 5, 2", c.Raw, c.Code)
	}
}

// Every function literal is a closure, nested ones included.
func TestFile_Closures(t *testing.T) {
	c := measure(t, `package p

var g = func() {}

func f() func() int {
	return func() int {
		h := func() int { return 1 }
		return h()
	}
}
`)
	if c.Closures != 3 {
		t.Errorf("Closures = %d, want 3", c.Closures)
	}
}

// A wrapper returns a func taking a *graph.Scope first, under whatever name
// the graph package is imported; a constructor itself, a func returning a
// func of other parameters, and a two-result func are not wrappers.
func TestFile_Wrappers(t *testing.T) {
	c := measure(t, `package p

import g "example.com/sdk/graph"

type N struct{}

func newA(n *N) func(*g.Scope) (int, error) { return nil }

func (n *N) newB() func(s *g.Scope, x int) (int, error) { return nil }

func (n *N) newC(s *g.Scope) (int, error) { return 0, nil }

func other() func(int, *g.Scope) error { return nil }

func two() (func(*g.Scope) error, error) { return nil, nil }
`)
	if c.Wrappers != 2 {
		t.Errorf("Wrappers = %d, want 2", c.Wrappers)
	}
}

// Without the graph package imported, nothing is a wrapper or an ordering
// site, whatever the names.
func TestFile_NoGraphImport(t *testing.T) {
	c := measure(t, `package p

import "example.com/other"

func newA() func(*graph.Scope) error { return nil }

func f(s *graph.Scope) { s.After(1); graph.After(2); other.After(3) }
`)
	if c.Wrappers != 0 || c.ScopeAfter != 0 || c.GraphAfter != 0 {
		t.Errorf("Wrappers, ScopeAfter, GraphAfter = %d, %d, %d, want 0, 0, 0", c.Wrappers, c.ScopeAfter, c.GraphAfter)
	}
}

// Name comments are the quoted names on fields of the struct Nodes, leading
// or trailing, a description after the name allowed. A quoted string in a
// comment on a func, a var, another struct's field, or Nodes itself, or one
// later in a field's comment, is not one.
func TestFile_NameComments(t *testing.T) {
	c := measure(t, `package p

// "nodes" names the type, not a field.
type Nodes struct {
	A int // "a"
	// "b", the second
	B int
	C int // the "c" node
	D int
}

// "f" is not a field.
func f() {}

// "v" is not a field either.
var v = 1

type Other struct {
	X int // "x"
}
`)
	if c.NameComments != 2 {
		t.Errorf("NameComments = %d, want 2", c.NameComments)
	}
}

// s.After counts After calls on a *graph.Scope parameter inside function
// bodies, in closures too; graph.After counts the package's After option.
// After on anything else, a time or another package, is neither.
func TestFile_After(t *testing.T) {
	c := measure(t, `package p

import (
	"time"

	"example.com/go-core/graph"
)

func newA(n int) func(*graph.Scope) (int, error) {
	return func(s *graph.Scope) (int, error) {
		s.After(n)
		for range 2 {
			s.After(n)
		}
		return 0, nil
	}
}

func (x *X) newB(sc *graph.Scope) (int, error) {
	sc.After(x)
	return 0, nil
}

func define(g *graph.Graph, t time.Time) {
	g.Define("a", nil, graph.After(1), graph.After(2, 3))
	_ = t.After(t)
	_ = time.After(0)
}
`)
	if c.ScopeAfter != 3 {
		t.Errorf("ScopeAfter = %d, want 3", c.ScopeAfter)
	}
	if c.GraphAfter != 2 {
		t.Errorf("GraphAfter = %d, want 2", c.GraphAfter)
	}
}

// Lines counts lines alone: the shapes stay zero.
func TestLines(t *testing.T) {
	c, err := count.Lines("x_test.go", []byte(`package p_test

import "testing"

func TestX(t *testing.T) { t.Run("x", func(t *testing.T) {}) }
`))
	if err != nil {
		t.Fatal(err)
	}
	want := count.Counts{Code: 2, Raw: 5}
	if c != want {
		t.Errorf("Lines = %+v, want %+v", c, want)
	}
}

// A nil source is a file absent at one side of the comparison: it counts
// zero rather than failing to parse.
func TestAbsent(t *testing.T) {
	for name, fn := range map[string]func(string, []byte) (count.Counts, error){"File": count.File, "Lines": count.Lines} {
		c, err := fn("absent.go", nil)
		if err != nil || c != (count.Counts{}) {
			t.Errorf("%s(nil) = %+v, %v, want zero counts, nil", name, c, err)
		}
	}
}

// Source that does not parse is an error naming the file.
func TestFile_ParseError(t *testing.T) {
	if _, err := count.File("bad.go", []byte("package p\nfunc {")); err == nil {
		t.Error("File(bad source) = nil error, want one")
	}
}

// Add sums every field.
func TestCounts_Add(t *testing.T) {
	a := count.Counts{Code: 1, Raw: 2, Closures: 3, Wrappers: 4, NameComments: 5, ScopeAfter: 6, GraphAfter: 7}
	want := count.Counts{Code: 2, Raw: 4, Closures: 6, Wrappers: 8, NameComments: 10, ScopeAfter: 12, GraphAfter: 14}
	if got := a.Add(a); got != want {
		t.Errorf("Add = %+v, want %+v", got, want)
	}
}
