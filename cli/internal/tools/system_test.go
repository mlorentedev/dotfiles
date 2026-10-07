package tools

import (
	"fmt"
	"strings"
	"testing"
)

// ghTool is a system entry naming every manager, as P5b will ship them.
func ghTool() Tool {
	return Tool{Name: "gh", Profile: "full", Source: Source{
		Type: "system", Apt: "gh", Brew: "gh", Winget: "GitHub.cli",
	}}
}

// world is a fake machine: which package managers are on PATH, which packages
// each reports installed, and a record of every command Installer ran. Nothing
// real is ever called.
type world struct {
	managers  map[string]bool // binaries on PATH
	installed map[string]bool // package -> reported by its manager
	root      bool
	ran       [][]string
	failRun   error // returned by Run
	noEffect  bool  // Run exits 0 but installs nothing
	queries   [][]string
	out       strings.Builder
}

func newWorld(managers ...string) *world {
	w := &world{managers: map[string]bool{}, installed: map[string]bool{}}
	for _, m := range managers {
		w.managers[m] = true
	}
	return w
}

func (w *world) installer(goos string) *Installer {
	return &Installer{
		GOOS: goos, GOARCH: "amd64", Dest: "unused", Out: &w.out,
		IsRoot:     func() bool { return w.root },
		HasCommand: func(name string) bool { return w.managers[name] },
		Query: func(name string, args ...string) ([]byte, error) {
			w.queries = append(w.queries, append([]string{name}, args...))
			pkg := ""
			switch name {
			case "dpkg-query":
				pkg = args[len(args)-1]
				if w.installed[pkg] {
					return []byte("install ok installed"), nil
				}
				return []byte("deinstall ok config-files"), fmt.Errorf("exit 1") // removed, config left
			case "brew":
				pkg = args[len(args)-1]
				if w.installed[pkg] {
					return []byte(pkg + " 2.40.0\n"), nil
				}
				return nil, fmt.Errorf("exit 1")
			case "winget":
				for i, a := range args {
					if a == "--id" {
						pkg = args[i+1]
					}
				}
				if w.installed[pkg] {
					return []byte("Name  Id\n" + pkg + "  " + pkg + "\n"), nil
				}
				// Some winget builds exit 0 and print a sentence, others exit non-zero.
				return []byte("No installed package found matching input criteria.\n"), nil
			}
			return nil, fmt.Errorf("unexpected query %s", name)
		},
		Run: func(name string, args ...string) error {
			w.ran = append(w.ran, append([]string{name}, args...))
			if w.failRun != nil {
				return w.failRun
			}
			if w.noEffect {
				return nil
			}
			// The package is the last bare argument, or the --id value.
			pkg := args[len(args)-1]
			for i, a := range args {
				if a == "--id" {
					pkg = args[i+1]
				}
			}
			w.installed[pkg] = true
			return nil
		},
	}
}

func TestInstallSystem_ArgvPerManager(t *testing.T) {
	cases := []struct {
		name, goos string
		root       bool
		managers   []string
		want       string
	}{
		{"apt through sudo", "linux", false, []string{"apt-get", "sudo"}, "sudo apt-get install -y gh"},
		{"apt as root has no sudo to ask", "linux", true, []string{"apt-get"}, "apt-get install -y gh"},
		{"brew", "darwin", false, []string{"brew"}, "brew install gh"},
		{"winget names the id, exact match, and both agreements", "windows", false, []string{"winget"},
			"winget install --id GitHub.cli -e --accept-source-agreements --accept-package-agreements"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(tc.managers...)
			w.root = tc.root
			res, err := w.installer(tc.goos).Install(ghTool())
			if err != nil || res != Installed {
				t.Fatalf("Install = %v, %v; want Installed, nil\n%s", res, err, w.out.String())
			}
			if len(w.ran) != 1 || strings.Join(w.ran[0], " ") != tc.want {
				t.Errorf("ran %v, want exactly %q", w.ran, tc.want)
			}
		})
	}
}

// A system package is not pinned, so presence is the whole criterion and a second
// run must run no manager command at all.
func TestInstallSystem_SecondRunRunsNothing(t *testing.T) {
	w := newWorld("brew")
	in := w.installer("darwin")
	if res, err := in.Install(ghTool()); err != nil || res != Installed {
		t.Fatalf("first run = %v, %v", res, err)
	}
	w.ran = nil
	res, err := in.Install(ghTool())
	if err != nil || res != Skipped {
		t.Fatalf("second run = %v, %v; want Skipped, nil", res, err)
	}
	if len(w.ran) != 0 {
		t.Errorf("the second run ran %v; a present package must change nothing", w.ran)
	}
	if !strings.Contains(w.out.String(), "already installed") {
		t.Errorf("the skip must say so, got %q", w.out.String())
	}
	if p := in.Plan(ghTool()); p.Action != PlanSkip {
		t.Errorf("Plan after install = %q, want %q (never an upgrade)", p.Action, PlanSkip)
	}
}

func TestInstallSystem_NoNameForThisOSIsSkipped(t *testing.T) {
	linuxOnly := Tool{Name: "xclip", Source: Source{Type: "system", Apt: "xclip"}}
	w := newWorld("brew", "winget")
	for _, goos := range []string{"darwin", "windows"} {
		w.out.Reset()
		res, err := w.installer(goos).Install(linuxOnly)
		if err != nil || res != Skipped {
			t.Errorf("%s: Install = %v, %v; want Skipped, nil", goos, res, err)
		}
		if !strings.Contains(w.out.String(), "skipping") || !strings.Contains(w.out.String(), goos) {
			t.Errorf("%s: the skip must be visible and name the OS, got %q", goos, w.out.String())
		}
		if p := w.installer(goos).Plan(linuxOnly); p.Action != PlanUnsupported {
			t.Errorf("%s: Plan = %q, want %q", goos, p.Action, PlanUnsupported)
		}
	}
	if len(w.ran) != 0 || len(w.queries) != 0 {
		t.Errorf("a skipped entry reached a manager: ran %v, queried %v", w.ran, w.queries)
	}
}

