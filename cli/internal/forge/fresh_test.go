package forge

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func writeDecl(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "forge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, DeclarationFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// clonePair returns a checkout of a bare origin whose main holds decl, and a
// second checkout of the same origin to push from: the other session that
// lands a declaration change while the first one does not pull.
func clonePair(t *testing.T, decl string) (work, other string) {
	t.Helper()
	base := t.TempDir()
	origin := filepath.Join(base, "origin.git")
	work, other = filepath.Join(base, "work"), filepath.Join(base, "other")
	git(t, base, "init", "-q", "--bare", "-b", "main", origin)
	git(t, base, "init", "-q", "-b", "main", work)
	writeDecl(t, work, decl)
	git(t, work, "add", DeclarationFile)
	git(t, work, "commit", "-q", "-m", "declare")
	git(t, work, "remote", "add", "origin", origin)
	git(t, work, "push", "-q", "-u", "origin", "main")
	git(t, base, "clone", "-q", origin, other)
	return work, other
}

func TestCheckFresh(t *testing.T) {
	t.Run("a checkout whose declaration is main's is fresh", func(t *testing.T) {
		work, _ := clonePair(t, `{"strict":false}`)
		if err := CheckFresh(work); err != nil {
			t.Fatalf("want fresh, got %v", err)
		}
	})

	t.Run("a checkout behind a landed declaration change is stale", func(t *testing.T) {
		work, other := clonePair(t, `{"strict":true}`)
		writeDecl(t, other, `{"strict":false}`)
		git(t, other, "commit", "-q", "-am", "relax")
		git(t, other, "push", "-q", "origin", "main")

		err := CheckFresh(work)
		if err == nil || !strings.Contains(err.Error(), "origin/main") {
			t.Fatalf("a stale declaration must be refused naming the ref it differs from, got %v", err)
		}
	})

	t.Run("a checkout behind on other files only is fresh", func(t *testing.T) {
		work, other := clonePair(t, `{"strict":false}`)
		if err := os.WriteFile(filepath.Join(other, "README"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		git(t, other, "add", "README")
		git(t, other, "commit", "-q", "-m", "unrelated")
		git(t, other, "push", "-q", "origin", "main")

		if err := CheckFresh(work); err != nil {
			t.Fatalf("only the declaration's content matters, got %v", err)
		}
	})

	t.Run("an unmerged edit of the declaration is refused", func(t *testing.T) {
		work, _ := clonePair(t, `{"strict":false}`)
		writeDecl(t, work, `{"strict":true}`)
		if err := CheckFresh(work); err == nil {
			t.Fatal("a declaration main does not hold must be refused")
		}
	})

	t.Run("a declaration absent from main is refused", func(t *testing.T) {
		work, other := clonePair(t, `{"strict":false}`)
		git(t, other, "rm", "-q", DeclarationFile)
		git(t, other, "commit", "-q", "-m", "drop")
		git(t, other, "push", "-q", "origin", "main")
		if err := CheckFresh(work); err == nil {
			t.Fatal("a declaration main no longer holds must be refused")
		}
	})

	t.Run("a local main is not origin's default branch", func(t *testing.T) {
		// origin is reachable, holds only dev, and its HEAD names trunk, which
		// does not exist, so no fetch can record an origin/HEAD. A fallback to
		// the local main would compare the file with itself and answer fresh
		// forever: the false OK this check exists to stop.
		base := t.TempDir()
		origin, work := filepath.Join(base, "origin.git"), filepath.Join(base, "work")
		git(t, base, "init", "-q", "--bare", "-b", "trunk", origin)
		git(t, base, "init", "-q", "-b", "main", work)
		writeDecl(t, work, `{"strict":false}`)
		git(t, work, "add", DeclarationFile)
		git(t, work, "commit", "-q", "-m", "declare")
		git(t, work, "remote", "add", "origin", origin)
		git(t, work, "push", "-q", "origin", "main:dev")
		if err := CheckFresh(work); err == nil {
			t.Fatal("with no origin default branch the freshness is unanswerable, and unanswerable is a refusal")
		}
	})

	t.Run("an unreachable origin is a refusal", func(t *testing.T) {
		work := t.TempDir()
		git(t, work, "init", "-q", "-b", "main")
		writeDecl(t, work, `{"strict":false}`)
		if err := CheckFresh(work); err == nil || !strings.Contains(err.Error(), "fetching origin failed") {
			t.Fatalf("a failed fetch must refuse, got %v", err)
		}
	})

	t.Run("origin/HEAD names the default branch when recorded", func(t *testing.T) {
		work, other := clonePair(t, `{"strict":false}`)
		git(t, other, "checkout", "-q", "-b", "trunk")
		writeDecl(t, other, `{"strict":true}`)
		git(t, other, "commit", "-q", "-am", "trunk differs")
		git(t, other, "push", "-q", "origin", "trunk")
		git(t, work, "fetch", "-q", "origin")
		git(t, work, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")

		err := CheckFresh(work)
		if err == nil || !strings.Contains(err.Error(), "origin/trunk") {
			t.Fatalf("origin/HEAD must win over origin/main, got %v", err)
		}
	})
}
