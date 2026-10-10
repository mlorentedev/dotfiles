//go:build !windows

package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The age root restored with `cp` keeps the copy's mode, 0644 under the default
// umask: every account on the machine can read the key that decrypts every age
// secret. `dotf secrets verify` failed on it while doctor passed it as "on
// disk", because doctor asked only whether the file existed (#2203). Unix only:
// Windows keeps no permission bits for the declaration to describe.
func ageRootFixture(t *testing.T, mode os.FileMode) (*System, *Config, string) {
	t.Helper()
	dotfiles := t.TempDir()
	home := t.TempDir()
	key := filepath.Join(home, ".config", "age", "key.txt")
	writeFile(t, key, "AGE-SECRET-KEY-1...")
	if err := os.Chmod(key, mode); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dotfiles, "secrets", "registry.yaml"),
		"version: 1\nsecrets:\n"+
			"  - id: age-root\n"+
			"    plane: root\n"+
			"    backend: file-authority\n"+
			"    recipient: age1test\n"+
			"    expose:\n"+
			"      file:\n"+
			"        var: AGE_KEY_PERSONAL\n"+
			"        path: ~/.config/age/key.txt\n"+
			"        mode: \"0600\"\n")
	return newSys(map[string]string{"HOME": home}, nil, nil), &Config{DotfilesDir: dotfiles}, key
}

func TestCheckSecrets_AFileAuthorityRootReadableByOthersFailsNamingTheMode(t *testing.T) {
	sys, cfg, key := ageRootFixture(t, 0o644)
	var buf bytes.Buffer
	rep := capture(&buf)
	checkSecrets(sys, cfg, rep, false)
	out := buf.String()
	if rep.Failures() != 1 || !strings.Contains(out, "0644") || !strings.Contains(out, "0600") || !strings.Contains(out, "dotf doctor --fix") {
		t.Errorf("want one FAIL naming the mode, the declared mode and the fix\n%s", out)
	}
	if fi, _ := os.Stat(key); fi.Mode().Perm() != 0o644 {
		t.Errorf("without --fix the mode must not change, got %04o", fi.Mode().Perm())
	}
}

func TestCheckSecrets_FixSetsTheDeclaredModeOnAFileAuthorityRoot(t *testing.T) {
	sys, cfg, key := ageRootFixture(t, 0o644)
	var buf bytes.Buffer
	rep := capture(&buf)
	checkSecrets(sys, cfg, rep, true)
	if rep.Failures() != 0 || !strings.Contains(buf.String(), "set to the declared mode") {
		t.Errorf("a repaired mode is reported as a repair, not a failure\n%s", buf.String())
	}
	if fi, _ := os.Stat(key); fi.Mode().Perm() != 0o600 {
		t.Errorf("--fix must leave the key at 0600, got %04o", fi.Mode().Perm())
	}
	// Idempotent: a second run has nothing to repair. Asserted on the repair
	// message, not on the digits "0644": the report prints the key's temp path,
	// and macOS's random temp names contained them (…Root2064490071).
	buf.Reset()
	rep = capture(&buf)
	checkSecrets(sys, cfg, rep, true)
	if rep.Failures() != 0 || strings.Contains(buf.String(), "set to the declared mode") {
		t.Errorf("second run must be clean\n%s", buf.String())
	}
}
