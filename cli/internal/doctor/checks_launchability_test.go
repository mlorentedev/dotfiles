package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// Launchability (WIN-012/#1293): the shell wrappers resolve the agents' keys
// through `dotf secrets run` before the binary starts, so a locked vault with
// no unlocked daemon means the agent never launches — while every file and
// PATH predicate in the section stays green. WARN with the literal remedy;
// PASS when the daemon is unlocked; silent while no key is bw-backed, because
// the age floor needs no daemon.
func TestCheckOpenCode_ReportsWrapperLaunchability(t *testing.T) {
	cfg := &Config{Versions: map[string]string{"PI_VERSION": "0.79.1", "OPENCODE_VERSION": "1.0.0"}}
	run := func(t *testing.T, bwBacked int, status string) string {
		t.Helper()
		home := t.TempDir()
		writeFile(t, filepath.Join(home, ".pi", "agent", "models.json"), `{"models":{}}`)
		sys := newSys(map[string]string{"HOME": home}, []string{"pi"}, map[string]string{"pi --version": "pi 0.79.1"})
		sys.BWBackedSecrets = func() (int, error) { return bwBacked, nil }
		sys.BWServeStatus = func() (string, error) { return status, nil }
		var buf bytes.Buffer
		checkOpenCode(sys, cfg, capture(&buf))
		return buf.String()
	}

	if out := run(t, 28, "absent"); !strings.Contains(out, "[WARN]") || !strings.Contains(out, "dotf secrets unlock") {
		t.Errorf("bw-backed keys and no daemon: must WARN with the remedy\n%s", out)
	}
	if out := run(t, 28, "locked"); !strings.Contains(out, "dotf secrets unlock") {
		t.Errorf("a locked daemon is the same outage\n%s", out)
	}
	if out := run(t, 28, "unlocked"); !strings.Contains(out, "agent wrappers can resolve their keys") {
		t.Errorf("an unlocked daemon must PASS\n%s", out)
	}
	if out := run(t, 0, "absent"); strings.Contains(out, "wrappers") {
		t.Errorf("no bw-backed key: the age floor needs no daemon, so stay silent\n%s", out)
	}
}
