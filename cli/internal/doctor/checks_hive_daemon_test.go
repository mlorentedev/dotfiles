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

// hiveServers writes a mcp-servers.json whose hive entry runs args, in a temp
// DotfilesDir; an empty args writes no hive entry at all.
func hiveServers(t *testing.T, args string) *Config {
	t.Helper()
	dir := t.TempDir()
	servers := `{"servers": [{"name": "context7", "transport": "http", "args": "https://mcp.context7.com/mcp"}`
	if args != "" {
		servers += `, {"name": "hive", "transport": "stdio", "args": "` + args + `", "prerequisite_binary": "uv"}`
	}
	servers += "]}"
	if err := os.WriteFile(filepath.Join(dir, "mcp-servers.json"), []byte(servers), 0o600); err != nil {
		t.Fatal(err)
	}
	return &Config{DotfilesDir: dir}
}

// statusSeam fakes `hive service status`. exitErr stands for its exit 1, which
// it returns for every state except healthy.
func statusSeam(stdout, stderr string, exitErr error) func(time.Duration, string, ...string) (string, string, error) {
	return func(_ time.Duration, name string, args ...string) (string, string, error) {
		if name != "hive" || strings.Join(args, " ") != "service status" {
			return "", "", errors.New("unexpected command: " + name + " " + strings.Join(args, " "))
		}
		return stdout, stderr, exitErr
	}
}

var errExit1 = errors.New("exit status 1")

func runHiveDaemon(sys *System, cfg *Config) string {
	var buf bytes.Buffer
	checkHiveDaemonAnswers(sys, cfg, capture(&buf))
	return buf.String()
}

// The incident: on the Mac mini (2026-10-09) no daemon ran, every session's
// hive MCP was CONNECTION_CLOSED, and doctor printed an INFO line. Output
// captured from `hive service status` on that host, hive-vault 4.3.0.
func TestHiveDaemon_DownOnDarwinFails(t *testing.T) {
	sys := &System{
		GOOS:                 "darwin",
		LookPath:             lookPathFor("uv", "hive"),
		CommandOutputBounded: statusSeam("hive daemon: down\n", "", errExit1),
	}

	got := runHiveDaemon(sys, hiveServers(t, "hive client"))

	if !strings.Contains(got, "[FAIL]") || !strings.Contains(got, "the hive daemon is down") {
		t.Fatalf("a dead daemon must FAIL and name its state, got: %s", got)
	}
	if !strings.Contains(got, "no hive supervisor yet") {
		t.Errorf("darwin has no supervisor yet, and the failure must say so, got: %s", got)
	}
	if strings.Contains(got, "setup-linux") {
		t.Errorf("the Mac must never be told to run setup-linux.sh (ADR-045), got: %s", got)
	}
}

// On Linux the supervisor's own view comes first. The verdict is the prefixed
// line, so systemd chatter that says "active" must not decide it.
func TestHiveDaemon_VerdictLineWinsOverPassthrough(t *testing.T) {
	passthrough := "● hive.service - Hive vault daemon\n     Active: active (running) since Fri\n\n"
	sys := &System{
		GOOS:                 "linux",
		LookPath:             lookPathFor("uv", "hive"),
		CommandOutputBounded: statusSeam(passthrough+"hive daemon: unverified listener (held by another account); possible impersonation\n", "", errExit1),
	}

	got := runHiveDaemon(sys, hiveServers(t, "hive client"))

	if !strings.Contains(got, "unverified listener (held by another account)") {
		t.Fatalf("the failure must quote hive's state verbatim, got: %s", got)
	}
	if !strings.Contains(got, "systemctl --user restart hive.service") {
		t.Errorf("the Linux failure must name the supervisor fix, got: %s", got)
	}
}

func TestHiveDaemon_HealthyPasses(t *testing.T) {
	sys := &System{
		GOOS:                 "linux",
		LookPath:             lookPathFor("uv", "hive"),
		CommandOutputBounded: statusSeam("● hive.service\nhive daemon: healthy\n", "", nil),
	}

	got := runHiveDaemon(sys, hiveServers(t, "hive client"))

	if !strings.Contains(got, "[ OK ]") || strings.Contains(got, "[FAIL]") {
		t.Fatalf("a healthy daemon must pass, got: %s", got)
	}
}

// The command exits 0 only when healthy, so a healthy line paired with an
// error is a probe that died after printing: it must not pass.
func TestHiveDaemon_HealthyLineWithAnErrorIsNoPass(t *testing.T) {
	sys := &System{
		GOOS:                 "linux",
		LookPath:             lookPathFor("uv", "hive"),
		CommandOutputBounded: statusSeam("hive daemon: healthy\n", "", errors.New("hive timed out after 20s")),
	}

	got := runHiveDaemon(sys, hiveServers(t, "hive client"))

	if strings.Contains(got, "[ OK ]") || !strings.Contains(got, "timed out") {
		t.Fatalf("a failed probe must not pass on a healthy line, got: %s", got)
	}
}

