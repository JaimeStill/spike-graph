// Command measure prints, for each of the spike's two composition roots,
// the service's internal/app and slab's tools/slab/internal/app, a table
// comparing the baseline commit against the working tree: the code and raw
// lines of the root's production files and the graph shapes they write,
// and the code and raw lines of its layer tests.
//
// The baseline is read from git (ls-tree and show), so the checkout and the
// working tree are never touched. measure runs from anywhere in the
// repository: `mise run measure`.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JaimeStill/spike-graph/tools/measure/internal/count"
)

// baseline is the commit the working tree is measured against: the spike's
// first merge, the copied go-web-service and slab before the API changed.
const baseline = "2999659"

// root is one composition root: its directory from the repository root and
// its layer test files, which are measured as their own row.
type root struct {
	label      string
	dir        string
	layerTests []string
}

var roots = []root{
	{label: "service", dir: "internal/app", layerTests: []string{"graph_test.go"}},
	{label: "slab", dir: "tools/slab/internal/app", layerTests: []string{"graph_test.go"}},
}

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "measure:", err)
		os.Exit(1)
	}
}

// run measures every root at the baseline and in the working tree and
// writes the tables to w, all at once once every root is measured, so a
// failure prints no partial table.
func run(w io.Writer) error {
	top, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("find the repository root: %w", err)
	}
	repo := strings.TrimSpace(string(top))
	if _, err := git("rev-parse", "--verify", "--quiet", baseline+"^{commit}"); err != nil {
		return fmt.Errorf("cannot read baseline commit %s: %w", baseline, err)
	}
	var out strings.Builder
	for i, r := range roots {
		before, err := measure(r, baselineSource{})
		if err != nil {
			return fmt.Errorf("%s at %s: %w", r.dir, baseline, err)
		}
		after, err := measure(r, treeSource{repo: repo})
		if err != nil {
			return fmt.Errorf("%s in the working tree: %w", r.dir, err)
		}
		if i > 0 {
			out.WriteString("\n")
		}
		write(&out, r, before, after)
	}
	_, err = io.WriteString(w, out.String())
	return err
}

// source reads a root's files at one side of the comparison.
type source interface {
	// list returns the names of the files directly in dir.
	list(dir string) ([]string, error)
	// read returns the file's content, or nil when it is absent, which
	// count measures as zero.
	read(file string) ([]byte, error)
}

// baselineSource reads the baseline commit through git.
type baselineSource struct{}

func (baselineSource) list(dir string) ([]string, error) {
	out, err := git("ls-tree", "--name-only", baseline, dir+"/")
	if err != nil {
		return nil, err
	}
	var names []string
	for line := range strings.Lines(string(out)) {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, path.Base(line))
		}
	}
	return names, nil
}

func (s baselineSource) read(file string) ([]byte, error) {
	names, err := s.list(path.Dir(file))
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, path.Base(file)) {
		return nil, nil
	}
	src, err := git("show", baseline+":"+file)
	if err != nil {
		return nil, err
	}
	// An empty file is present, not absent.
	if src == nil {
		src = []byte{}
	}
	return src, nil
}

// treeSource reads the working tree under repo.
type treeSource struct{ repo string }

func (s treeSource) list(dir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.repo, dir))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (s treeSource) read(file string) ([]byte, error) {
	src, err := os.ReadFile(filepath.Join(s.repo, filepath.FromSlash(file)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// An empty file is present, not absent.
	if src == nil {
		src = []byte{}
	}
	return src, nil
}

// measured is one side of a root's comparison.
type measured struct {
	production count.Counts
	layerTests count.Counts
}

// measure sums r's production files, its non-test .go files but doc.go,
// and its layer tests, each read from src; a layer test absent from src,
// as slab's is at the baseline, counts zero.
func measure(r root, src source) (measured, error) {
	var m measured
	names, err := src.list(r.dir)
	if err != nil {
		return m, err
	}
	for _, name := range names {
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "doc.go" {
			continue
		}
		file := path.Join(r.dir, name)
		b, err := src.read(file)
		if err != nil {
			return m, err
		}
		c, err := count.File(file, b)
		if err != nil {
			return m, err
		}
		m.production = m.production.Add(c)
	}
	for _, name := range r.layerTests {
		file := path.Join(r.dir, name)
		b, err := src.read(file)
		if err != nil {
			return m, err
		}
		c, err := count.Lines(file, b)
		if err != nil {
			return m, err
		}
		m.layerTests = m.layerTests.Add(c)
	}
	return m, nil
}

// write prints r's table: one row per measurement, before, after, and the
// delta.
func write(w *strings.Builder, r root, before, after measured) {
	type row struct {
		label  string
		before int
		after  int
	}
	b, a := before.production, after.production
	tb, ta := before.layerTests, after.layerTests
	rows := []row{
		{"code lines", b.Code, a.Code},
		{"raw lines", b.Raw, a.Raw},
		{"closures", b.Closures, a.Closures},
		{"wrapper constructors", b.Wrappers, a.Wrappers},
		{"name comments", b.NameComments, a.NameComments},
		{"s.After sites", b.ScopeAfter, a.ScopeAfter},
		{"graph.After sites", b.GraphAfter, a.GraphAfter},
		{"layer tests code lines", tb.Code, ta.Code},
		{"layer tests raw lines", tb.Raw, ta.Raw},
	}
	fmt.Fprintf(w, "%s (%s): %s -> working tree\n", r.label, r.dir, baseline)
	fmt.Fprintf(w, "  %-24s %7s %7s %7s\n", "", "before", "after", "delta")
	for _, x := range rows {
		fmt.Fprintf(w, "  %-24s %7d %7d %+7d\n", x.label, x.before, x.after, x.after-x.before)
	}
}

// git runs git with args and returns its stdout, or an error carrying its
// stderr.
func git(args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("git %s: %w: %s", args[0], err, msg)
		}
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return out, nil
}
