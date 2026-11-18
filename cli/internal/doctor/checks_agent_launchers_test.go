package doctor

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// launcherSys is a Windows System whose PATH resolves each agent to the given
// file and whose tool manager answers with behind. A process started through
// the System fails the test. That does not cover lane.MiseNative, which starts
// mise by itself: what keeps a test off the machine's mise is that the check
// only asks the seam, which TestCheckAgentLaunchers_AsksOnlyTheSeam holds.
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

// The tool manager is asked through the seam, once for each agent whose
// launcher is a batch file and for no other: a check that asked mise itself
// would leave the seam uncalled, and pi's native executable unreported.
func TestCheckAgentLaunchers_AsksOnlyTheSeam(t *testing.T) {
	sys := launcherSys(t, map[string]string{
		"claude":   `C:\bin\claude.exe`,
		"pi":       `C:\npm\pi.cmd`,
		"opencode": `C:\npm\opencode.cmd`,
	}, nil)
	var asked []string
	sys.NativeBehind = func(name string) string {
		asked = append(asked, name)
		return ""
	}
	var buf bytes.Buffer
	checkAgentLaunchers(sys, capture(&buf))
	sort.Strings(asked)
	if want := []string{"opencode", "pi"}; !reflect.DeepEqual(asked, want) {
		t.Errorf("the tool manager was asked about %v, want %v", asked, want)
	}
}

// The review launcher's runners are a switch, not a list the check can read.
// Each runner it accepts must be in the lane launch table, the one the check
// walks, or doctor would say nothing about a runner a review starts.
func TestCheckAgentLaunchers_CoversTheReviewRunners(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "spec", "review_launch.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var runners []string
	ast.Inspect(file, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		if tag, ok := sw.Tag.(*ast.SelectorExpr); !ok || tag.Sel.Name != "Runner" {
			return true
		}
		for _, stmt := range sw.Body.List {
			for _, expr := range stmt.(*ast.CaseClause).List {
				if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if name, err := strconv.Unquote(lit.Value); err == nil {
						runners = append(runners, name)
					}
				}
			}
		}
		return true
	})
	if len(runners) == 0 {
		t.Fatal("no runner found in the review launcher's switch: the test reads nothing")
	}
	known := map[string]bool{}
	for _, agent := range lane.Agents() {
		known[agent] = true
	}
	for _, runner := range runners {
		if !known[runner] {
			t.Errorf("review runner %q is not in the lane launch table, so doctor does not report its launcher", runner)
		}
	}
}

// A System with no tool-manager seam (every other test's) reads as a machine
// whose tool manager has nothing: the batch launcher is reported.
func TestCheckAgentLaunchers_WithoutAToolManagerAnswer(t *testing.T) {
	sys := launcherSys(t, map[string]string{"pi": `C:\npm\pi.cmd`}, nil)
	sys.NativeBehind = nil
	var buf bytes.Buffer
	checkAgentLaunchers(sys, capture(&buf))
	if line := lineFor(buf.String(), "pi"); !strings.Contains(line, "[WARN]") {
		t.Errorf("line %q, want the batch launcher reported\n%s", line, buf.String())
	}
}

// The production System carries the seam: without it every batch launcher
// would be reported as having nothing native behind it.
func TestRealSystemAsksTheToolManager(t *testing.T) {
	if realSystem().NativeBehind == nil {
		t.Fatal("realSystem has no NativeBehind: doctor would never find a native executable behind a shim")
	}
}