// The gate reads the registration as registration does (strings.Fields), and
// a registration that outlived its prerequisite still launches the shim while
// `hive` is installed.
func TestHiveDaemon_GateMatchesWhatAgentsLaunch(t *testing.T) {
	cases := map[string]struct {
		args    string
		present []string
	}{
		"extra spacing":                {"hive  client", []string{"uv", "hive"}},
		"prerequisite gone, hive kept": {"hive client", []string{"hive"}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			sys := &System{
				GOOS:                 "darwin",
				LookPath:             lookPathFor(c.present...),
				CommandOutputBounded: statusSeam("hive daemon: down\n", "", errExit1),
			}

			if got := runHiveDaemon(sys, hiveServers(t, c.args)); !strings.Contains(got, "the hive daemon is down") {
				t.Fatalf("the daemon must be checked, got: %q", got)
			}
		})
	}
}

// No verdict line is a different state from "down": the probe itself did not
// answer, and the report must not claim the daemon's state.
func TestHiveDaemon_NoVerdictIsNotReportedAsDown(t *testing.T) {
	cases := map[string]struct {
		stderr string
		err    error
		want   string
	}{
		"probe error": {"hive: cannot probe the daemon: HIVE_DAEMON_PORT must be an integer port, got 'x'\n", errExit1, "HIVE_DAEMON_PORT must be an integer"},
		"timeout":     {"", errors.New("hive timed out after 20s"), "timed out"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			sys := &System{
				GOOS:                 "darwin",
				LookPath:             lookPathFor("uv", "hive"),
				CommandOutputBounded: statusSeam("", c.stderr, c.err),
			}

			got := runHiveDaemon(sys, hiveServers(t, "hive client"))

			if !strings.Contains(got, "gave no verdict") || !strings.Contains(got, c.want) {
				t.Fatalf("expected a no-verdict failure carrying %q, got: %s", c.want, got)
			}
			if strings.Contains(got, "is down") {
				t.Errorf("an unanswered probe must not be reported as a down daemon, got: %s", got)
			}
		})
	}
}

// Registered to launch a binary that is not installed: every agent's hive
// fails, so this is a FAIL, not a silent skip.
func TestHiveDaemon_RegisteredButNotInstalledFails(t *testing.T) {
	sys := &System{
		GOOS:                 "darwin",
		LookPath:             lookPathFor("uv"),
		CommandOutputBounded: statusSeam("", "", errors.New("must not run")),
	}

	got := runHiveDaemon(sys, hiveServers(t, "hive client"))

	if !strings.Contains(got, "`hive` is not on PATH") {
		t.Fatalf("expected the not-installed failure, got: %s", got)
	}
}

// Nothing launches `hive client` here, so there is no claim to check and the
// section must not appear at all.
func TestHiveDaemon_SilentWhenNoAgentLaunchesHiveClient(t *testing.T) {
	probe := statusSeam("hive daemon: down\n", "", errExit1)
	cases := map[string]struct {
		cfg     func(t *testing.T) *Config
		present []string
	}{
		"no hive entry":           {func(t *testing.T) *Config { return hiveServers(t, "") }, []string{"uv", "hive"}},
		"per-session server":      {func(t *testing.T) *Config { return hiveServers(t, "uvx hive-vault") }, []string{"uv", "hive"}},
		"neither uv nor hive":     {func(t *testing.T) *Config { return hiveServers(t, "hive client") }, nil},
		"no server list deployed": {func(t *testing.T) *Config { return &Config{DotfilesDir: t.TempDir()} }, []string{"uv", "hive"}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			sys := &System{GOOS: "darwin", LookPath: lookPathFor(c.present...), CommandOutputBounded: probe}

			if got := runHiveDaemon(sys, c.cfg(t)); got != "" {
				t.Fatalf("expected no output, got: %s", got)
			}
		})
	}
}

// A server list that does not decode is drift of its own: say the check did
// not run rather than skip it silently.
func TestHiveDaemon_UnreadableServerListWarns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mcp-servers.json"), []byte(`{"servers": [`), 0o600); err != nil {
		t.Fatal(err)
	}
	sys := &System{GOOS: "linux", LookPath: lookPathFor("uv", "hive")}

	got := runHiveDaemon(sys, &Config{DotfilesDir: dir})

	if !strings.Contains(got, "[WARN]") || !strings.Contains(got, "not checked") {
		t.Fatalf("expected a not-checked warning, got: %s", got)
	}
}