// The package manager exiting 0 is not the package being there (the W1 lesson,
// applied to class 3): the manager's own record is the post-condition.
func TestInstallSystem_ExitZeroWithoutThePackageIsAnError(t *testing.T) {
	w := newWorld("brew")
	w.noEffect = true
	_, err := w.installer("darwin").Install(ghTool())
	if err == nil || !strings.Contains(err.Error(), "gh") || !strings.Contains(err.Error(), "brew") {
		t.Errorf("want an error naming gh and brew, got %v", err)
	}
}

func TestInstallSystem_ManagerFailureIsReportedWithTheCommand(t *testing.T) {
	w := newWorld("brew")
	w.failRun = fmt.Errorf("exit status 1")
	_, err := w.installer("darwin").Install(ghTool())
	if err == nil || !strings.Contains(err.Error(), "brew install gh") {
		t.Errorf("want the failing command in the error, got %v", err)
	}
}

// With no manager on PATH there is nothing to run: a skip that names the manager
// (setup installs it later), the same as a uv tool before uv exists.
func TestInstallSystem_NoManagerOnPathIsASkipNamingIt(t *testing.T) {
	w := newWorld() // nothing on PATH
	in := w.installer("darwin")
	res, err := in.Install(ghTool())
	if err != nil || res != Skipped {
		t.Fatalf("Install = %v, %v; want Skipped, nil", res, err)
	}
	if !strings.Contains(w.out.String(), "brew") {
		t.Errorf("the skip must name the missing manager, got %q", w.out.String())
	}
	if len(w.ran) != 0 {
		t.Errorf("ran %v without a manager", w.ran)
	}
	if p := in.Plan(ghTool()); p.Action != PlanMissingManager {
		t.Errorf("Plan = %q, want %q", p.Action, PlanMissingManager)
	}
}

// The presence rule: the manager's own query, or the entry's declared command on
// PATH. A removed package whose config lingers (dpkg "config-files") is absent.
func TestInstallSystem_PresenceRule(t *testing.T) {
	t.Run("a declared command on PATH satisfies the entry without asking the manager", func(t *testing.T) {
		w := newWorld("apt-get", "sudo", "gh")
		tool := ghTool()
		tool.Source.Command = "gh"
		res, err := w.installer("linux").Install(tool)
		if err != nil || res != Skipped || len(w.ran) != 0 {
			t.Errorf("Install = %v, %v, ran %v; want a skip", res, err, w.ran)
		}
	})
	t.Run("a command that is not on PATH falls back to the manager's record", func(t *testing.T) {
		w := newWorld("apt-get", "sudo")
		w.installed["gh"] = true
		tool := ghTool()
		tool.Source.Command = "gh"
		if res, _ := w.installer("linux").Install(tool); res != Skipped || len(w.ran) != 0 {
			t.Errorf("a package the manager lists must be left alone, ran %v", w.ran)
		}
	})
	t.Run("dpkg config-files only is absent", func(t *testing.T) {
		w := newWorld("apt-get", "sudo")
		if p := w.installer("linux").Plan(ghTool()); p.Action != PlanInstall || p.Installed != "" {
			t.Errorf("Plan = %+v, want install of an absent package", p)
		}
	})
	t.Run("winget's no-match sentence is absent even when it exits 0", func(t *testing.T) {
		w := newWorld("winget")
		if p := w.installer("windows").Plan(ghTool()); p.Action != PlanInstall {
			t.Errorf("Plan = %+v, want install", p)
		}
	})
	t.Run("a present package reads as present, not as a version", func(t *testing.T) {
		w := newWorld("brew")
		w.installed["gh"] = true
		p := w.installer("darwin").Plan(ghTool())
		if p.Action != PlanSkip || p.Installed != "present" || p.Pin != "" {
			t.Errorf("Plan = %+v", p)
		}
	})
}

// A dry run reaches only the queries, never the manager's install.
func TestPlanSystem_NeverRuns(t *testing.T) {
	w := newWorld("apt-get", "sudo")
	in := w.installer("linux")
	if p := in.Plan(ghTool()); p.Action != PlanInstall {
		t.Errorf("Plan = %+v", p)
	}
	if len(w.ran) != 0 {
		t.Errorf("a plan ran %v", w.ran)
	}
}

// A catalog written for a newer dotf must not fail the whole run on an older one:
// an unknown source type is a skip with a warning, while the plan still calls it
// unsupported (the same agreement AC3 sets for a missing asset).
func TestInstall_UnknownSourceTypeIsSkippedNotFailed(t *testing.T) {
	var out strings.Builder
	in := &Installer{GOOS: "linux", GOARCH: "amd64", Dest: t.TempDir(), Out: &out}
	tool := Tool{Name: "x", Version: "1.0.0", Source: Source{Type: "flatpak"}}
	res, err := in.Install(tool)
	if err != nil || res != Skipped {
		t.Fatalf("Install = %v, %v; want Skipped, nil", res, err)
	}
	if !strings.Contains(out.String(), "flatpak") || !strings.Contains(out.String(), "skipping") {
		t.Errorf("the warning must name the type and say it skips, got %q", out.String())
	}
	if p := in.Plan(tool); p.Action != PlanUnsupported {
		t.Errorf("Plan = %q, want %q", p.Action, PlanUnsupported)
	}
}
