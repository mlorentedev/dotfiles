package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// colimaYAML is the shape colima writes on first start: comments around every
// key, which --fix must keep.
func colimaYAML(cpu, mem string) string {
	return "# Number of CPUs to be allocated to the virtual machine.\n# Default: 2\ncpu: " + cpu +
		"\n\n# Size of the disk in GiB to be allocated to the virtual machine.\ndisk: 100\n\n" +
		"# Size of the memory in GiB to be allocated to the virtual machine.\n# Default: 2\nmemory: " + mem +
		"\n\nruntime: docker\n"
}

const (
	vmRunning44 = `{"name":"default","status":"Running","arch":"aarch64","cpus":4,"memory":8589934592,"disk":107374182400,"runtime":"docker"}`
	vmRunning22 = `{"name":"default","status":"Running","arch":"aarch64","cpus":2,"memory":2147483648,"disk":107374182400,"runtime":"docker"}`
	vmStopped22 = `{"name":"default","status":"Stopped","arch":"aarch64","cpus":2,"memory":2147483648,"disk":107374182400,"runtime":"docker"}`
)

// The repo declares Colima's VM size (owner, #2013 P5b). colima.yaml is
// colima's own file, so doctor reads it and the running VM, and --fix edits the
// two keys and restarts the Homebrew service, announcing that containers stop.
func TestCheckColimaSize(t *testing.T) {
	engineProbeInterval = 0
	t.Cleanup(func() { engineProbeInterval = 3 * time.Second })

	cases := []struct {
		name        string
		goos        string
		onPath      []string
		file        string // "" = no instance yet
		list        string // colima list --json; "" = command fails
		service     bool   // brew services reports colima running
		fix         bool
		stuck       bool // the VM keeps reporting its old size after the restart
		wantSubstr  string
		wantWarn    int
		wantRestart bool
		wantFile    string // "" = file must be unchanged
	}{
		{name: "not darwin → nothing", goos: "linux", onPath: []string{"colima"}, file: colimaYAML("2", "2")},
		{name: "no colima → nothing", goos: "darwin", file: colimaYAML("2", "2")},
		{name: "no instance yet → skip naming the engine fix", goos: "darwin", onPath: []string{"colima"}, wantSubstr: "no Colima instance yet"},
		{name: "in sync and running at size → pass", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("4", "8"), list: vmRunning44, wantSubstr: "4 CPU, 8 GiB"},
		{name: "in sync, VM stopped → pass", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("4", "8"), list: vmStopped22, wantSubstr: "[ OK ] 4 CPU, 8 GiB"},
		{name: "file drift, no --fix → warn, file untouched", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("2", "2"), list: vmRunning22, service: true,
			wantWarn: 1, wantSubstr: "colima.yaml has 2 CPU, 2 GiB; the repo declares 4 CPU, 8 GiB (run: dotf doctor --fix, which restarts Colima"},
		{name: "memory alone drifts → warn", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("4", "2"), list: vmStopped22,
			wantWarn: 1, wantSubstr: "colima.yaml has 4 CPU, 2 GiB"},
		{name: "file drift, --fix, VM stopped → edit only", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("2", "2"), list: vmStopped22, fix: true,
			wantSubstr: "applies when Colima next starts", wantFile: colimaYAML("4", "8")},
		{name: "file drift, --fix, VM running under the service → edit and restart", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("2", "2.5"), list: vmRunning22, service: true, fix: true,
			wantSubstr: "its containers stopped", wantRestart: true, wantFile: colimaYAML("4", "8")},
		{name: "file in sync, VM at the old size, --fix → restart only", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("4", "8"), list: vmRunning22, service: true, fix: true,
			wantSubstr: "restarted Colima at 4 CPU, 8 GiB", wantRestart: true},
		{name: "restart that never reaches the size → warn, no repair claimed", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("4", "8"), list: vmRunning22, service: true, fix: true, stuck: true,
			wantWarn: 1, wantSubstr: "has not reported 4 CPU, 8 GiB yet", wantRestart: true},
		{name: "VM at the old size, no --fix → warn naming the running size", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("4", "8"), list: vmRunning22,
			wantWarn: 1, wantSubstr: "the VM runs with 2 CPU, 2 GiB"},
		{name: "running outside the service → edit, no restart, the command named", goos: "darwin", onPath: []string{"colima"}, file: colimaYAML("2", "2"), list: vmRunning22, fix: true,
			wantWarn: 1, wantSubstr: "colima stop && brew services start colima", wantFile: colimaYAML("4", "8")},
		{name: "file does not parse → warn", goos: "darwin", onPath: []string{"colima"}, file: "cpu: [\n", wantWarn: 1, wantSubstr: "does not parse"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			file := filepath.Join(home, ".colima", "default", "colima.yaml")
			if tc.file != "" {
				if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(tc.file), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			sys := newSys(map[string]string{"HOME": home}, tc.onPath, nil)
			sys.GOOS = tc.goos
			restarted := false
			sys.CommandOutputBounded = func(_ time.Duration, name string, args ...string) (string, string, error) {
				switch strings.Join(append([]string{name}, args...), " ") {
				case "colima list --json":
					if restarted && !tc.stuck {
						return vmRunning44 + "\n", "", nil
					}
					if tc.list != "" {
						return tc.list + "\n", "", nil
					}
				case "brew services info colima --json":
					return `[{"name":"colima","running":` + map[bool]string{true: "true", false: "false"}[tc.service] + `}]`, "", nil
				case "brew services restart colima":
					restarted = true
					return "", "", nil
				case "docker info --format {{.ServerVersion}}":
					return "28.5.1", "", nil
				}
				return "", "", errors.New("unexpected")
			}
			var buf bytes.Buffer
			rep := capture(&buf)
			checkColimaSize(sys, rep, tc.fix)
			out := buf.String()

			if tc.wantSubstr == "" && out != "" {
				t.Fatalf("want no section, got\n%s", out)
			}
			if !strings.Contains(out, tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, out)
			}
			if rep.Warnings() != tc.wantWarn || rep.Failures() != 0 {
				t.Fatalf("warnings = %d (want %d), failures = %d\n%s", rep.Warnings(), tc.wantWarn, rep.Failures(), out)
			}
			if restarted != tc.wantRestart {
				t.Fatalf("restarted = %v, want %v\n%s", restarted, tc.wantRestart, out)
			}
			if tc.file == "" {
				return
			}
			got, _ := os.ReadFile(file)
			want := tc.file
			if tc.wantFile != "" {
				want = tc.wantFile
			}
			if string(got) != want {
				t.Fatalf("colima.yaml:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// The edit touches only the top-level key's line: an inline comment stays, a
// nested key of the same name is not the VM's, and an absent key is appended.
func TestSetYAMLLine(t *testing.T) {
	in := "cpu: 2 # mine\nkubernetes:\n  cpu: 1\n"
	got := string(setYAMLLine(setYAMLLine([]byte(in), "cpu", 4), "memory", 8))
	if want := "cpu: 4 # mine\nkubernetes:\n  cpu: 1\nmemory: 8\n"; got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}
