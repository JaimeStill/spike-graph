package count

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	pathpkg "path"
	"regexp"
	"strconv"
	"strings"
)

// graphBase is the last element of the import path of the graph package
// whose shapes File counts: go-core/graph at the baseline, sdk/graph since.
// The local name a file imports it under is read from the file.
const graphBase = "graph"

// nameComment matches a comment whose text opens with a quoted name, the
// baseline's `// "logger"` beside each Nodes field.
var nameComment = regexp.MustCompile(`^"[^"]+"`)

// Counts are the measurements of one file, or the sum of several.
type Counts struct {
	// Code is the lines that are neither blank, comment-only, nor inside
	// an import declaration.
	Code int
	// Raw is every line, as wc -l counts them.
	Raw int
	// Closures is the function literals.
	Closures int
	// Wrappers is the func declarations, functions or methods, whose one
	// result is a func type taking a *graph.Scope first: each returns a
	// constructor rather than being one.
	Wrappers int
	// NameComments is the comments on fields of the struct type Nodes whose
	// text opens with a quoted name. A quoted string in any other comment
	// is not one.
	NameComments int
	// ScopeAfter is the After method calls on a *graph.Scope inside function
	// bodies, the baseline's s.After(n.X).
	ScopeAfter int
	// GraphAfter is the graph.After calls, ordering given as an option to
	// Define.
	GraphAfter int
}

// Add returns the sum of c and o.
func (c Counts) Add(o Counts) Counts {
	return Counts{
		Code:         c.Code + o.Code,
		Raw:          c.Raw + o.Raw,
		Closures:     c.Closures + o.Closures,
		Wrappers:     c.Wrappers + o.Wrappers,
		NameComments: c.NameComments + o.NameComments,
		ScopeAfter:   c.ScopeAfter + o.ScopeAfter,
		GraphAfter:   c.GraphAfter + o.GraphAfter,
	}
}

// File measures the Go source src; name is used only in parse errors. A
// nil src is a file absent at that side of the comparison and counts zero.
func File(name string, src []byte) (Counts, error) {
	if src == nil {
		return Counts{}, nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return Counts{}, fmt.Errorf("parse %s: %w", name, err)
	}
	c := lines(fset, f, src)
	graph := graphName(f)
	scopes := scopeParams(f, graph)
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			c.Closures++
		case *ast.FuncDecl:
			if returnsConstructor(x, graph) {
				c.Wrappers++
			}
			if x.Body != nil {
				c.ScopeAfter += scopeAfters(x.Body, scopes)
			}
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "After" && isIdent(sel.X, graph) {
				c.GraphAfter++
			}
		case *ast.TypeSpec:
			if st, ok := x.Type.(*ast.StructType); ok && x.Name.Name == "Nodes" {
				c.NameComments += nameComments(st)
			}
		}
		return true
	})
	return c, nil
}

// Lines measures only the code and raw lines of the Go source src, leaving
// the shape counts zero. A nil src, as for File, counts zero.
func Lines(name string, src []byte) (Counts, error) {
	if src == nil {
		return Counts{}, nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.ImportsOnly|parser.SkipObjectResolution)
	if err != nil {
		return Counts{}, fmt.Errorf("parse %s: %w", name, err)
	}
	return lines(fset, f, src), nil
}

// lines returns the code and raw lines of src, parsed as f. A line is code
// when a token other than a comment touches it, so a raw string's every
// line counts and a line holding only a comment does not; the lines of
// each import declaration are then taken back out.
func lines(fset *token.FileSet, f *ast.File, src []byte) Counts {
	code := map[int]bool{}
	var s scanner.Scanner
	file := token.NewFileSet().AddFile("", -1, len(src))
	s.Init(file, src, nil, 0)
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		// The scanner inserts a semicolon at a line's end; it is not code.
		if tok == token.SEMICOLON && lit == "\n" {
			continue
		}
		start := file.Line(pos)
		end := start + strings.Count(lit, "\n")
		for l := start; l <= end; l++ {
			code[l] = true
		}
	}
	for _, d := range f.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			for l := fset.Position(gd.Pos()).Line; l <= fset.Position(gd.End()).Line; l++ {
				delete(code, l)
			}
		}
	}
	raw := bytes.Count(src, []byte("\n"))
	if len(src) > 0 && src[len(src)-1] != '\n' {
		raw++
	}
	return Counts{Code: len(code), Raw: raw}
}

// graphName returns the local name f imports the graph package under, or
// "" when f does not import it.
func graphName(f *ast.File) string {
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || pathpkg.Base(path) != graphBase {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "graph"
	}
	return ""
}

// isScope reports whether expr is the type *graph.Scope, graph being the
// graph package's local name.
func isScope(expr ast.Expr, graph string) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Scope" && isIdent(sel.X, graph)
}

// isIdent reports whether expr is the identifier name; an empty name, the
// graph package not imported, matches nothing.
func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && name != "" && id.Name == name
}

// returnsConstructor reports whether fn's one result is a func type whose
// first parameter is a *graph.Scope.
func returnsConstructor(fn *ast.FuncDecl, graph string) bool {
	res := fn.Type.Results
	if res == nil || len(res.List) != 1 || len(res.List[0].Names) > 1 {
		return false
	}
	ft, ok := res.List[0].Type.(*ast.FuncType)
	if !ok || ft.Params == nil || len(ft.Params.List) == 0 {
		return false
	}
	return isScope(ft.Params.List[0].Type, graph)
}

// scopeParams returns the names of f's parameters, of function
// declarations and literals alike, whose type is *graph.Scope. The names
// are not tied to their functions: a composition root names its scope s
// throughout, and without type checking a name is the closest handle.
func scopeParams(f *ast.File, graph string) map[string]bool {
	names := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if ft, ok := n.(*ast.FuncType); ok && ft.Params != nil {
			for _, p := range ft.Params.List {
				if isScope(p.Type, graph) {
					for _, id := range p.Names {
						names[id.Name] = true
					}
				}
			}
		}
		return true
	})
	return names
}

// scopeAfters counts the After method calls in body whose receiver is one
// of scopes.
func scopeAfters(body *ast.BlockStmt, scopes map[string]bool) int {
	n := 0
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "After" {
			if id, ok := sel.X.(*ast.Ident); ok && scopes[id.Name] {
				n++
			}
		}
		return true
	})
	return n
}

// nameComments counts the comments, leading or trailing, on st's fields
// whose text opens with a quoted name.
func nameComments(st *ast.StructType) int {
	n := 0
	for _, field := range st.Fields.List {
		for _, group := range []*ast.CommentGroup{field.Doc, field.Comment} {
			if group == nil {
				continue
			}
			for _, c := range group.List {
				text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
				if nameComment.MatchString(text) {
					n++
				}
			}
		}
	}
	return n
}
