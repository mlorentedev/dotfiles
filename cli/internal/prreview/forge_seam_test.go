package prreview

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Every forge call this package makes goes through GHRunner, the one seam a
// Gitea path has to answer (#1005). The call sites are a snapshot that ages, so
// this test holds the shape instead: a new `gh` subprocess, HTTP client or
// forge URL outside that seam fails here rather than waiting for a re-grep.
//
// It checks prreview's non-test source, so it cannot see a command that reaches
// a forge on its own. cmd/pr_*.go only passes a GHRunner in today.

// seamLiteralURLs are the URLs the package may name. None is a forge API: the
// first two are text inside PR-Agent's own marker, the rest are the model
// providers. A new one is a decision, so it is added here by hand.
var seamLiteralURLs = []string{
	"https://github.com/The-PR-Agent/pr-agent",
	"https://nan.builders/docs/models",
	"https://api.nan.builders/v1",
	"https://api.anthropic.com",
}

// seamSpecNames are the spec identifiers the package may use: the pool and
// nothing else, because spec also holds calls that run `gh`.
var seamSpecNames = []string{"PRAgentBlock", "ParseReviewerPool", "ReviewerPool"}

// seamModulePackages are the dotf packages the package may import.
var seamModulePackages = []string{"github.com/mlorentedev/dotfiles/cli/internal/spec"}

const seamModulePrefix = "github.com/mlorentedev/dotfiles/"

var seamURL = regexp.MustCompile(`https?://[^\s"'` + "`" + `)]+`)

// forgeSeamViolations lists what reaches past GHRunner in the parsed files.
func forgeSeamViolations(fset *token.FileSet, files []*ast.File) []string {
	var found []string
	report := func(pos token.Pos, format string, a ...any) {
		found = append(found, fset.Position(pos).String()+": "+fmt.Sprintf(format, a...))
	}
	for _, f := range files {
		for _, imp := range f.Imports {
			checkSeamImport(imp, report)
		}
		for _, decl := range f.Decls {
			fn, _ := decl.(*ast.FuncDecl)
			c := seamCheck{fn: fn, report: report, seamDefault: map[*ast.Ident]bool{}}
			ast.Inspect(decl, c.node)
		}
	}
	return found
}

type seamReport func(pos token.Pos, format string, a ...any)

func checkSeamImport(imp *ast.ImportSpec, report seamReport) {
	path, err := strconv.Unquote(imp.Path.Value)
	if err != nil {
		report(imp.Pos(), "unreadable import %s", imp.Path.Value)
		return
	}
	switch {
	case path == "net" || path == "syscall" || strings.HasPrefix(path, "net/rpc"):
		report(imp.Pos(), "imports %s, which can reach a forge outside GHRunner", path)
	case strings.HasPrefix(path, seamModulePrefix) && !slices.Contains(seamModulePackages, path):
		report(imp.Pos(), "imports %s: a forge call belongs behind GHRunner", path)
	}
}

// seamCheck walks one declaration. seamDefault holds the execGH identifiers
// that are the seam's own default, `env.GH = execGH`.
type seamCheck struct {
	fn          *ast.FuncDecl
	report      seamReport
	seamDefault map[*ast.Ident]bool
}

func (c seamCheck) inFunc(name string) bool { return c.fn != nil && c.fn.Name.Name == name }

func (c seamCheck) declares(id *ast.Ident) bool { return c.fn != nil && id == c.fn.Name }

func (c seamCheck) node(n ast.Node) bool {
	switch x := n.(type) {
	case *ast.BasicLit:
		checkSeamLiteral(x, c.report)
	case *ast.AssignStmt:
		c.markSeamDefault(x)
	case *ast.Ident:
		if x.Name == "execGH" && !c.seamDefault[x] && !c.declares(x) {
			c.report(x.Pos(), "execGH used directly: a call must go through a GHRunner a caller can replace")
		}
	case *ast.SelectorExpr:
		c.selector(x)
	}
	return true
}

func (c seamCheck) markSeamDefault(a *ast.AssignStmt) {
	if len(a.Lhs) != 1 || len(a.Rhs) != 1 {
		return
	}
	lhs, isField := a.Lhs[0].(*ast.SelectorExpr)
	rhs, isIdent := a.Rhs[0].(*ast.Ident)
	if isField && isIdent && lhs.Sel.Name == "GH" {
		c.seamDefault[rhs] = true
	}
}

