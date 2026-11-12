package lane

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

// The switch path grew past the 40-line limit before a review noticed
// (#3265); this pins the extraction so it does not grow back.
func TestSwitchLaneLockedStaysUnderTheLimit(t *testing.T) {
	const limit = 40
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unknown")
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(filepath.Dir(self), "named.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "switchLaneLocked" {
			continue
		}
		if n := fset.Position(fn.End()).Line - fset.Position(fn.Pos()).Line + 1; n > limit {
			t.Errorf("switchLaneLocked is %d lines, over the %d-line limit", n, limit)
		}
		return
	}
	t.Fatal("switchLaneLocked not found in named.go")
}
