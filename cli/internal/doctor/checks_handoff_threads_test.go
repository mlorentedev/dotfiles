package doctor

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

// handoffFixture is a real repository named "proj" with linked worktrees, and a
// vault holding its project directory. Real git, not a canned listing: the
// check keys each worktree with mem.ThreadKey, which reads git's own on-disk
// state, so a fake listing over fake paths would test nothing.
type handoffFixture struct {
	sys     *System
	repo    string
	memDir  string
	vault   string
	trees   map[string]string // branch -> worktree path
	threads string
}

func handoffGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir,
		"-c", "user.email=t@example.invalid", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func newHandoffFixture(t *testing.T, branches ...string) *handoffFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &handoffFixture{
		repo:  filepath.Join(root, "proj"),
		vault: filepath.Join(root, "vault"),
		trees: map[string]string{},
	}
	mkdirAll(t, f.repo)
	handoffGit(t, f.repo, "init", "-q", "-b", "main")
	handoffGit(t, f.repo, "commit", "-q", "--allow-empty", "-m", "init")
	for _, b := range branches {
		path := filepath.Join(root, "wt-"+strings.ReplaceAll(b, "/", "-"))
		handoffGit(t, f.repo, "worktree", "add", "-q", "-b", b, path)
		f.trees[b] = path
	}
	f.memDir = filepath.Join(f.vault, "10_projects", "proj", "memory")
	mkdirAll(t, f.memDir)
	f.threads = filepath.Join(f.memDir, "threads")
	f.sys = newSys(map[string]string{"HOME": root, "VAULT_PATH": f.vault}, nil, nil)
	f.sys.CommandOutputDir = func(dir, name string, args ...string) (string, error) {
		out, err := exec.Command(name, append([]string{"-C", dir}, args...)...).Output()
		return string(out), err
	}
	return f
}

