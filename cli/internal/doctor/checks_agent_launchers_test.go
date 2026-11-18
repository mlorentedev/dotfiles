package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// launcherSys is a Windows System whose PATH resolves each agent to the given
// file and whose tool manager answers with behind. Nothing may be started: the
// check reads where a launch would go, it does not make one.
func launcherSys(t *testing.T, onPath, behind map[string]string) *System {
	t.Helper()
	sys := newSys(nil, nil, nil)
	sys.GOOS = "windows"
	sys.LookPath = func(name string) (string, error) {
		if bin, ok := onPath[name]; ok {
			return bin, nil
		}
		return "", errors.New("not found: " + name)
	}
	sys.NativeBehind = func(name string) string { return behind[name] }
	sys.CommandOutput = func(name string, args ...string) (string, error) {
		t.Errorf("the check started %s %v", name, args)
		return "", errors.New("must not run")
	}
	return sys
}

// lineFor returns the report line that names agent, "" when there is none.
func lineFor(out, agent string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, " "+agent+":") {
			return line
		}
	}
	return ""
}

// WIN-015: on Windows doctor says, per installed agent, which file a launch
// starts, and warns when the only launcher is a batch file dotf refuses.
func TestCheckAgentLaunchers_ReportsWhatALaunchStarts(t *testing.T) {
	dir := t.TempDir()
	native := filepath.Join(dir, "pi.exe")
	stub := filepath.Join(dir, "opencode") // npm's extensionless shell stub
	for _, f := range []string{native, stub} {
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sys := launcherSys(t,
		map[string]string{
			"claude":   `C:\Users\u\.local\bin\claude.exe`,
			"pi":       `C:\Users\u\.pi\agent\bin\pi.cmd`,
			"opencode": `C:\Users\u\AppData\Roaming\npm\opencode.cmd`,
			"copilot":  `C:\Users\u\AppData\Roaming\npm\copilot.bat`,
		},
		map[string]string{"pi": native, "opencode": stub})

	var buf bytes.Buffer
	checkAgentLaunchers(sys, capture(&buf))
	out := buf.String()

	for agent, want := range map[string][]string{
		"claude":   {"[ OK ]", `claude.exe`},
		"pi":       {"[ OK ]", native, `pi.cmd`},
		"opencode": {"[WARN]", `opencode.cmd`, "mise use -g opencode"},
		"copilot":  {"[WARN]", `copilot.bat`, "mise use -g copilot"},
		"agy":      {"[SKIP]", "not installed"},
	} {
		line := lineFor(out, agent)
		for _, w := range want {
			if !strings.Contains(line, w) {
				t.Errorf("%s: line %q, want it to hold %q\n%s", agent, line, w, out)
			}
		}
	}
	if strings.Contains(lineFor(out, "opencode"), stub) {
		t.Errorf("an extensionless stub is not a native executable and must not be named as the launch target\n%s", out)
	}
}

// Every agent a launch table knows is reported: one added to the table without
// a line here would be the agent whose first launch fails unannounced.
func TestCheckAgentLaunchers_CoversTheLaunchTable(t *testing.T) {
	var buf bytes.Buffer
	checkAgentLaunchers(launcherSys(t, nil, nil), capture(&buf))
	for _, agent := range lane.Agents() {
		if lineFor(buf.String(), agent) == "" {
			t.Errorf("no line for %s\n%s", agent, buf.String())
		}
	}
}

// Off Windows no launch goes through cmd.exe: one skipped line, and neither
// PATH nor the tool manager is asked.
func TestCheckAgentLaunchers_SkipsOffWindows(t *testing.T) {
	for _, goos := range []string{"linux", "darwin"} {
		sys := launcherSys(t, nil, nil)
		sys.GOOS = goos
		sys.LookPath = func(name string) (string, error) {
			t.Errorf("%s: PATH asked about %s", goos, name)
			return "", errors.New("unused")
		}
		sys.NativeBehind = func(name string) string {
			t.Errorf("%s: the tool manager asked about %s", goos, name)
			return ""
		}
		var buf bytes.Buffer
		checkAgentLaunchers(sys, capture(&buf))
		if out := buf.String(); strings.Count(out, "[SKIP]") != 1 || strings.Contains(out, "[WARN]") || strings.Contains(out, "[ OK ]") {
			t.Errorf("%s: want one skipped line and nothing else\n%s", goos, out)
		}
	}
}

// A System with no tool-manager seam (every other test's) reads as a machine
// whose tool manager has nothing: the batch launcher is reported, and the
// machine's own mise is never reached.
func TestCheckAgentLaunchers_WithoutAToolManagerAnswer(t *testing.T) {
	sys := launcherSys(t, map[string]string{"pi": `C:\npm\pi.cmd`}, nil)
	sys.NativeBehind = nil
	var buf bytes.Buffer
	checkAgentLaunchers(sys, capture(&buf))
	if line := lineFor(buf.String(), "pi"); !strings.Contains(line, "[WARN]") {
		t.Errorf("line %q, want the batch launcher reported\n%s", line, buf.String())
	}
}
