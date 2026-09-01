package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CLI-066 (#1364), Minor from the CLI-064 review: doctor and profile-heal.ps1
// each carry the two corruption thresholds as their own literal — Go counts
// bytes and marker occurrences, the script `-gt 1MB` and `[regex]::Matches
// ... -gt 1`. Nothing linked them, so one could move and the other would keep
// flagging (or clearing) a profile the heal disagrees about. This test reads
// the script the repo ships and pins its literals beside doctor's constants:
// change one, and this names the other.
func TestProfileHealThresholdsMatchTheScript(t *testing.T) {
	root := repoRootForDoctorTest(t)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", profileHealScript))
	if err != nil {
		t.Fatalf("the heal the doctor invokes must ship with the repo: %v", err)
	}
	script := string(raw)

	// Size: doctor's profileMaxBytes is exactly 1 MiB, the script's `-gt 1MB`
	// (PowerShell's 1MB is 1048576, the same power of two).
	if profileMaxBytes != 1<<20 {
		t.Fatalf("profileMaxBytes = %d; the script says `-gt 1MB` (1048576) — change both or neither", profileMaxBytes)
	}
	if !strings.Contains(script, "$state.Size -gt 1MB") {
		t.Fatalf("%s no longer tests `$state.Size -gt 1MB`; doctor's profileMaxBytes (%d) assumes it", profileHealScript, profileMaxBytes)
	}

	// Markers: doctor flags more than one START or END; the script the same,
	// on the same two strings.
	for _, want := range []string{
		"$state.StartMarkers -gt 1 -or $state.EndMarkers -gt 1",
		"'" + profileStartMarker + "'",
		"'" + profileEndMarker + "'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("%s no longer contains %q; doctor's marker rule (>1 of either, on these exact strings) assumes it", profileHealScript, want)
		}
	}

	// The parameter doctor passes exists, and the heal defaults to $PROFILE
	// without it — the contract --fix relies on (AC3/AC4).
	for _, want := range []string{
		"[string]$ProfilePath = ''",
		"$profilePath = if ($ProfilePath) { $ProfilePath } else { $PROFILE }",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("%s no longer carries %q; doctor invokes it with -ProfilePath", profileHealScript, want)
		}
	}
}