func (c seamCheck) selector(x *ast.SelectorExpr) {
	pkg, isIdent := x.X.(*ast.Ident)
	if !isIdent {
		return
	}
	switch {
	case pkg.Name == "exec" && !c.inFunc("execGH"):
		c.report(x.Pos(), "exec.%s outside execGH: a subprocess can reach a forge past GHRunner", x.Sel.Name)
	case pkg.Name == "http" && httpReachesOut(x.Sel.Name) && !c.inFunc("post"):
		c.report(x.Pos(), "http.%s outside post, the model providers' one request", x.Sel.Name)
	case pkg.Name == "spec" && !slices.Contains(seamSpecNames, x.Sel.Name):
		c.report(x.Pos(), "spec.%s: spec holds calls that run gh, so only the pool is read from it", x.Sel.Name)
	}
}

// httpReachesOut is whether the net/http name starts a request.
func httpReachesOut(name string) bool {
	return slices.Contains([]string{"Get", "Head", "Post", "PostForm", "NewRequest", "NewRequestWithContext"}, name)
}

func checkSeamLiteral(lit *ast.BasicLit, report seamReport) {
	if lit.Kind != token.STRING {
		return
	}
	for _, u := range seamURL.FindAllString(lit.Value, -1) {
		if !slices.Contains(seamLiteralURLs, u) {
			report(lit.Pos(), "names %s: a forge URL belongs behind GHRunner", u)
		}
	}
}

func parseSeamSource(t *testing.T, src string) (*token.FileSet, []*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "planted.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	return fset, []*ast.File{f}
}

func TestForgeCallsStayBehindGHRunner(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset, files := token.NewFileSet(), []*ast.File{}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		t.Fatal("no source files read, so this would pass on nothing")
	}
	for _, v := range forgeSeamViolations(fset, files) {
		t.Error(v)
	}
}

// A check that cannot fail proves nothing, so each kind of reach is planted.
func TestForgeSeamCheckCatchesEachKindOfReach(t *testing.T) {
	cases := map[string]struct{ src, want string }{
		"a second gh subprocess": {`package p
import "os/exec"
func readPR() { _ = exec.Command("gh", "api", "repos/o/r/pulls/1") }`, "exec.Command outside execGH"},
		"a request outside post": {`package p
import "net/http"
func readPR() { _, _ = http.Get(base) }`, "http.Get outside post"},
		"a github API URL": {`package p
const base = "https://api.github.com/repos"`, "names https://api.github.com/repos"},
		"a gitea API URL": {`package p
func f() string { return "https://forge.example/api/v1/repos" }`, "names https://forge.example/api/v1/repos"},
		"the forge package": {`package p
import _ "github.com/mlorentedev/dotfiles/cli/internal/forge"`, "imports github.com/mlorentedev/dotfiles/cli/internal/forge"},
		"execGH called past the seam": {`package p
func readPR() { _, _, _ = execGH(ctx, "api", "repos/o/r/pulls/1") }`, "execGH used directly"},
		"a raw socket": {`package p
import _ "net"`, "imports net"},
		"a spec call that runs gh": {`package p
func f() { spec.IssueOpen(1) }`, "spec.IssueOpen"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			fset, files := parseSeamSource(t, c.src)
			got := strings.Join(forgeSeamViolations(fset, files), "\n")
			if !strings.Contains(got, c.want) {
				t.Errorf("want %q in the violations, got %q", c.want, got)
			}
		})
	}
}

// The places the rules name must stay allowed, or the rule
// would be satisfied only by moving the call somewhere it no longer applies.
func TestForgeSeamCheckAllowsTheSeamItself(t *testing.T) {
	fset, files := parseSeamSource(t, `package p
import ("net/http"; "os/exec")
func execGH() { _ = exec.CommandContext(ctx, "gh") }
func post() { _, _ = http.NewRequestWithContext(ctx, "POST", u, nil) }
func defaults(env Env) { env.GH = execGH }`)
	if v := forgeSeamViolations(fset, files); len(v) != 0 {
		t.Errorf("the seam was reported: %v", v)
	}
}
