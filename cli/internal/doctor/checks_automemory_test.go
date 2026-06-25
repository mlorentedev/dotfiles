package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

// autoMemEnv builds a HOME+VAULT_PATH env and a vault holding a memory source for
// project "myproj", returning the System and the start dir whose base is myproj.
func autoMemEnv(t *testing.T) (sys *System, start, vault string) {
	t.Helper()
	home := t.TempDir()
	vault = t.TempDir()
	mkdirAll(t, filepath.Join(vault, "10_projects", "myproj", "memory"))
	start = "/work/myproj" // filepath.Base → "myproj"; resolves convention 1
	sys = newSys(map[string]string{"HOME": home, "VAULT_PATH": vault}, nil, nil)
	return sys, start, vault
}

func TestCheckAutoMemoryLink(t *testing.T) {
	t.Run("no vault source → SKIP", func(t *testing.T) {
		home := t.TempDir()
		vault := t.TempDir()
		sys := newSys(map[string]string{"HOME": home, "VAULT_PATH": vault}, nil, nil)
		var buf bytes.Buffer
		rep := capture(&buf)
		checkAutoMemoryLink(sys, "/work/orphan", rep, false)
		if rep.Failures() != 0 || !strings.Contains(buf.String(), "nothing to link") {
			t.Errorf("expected SKIP, got\n%s", buf.String())
		}
	})

	t.Run("source exists, link missing, no --fix → FAIL", func(t *testing.T) {
		sys, start, _ := autoMemEnv(t)
		var buf bytes.Buffer
		rep := capture(&buf)
		checkAutoMemoryLink(sys, start, rep, false)
		if rep.Failures() != 1 || !strings.Contains(buf.String(), "dotf doctor --fix") {
			t.Errorf("expected FAIL with a --fix hint, got\n%s", buf.String())
		}
	})

	// NB: subtest names must avoid commas/parens — they leak into t.TempDir() paths,
	// and a bare comma in a path breaks `cmd /c mklink` arg parsing on Windows.
	t.Run("fix links it then a re-run is idempotent", func(t *testing.T) {
		sys, start, _ := autoMemEnv(t)

		var fixBuf bytes.Buffer
		repFix := capture(&fixBuf)
		checkAutoMemoryLink(sys, start, repFix, true)
		if !strings.Contains(fixBuf.String(), "linked auto-memory to the vault") {
			t.Fatalf("--fix should create the link, got\n%s", fixBuf.String())
		}

		var reBuf bytes.Buffer
		repRe := capture(&reBuf)
		checkAutoMemoryLink(sys, start, repRe, true)
		if repRe.Failures() != 0 || !strings.Contains(reBuf.String(), "auto-memory linked to the vault") {
			t.Errorf("re-run should be an idempotent PASS, got\n%s", reBuf.String())
		}
	})

	t.Run("real non-empty dir → WARN, never destroyed by --fix", func(t *testing.T) {
		sys, start, _ := autoMemEnv(t)
		target := memlink.ClaudeMemoryTarget(sys.home(), start)
		ownFile := filepath.Join(target, "MEMORY.md")
		writeFile(t, ownFile, "local agent data")

		var buf bytes.Buffer
		rep := capture(&buf)
		checkAutoMemoryLink(sys, start, rep, true) // even with --fix
		out := buf.String()
		if rep.Failures() != 0 {
			t.Errorf("a real data dir must WARN, not FAIL\n%s", out)
		}
		if !strings.Contains(out, "real directory") || !strings.Contains(out, "Reconcile") {
			t.Errorf("expected the manual-reconcile WARN, got\n%s", out)
		}
		if !pathExists(ownFile) {
			t.Error("--fix must NOT destroy the agent's own data dir")
		}
	})
}
