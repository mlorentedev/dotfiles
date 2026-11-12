package worktree

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

// funcLines is the line span of every top-level function in file, by name,
// so a test can pin the limit AGENTS.md sets and nothing else enforces.
func funcLines(t *testing.T, file string) map[string]int {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	out := map[string]int{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
			out[fn.Name.Name] = fset.Position(fn.End()).Line - fset.Position(fn.Pos()).Line + 1
		}
	}
	return out
}

// packageFile resolves a sibling source file of the calling test, so the
// guard reads the tree under test and not the working directory.
func packageFile(t *testing.T, name string) string {
	t.Helper()
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unknown")
	}
	return filepath.Join(filepath.Dir(self), name)
}

// The park path grew past the under-40-line limit three times before a review
// noticed (#3265), and the sweep loop once more (#3360); this pins both
// extractions so they do not grow back.
func TestParkPathFunctionsStayUnderTheLimit(t *testing.T) {
	const limit = 40
	for file, names := range map[string][]string{
		"done.go":  {"Park"},
		"sweep.go": {"parkSingleNamedLane", "SweepWithRunner", "sweepOne", "sweepLane"},
	} {
		lines := funcLines(t, packageFile(t, file))
		for _, name := range names {
			if n, ok := lines[name]; !ok {
				t.Errorf("%s: %s not found", file, name)
			} else if n >= limit {
				t.Errorf("%s: %s is %d lines, not under the %d-line limit", file, name, n, limit)
			}
		}
	}
}
