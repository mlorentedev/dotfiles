package worktree

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A GIT_DIR inherited from a hook or a parent git outranks -C, so without the
// scrub the runner lists the other repository's worktrees (#3122).
func TestRealGitRunner_ListsTheNamedRepositoryWhateverGitDirSays(t *testing.T) {
	tmp := t.TempDir()
	mine, other := filepath.Join(tmp, "mine"), filepath.Join(tmp, "other")
	for _, repo := range []string{mine, other} {
		gitT(t, tmp, "init", "-q", "-b", "main", repo)
		gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	}
	gitT(t, other, "worktree", "add", "-q", "-b", "elsewhere", filepath.Join(tmp, "other-wt"))
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)

	got, err := (&RealGitRunner{}).WorktreeListPorcelain(mine)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "elsewhere") || strings.Contains(got, "other") {
		t.Errorf("listed another repository's worktrees:\n%s", got)
	}
	if !strings.Contains(got, "branch refs/heads/main") {
		t.Errorf("did not list the named repository:\n%s", got)
	}
}

// The fetch sets one variable of its own; adding it to os.Environ() instead of
// gitIn's environment put GIT_DIR back on the one call that reaches the network.
func TestFetchCommitCmd_KeepsTheScrub(t *testing.T) {
	t.Setenv("GIT_DIR", "/elsewhere/.git")
	env := strings.Join(fetchCommitCmd(t.TempDir(), "abc").Env, "\n")
	if strings.Contains(env, "GIT_DIR=") {
		t.Errorf("fetch inherits GIT_DIR:\n%s", env)
	}
	if !strings.Contains(env, "GIT_TERMINAL_PROMPT=0") {
		t.Errorf("fetch lost GIT_TERMINAL_PROMPT=0")
	}
}

// resolveBaseRef is called by the runner's merge check; its ref lookup must read
// the named repository too.
func TestFirstRef_ReadsTheNamedRepositoryWhateverGitDirSays(t *testing.T) {
	tmp := t.TempDir()
	mine, other := filepath.Join(tmp, "mine"), filepath.Join(tmp, "other")
	gitT(t, tmp, "init", "-q", "-b", "main", mine)
	gitT(t, tmp, "init", "-q", "-b", "trunk", other)
	for _, repo := range []string{mine, other} {
		gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	}
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	if got := firstRef(mine, []string{"main"}); got != "main" {
		t.Errorf("firstRef = %q, want main from the named repository", got)
	}
}

// The destructive calls are the ones that matter: with GIT_DIR naming another
// repository, branch -D deleted that repository's branch of the same name (#3275).
func TestRealSweepRunner_DeletesTheBranchOfTheNamedRepositoryWhateverGitDirSays(t *testing.T) {
	tmp := t.TempDir()
	mine, other := filepath.Join(tmp, "mine"), filepath.Join(tmp, "other")
	for _, repo := range []string{mine, other} {
		gitT(t, tmp, "init", "-q", "-b", "main", repo)
		gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
		gitT(t, repo, "branch", "doomed")
	}
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))

	if _, err := (&RealSweepRunner{}).BranchDelete(mine, "doomed"); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("GIT_DIR"); err != nil { // t.Setenv above restores it
		t.Fatal(err)
	}
	if out := gitT(t, other, "branch", "--list", "doomed"); !strings.Contains(out, "doomed") {
		t.Errorf("deleted the branch of the repository GIT_DIR named")
	}
	if out := gitT(t, mine, "branch", "--list", "doomed"); strings.Contains(out, "doomed") {
		t.Errorf("left the named repository's branch in place")
	}
}

// Every git call in this package goes through gitIn, and nothing rebuilds an
// environment from os.Environ() but gitIn: one call that does lets an inherited
// GIT_DIR point it at another repository (#3122, #3275).
func TestPackage_RunsGitOnlyThroughGitIn(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Name.Name == "gitIn" {
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch name := callee(call); name {
				case "exec.Command", "exec.CommandContext":
					prog := call.Args[0]
					if name == "exec.CommandContext" {
						prog = call.Args[1]
					}
					// A program named by a variable could be git: only a literal proves it is not.
					if lit, ok := prog.(*ast.BasicLit); !ok || lit.Value == `"git"` {
						t.Errorf("%s: %s runs git, or a program this test cannot name, outside gitIn", fset.Position(call.Pos()), name)
					}
				case "os.Environ":
					t.Errorf("%s: os.Environ() outside gitIn", fset.Position(call.Pos()))
				}
				return true
			})
		}
	}
}

func callee(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return pkg.Name + "." + sel.Sel.Name
}
