package spec

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// printEnvChild marks the test binary re-run as a runner: it prints its
// environment and exits. The re-run is the child, so the test runs on Windows,
// where #3615 was measured, and not only where `sh` exists.
const printEnvChild = "DOTF_SPEC_TEST_PRINT_ENV"

func TestHelperRunnerPrintsItsEnvironment(t *testing.T) {
	if os.Getenv(printEnvChild) != "1" {
		t.Skip("helper: runs only as the child of TestRunWithLimitsDropsBunInspectorVariables")
	}
	for _, entry := range os.Environ() {
		fmt.Println(entry)
	}
	os.Exit(0)
}

// Claude Code runs its own Bun inspector and exports BUN_INSPECT to the shells
// it starts. A Bun-based runner (pi) that inherits it tries to bind the same
// port and exits before reviewing (#3615). Every runner goes through
// RunWithLimits, so the variables are dropped there; the rest of the
// environment, which carries the runner's credentials and PATH, still arrives.
func TestRunWithLimitsDropsBunInspectorVariables(t *testing.T) {
	t.Setenv("BUN_INSPECT", "ws://127.0.0.1:9221/inspector")
	t.Setenv("BUN_INSPECT_CONNECT_TO", "ws://127.0.0.1:9221/connect")
	t.Setenv("Bun_Inspect_Notify", "unix:///tmp/notify")
	t.Setenv("DOTF_SPEC_TEST_KEEP", "kept")
	t.Setenv("BUN_INSPECTOR", "kept") // shares the prefix, not the name: it must arrive
	t.Setenv(printEnvChild, "1")

	var out, errOut bytes.Buffer
	argv := []string{os.Args[0], "-test.run=^TestHelperRunnerPrintsItsEnvironment$"}
	res, err := RunWithLimits(Limits{Deadline: time.Minute}, argv, nil, &out, &errOut)
	if err != nil || res.Code != 0 {
		t.Fatalf("runner: code %d, err %v, stderr %q", res.Code, err, errOut.String())
	}

	kept := map[string]bool{"DOTF_SPEC_TEST_KEEP=kept": false, "BUN_INSPECTOR=kept": false}
	for _, line := range strings.Split(strings.ReplaceAll(out.String(), "\r\n", "\n"), "\n") {
		key, _, _ := strings.Cut(line, "=")
		upper := strings.ToUpper(key)
		if upper == "BUN_INSPECT" || strings.HasPrefix(upper, "BUN_INSPECT_") {
			t.Errorf("the runner inherited %s", line)
		}
		if _, ok := kept[line]; ok {
			kept[line] = true
		}
	}
	for entry, arrived := range kept {
		if !arrived {
			t.Errorf("the runner lost %s; its environment was:\n%s", entry, out.String())
		}
	}
}
