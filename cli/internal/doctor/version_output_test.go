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