func (f *handoffFixture) writeMemory(t *testing.T, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.memDir, "MEMORY.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeThread stores a thread file for key, as StoreThreadAs would name it.
func (f *handoffFixture) writeThread(t *testing.T, dir, key string) string {
	t.Helper()
	name, err := mem.ThreadFileName(key)
	if err != nil {
		t.Fatal(err)
	}
	mkdirAll(t, dir)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("**Next action:** x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func (f *handoffFixture) run(t *testing.T) (string, *Report) {
	t.Helper()
	var buf bytes.Buffer
	rep := capture(&buf)
	checkHandoffThreads(f.sys, f.repo, rep)
	return buf.String(), rep
}

// AC1: file present, no entry → WARN naming key and file
func TestHandoffThreads_WarnsOnALostEntry(t *testing.T) {
	f := newHandoffFixture(t, "feat/a")
	file := f.writeThread(t, f.threads, "feat-a")
	f.writeMemory(t, "# proj\n\n### thread: other\n")
	out, rep := f.run(t)
	if !strings.Contains(out, `thread "feat-a"`) || !strings.Contains(out, file) {
		t.Errorf("expected a WARN naming feat-a and %s, got\n%s", file, out)
	}
	if rep.Failures() != 0 {
		t.Errorf("must never FAIL, got %d failures\n%s", rep.Failures(), out)
	}
}

// AC2: entry present, or a fork entry → no warning
func TestHandoffThreads_EntryOrForkPasses(t *testing.T) {
	for name, mem := range map[string]string{
		"own":  "### thread: feat-a\n",
		"fork": "### thread: feat-a+copilot\n",
	} {
		f := newHandoffFixture(t, "feat/a")
		f.writeThread(t, f.threads, "feat-a")
		f.writeMemory(t, "# proj\n\n"+mem)
		out, _ := f.run(t)
		if strings.Contains(out, "no `### thread") || !strings.Contains(out, "keeps its entry") {
			t.Errorf("%s: expected PASS, got\n%s", name, out)
		}
	}
}

// AC3: file moved to archive/ → no warning
func TestHandoffThreads_ArchivedThreadIsQuiet(t *testing.T) {
	f := newHandoffFixture(t, "feat/a")
	mkdirAll(t, f.threads)
	f.writeThread(t, filepath.Join(f.memDir, "archive"), "feat-a")
	f.writeMemory(t, "# proj\n")
	out, _ := f.run(t)
	if strings.Contains(out, "no `### thread") {
		t.Errorf("an archived thread must not warn, got\n%s", out)
	}
}

// AC4: no thread file but a journal under sessions/ → no warning
func TestHandoffThreads_JournalAloneIsNotProof(t *testing.T) {
	f := newHandoffFixture(t, "feat/a")
	mkdirAll(t, f.threads)
	sessions := filepath.Join(f.vault, "10_projects", "proj", "sessions")
	mkdirAll(t, sessions)
	if err := os.WriteFile(filepath.Join(sessions, "2026-11-01-proj-claude-feat-a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.writeMemory(t, "# proj\n")
	out, _ := f.run(t)
	if strings.Contains(out, "no `### thread") {
		t.Errorf("a journal is not proof of a handoff, got\n%s", out)
	}
}

// AC5: skips say why
func TestHandoffThreads_Skips(t *testing.T) {
	f := newHandoffFixture(t, "feat/a")
	f.sys.Getenv = func(string) string { return "" }
	f.sys.ResolvePath = func(string) string { return "" }
	if out, _ := f.run(t); !strings.Contains(out, "no vault resolves") {
		t.Errorf("no vault: want SKIP, got\n%s", out)
	}

	f = newHandoffFixture(t, "feat/a")
	if err := os.RemoveAll(filepath.Join(f.vault, "10_projects")); err != nil {
		t.Fatal(err)
	}
	if out, _ := f.run(t); !strings.Contains(out, "no vault project") {
		t.Errorf("no vault project: want SKIP, got\n%s", out)
	}

	f = newHandoffFixture(t, "feat/a")
	if out, _ := f.run(t); !strings.Contains(out, "no MEMORY.md") {
		t.Errorf("no MEMORY.md: want SKIP, got\n%s", out)
	}

	f = newHandoffFixture(t, "feat/a")
	f.writeMemory(t, "# proj\n")
	if out, _ := f.run(t); !strings.Contains(out, "inline") || !strings.Contains(out, "does not cover") {
		t.Errorf("no threads/: want SKIP naming the inline layout as not covered, got\n%s", out)
	}
}

// AC6: the vault is byte-identical after the check and it cannot FAIL
func TestHandoffThreads_NeverWritesNeverFails(t *testing.T) {
	f := newHandoffFixture(t, "feat/a")
	f.writeThread(t, f.threads, "feat-a")
	f.writeMemory(t, "# proj\n")
	before := snapshotTree(t, f.vault)
	_, rep := f.run(t)
	if after := snapshotTree(t, f.vault); after != before {
		t.Errorf("the check changed the vault:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if rep.Failures() != 0 {
		t.Errorf("must never FAIL, got %d", rep.Failures())
	}
}

// AC7: keys come from mem.ThreadKey for a slash branch and a detached worktree
func TestHandoffThreads_KeysLikeTheWriter(t *testing.T) {
	f := newHandoffFixture(t, "fix/deep/x")
	detached := filepath.Join(filepath.Dir(f.repo), "wt-detached")
	handoffGit(t, f.repo, "worktree", "add", "-q", "--detach", detached)
	f.writeMemory(t, "# proj\n")
	slashKey := mem.ThreadKey(f.trees["fix/deep/x"])
	detKey := mem.ThreadKey(detached)
	if !strings.Contains(detKey, "@") {
		t.Fatalf("a detached worktree must key as <name>@<host>, got %q", detKey)
	}
	f.writeThread(t, f.threads, slashKey)
	f.writeThread(t, f.threads, detKey)
	out, _ := f.run(t)
	for _, k := range []string{slashKey, detKey} {
		if !strings.Contains(out, `thread "`+k+`"`) {
			t.Errorf("want a WARN for the key mem.ThreadKey gives (%s), got\n%s", k, out)
		}
	}
}

// snapshotTree lists every path under root with its size and content hash, so a
// write of any kind shows as a difference.
func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		b.WriteString(rel)
		if info.Mode().IsRegular() {
			data, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			b.WriteString(" " + string(data))
		}
		b.WriteString("\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
