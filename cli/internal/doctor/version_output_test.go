package doctor

import (
	"os/exec"
	"testing"
)

// doctor's version probes drop a failed run's stderr like `dotf tools`'
// (BUG-141): a crash leaves warnings there whose versions are not the tool's.
// Before, semverOf went through CommandOutput's merged stream.
func TestRealSystem_VersionOutputDropsAFailedRunsStderr(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not on PATH")
	}
	out, err := realSystem().VersionOutput("sh", "-c", "echo 'ansible-core 2.21.5 supports Python up to 3.14' >&2; exit 3")
	if err == nil || len(out) != 0 {
		t.Errorf("a crash read as output %q (err %v), want none", out, err)
	}
}

// A System without the VersionOutput seam answers from its CommandOutput fake:
// a nil runner would let ProbeVersion run the machine's real binary.
func TestSemverOf_WithoutTheSeamUsesCommandOutput(t *testing.T) {
	sys := &System{CommandOutput: func(name string, args ...string) (string, error) {
		return "fake-tool 9.8.7\n", nil
	}}
	if got := semverOf(sys, "fake-tool-that-is-not-installed"); got != "9.8.7" {
		t.Errorf("semverOf = %q, want the fake's 9.8.7", got)
	}
}
