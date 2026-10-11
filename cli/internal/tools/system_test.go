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
	failIndex error // returned by Run for `apt-get update` alone
	sudoOK    bool  // `sudo -n true` succeeds, so sudo itself is not what refused
	noEffect  bool  // Run exits 0 but installs nothing
	queries   [][]string
	out       strings.Builder
	apps      map[string]string // cask -> the `app` artifact `brew info` reports
	bundles   map[string]bool   // app bundles on disk, from any channel
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
		AppExists:  func(bundle string) bool { return w.bundles[bundle] },
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
			case "sudo":
				if w.sudoOK {
					return nil, nil
				}
				return []byte("sudo: a password is required"), fmt.Errorf("exit 1")
			case "brew":
				pkg = args[len(args)-1]
				if args[0] == "info" {
					app, ok := w.apps[pkg]
					if !ok {
						return nil, fmt.Errorf("exit 1")
					}
					return []byte(`{"casks":[{"token":"` + pkg + `","artifacts":[{"uninstall":[{"quit":"x"}]},{"app":[` + app + `]},{"zap":[]}]}]}`), nil
				}
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
			if args[len(args)-1] == "update" {
				if w.failRun != nil {
					return w.failRun
				}
				return w.failIndex
			}
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
		want       []string
	}{
		{"apt through sudo -n, which never prompts, after refreshing its index", "linux", false, []string{"apt-get", "sudo"},
			[]string{"sudo -n apt-get update", "sudo -n apt-get install -y --no-remove gh"}},
		{"apt as root has no sudo to ask", "linux", true, []string{"apt-get"},
			[]string{"apt-get update", "apt-get install -y --no-remove gh"}},
		{"brew", "darwin", false, []string{"brew"}, []string{"brew install gh"}},
		{"winget names the id, exact match, and both agreements", "windows", false, []string{"winget"},
			[]string{"winget install --id GitHub.cli -e --accept-source-agreements --accept-package-agreements"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(tc.managers...)
			w.root = tc.root
			w.sudoOK = true
			res, err := w.installer(tc.goos).Install(ghTool())
			if err != nil || res != Installed {
				t.Fatalf("Install = %v, %v; want Installed, nil\n%s", res, err, w.out.String())
			}
			var ran []string
			for _, argv := range w.ran {
				ran = append(ran, strings.Join(argv, " "))
			}
			if strings.Join(ran, "; ") != strings.Join(tc.want, "; ") {
				t.Errorf("ran %q, want exactly %q", ran, tc.want)
			}
		})
	}
}

// A cask is a separate brew namespace: `brew list --versions <cask>` exits 1 for
// an installed cask (measured, Homebrew 7.0.9), so querying it as a formula
// would plan an install on every run and fail the post-condition after it.
func TestInstallSystem_CaskIsQueriedAndInstalledAsACask(t *testing.T) {
	obsidian := Tool{Name: "obsidian", Profile: "full", Source: Source{Type: "system", Cask: "obsidian"}}
	w := newWorld("brew")
	res, err := w.installer("darwin").Install(obsidian)
	if err != nil || res != Installed {
		t.Fatalf("Install = %v, %v; want Installed, nil\n%s", res, err, w.out.String())
	}
	if len(w.ran) != 1 || strings.Join(w.ran[0], " ") != "brew install --cask obsidian" {
		t.Errorf("ran %v, want exactly %q", w.ran, "brew install --cask obsidian")
	}
	for _, q := range w.queries {
		if q[0] == "brew" && !strings.Contains(strings.Join(q, " "), " --cask ") {
			t.Errorf("queried %v; a cask is asked about with --cask", q)
		}
	}

	w.ran, w.queries = nil, nil
	if res, err := w.installer("darwin").Install(obsidian); err != nil || res != Skipped || len(w.ran) != 0 {
		t.Errorf("second run: %v, %v, ran %v; want Skipped and nothing run", res, err, w.ran)
	}
}

// An app installed by hand (dragged into /Applications, a vendor installer) is not
// in `brew list --cask`, and `brew install --cask` refuses to overwrite its
// bundle, so an entry for it failed on every run. The bundle on disk is the
// cask's presence, as a declared command on PATH is a formula's.
func TestInstallSystem_CaskAppInstalledOutsideBrewIsPresent(t *testing.T) {
	obsidian := Tool{Name: "obsidian", Profile: "full", Source: Source{Type: "system", Cask: "obsidian"}}
	for _, tc := range []struct {
		name, app, bundle string
		present           bool
	}{
		{"the bundle the cask names is on disk", `"Obsidian.app"`, "Obsidian.app", true},
		{"an app renamed by its target is looked up by the target", `{"target":"Obsidian Beta.app"}`, "Obsidian Beta.app", true},
		{"no bundle on disk is absent", `"Obsidian.app"`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld("brew")
			w.apps = map[string]string{"obsidian": tc.app}
			w.bundles = map[string]bool{tc.bundle: tc.bundle != ""}
			p := w.installer("darwin").Plan(obsidian)
			if got := p.Action == PlanSkip; got != tc.present {
				t.Errorf("Plan = %+v; present = %v, want %v", p, got, tc.present)
			}
		})
	}
	t.Run("a formula never falls back to an app bundle", func(t *testing.T) {
		w := newWorld("brew")
		w.apps = map[string]string{"gh": `"gh.app"`}
		w.bundles = map[string]bool{"gh.app": true}
		if p := w.installer("darwin").Plan(ghTool()); p.Action != PlanInstall {
			t.Errorf("Plan = %+v, want install", p)
		}
	})
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
	// The plan names the manager as the uv and npm waits do, so a converge
	// report reads "gh (brew)", not "gh ()".
	if p := in.Plan(ghTool()); p.Action != PlanMissingManager || p.Note != "waits on brew" {
		t.Errorf("Plan = %q %q, want %q naming brew", p.Action, p.Note, PlanMissingManager)
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
		w.sudoOK = true
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
	w.sudoOK = true
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
	// The plan says what the apply does: a skip, in the same words.
	p := in.Plan(tool)
	if p.Action != PlanSkip || !strings.Contains(p.Note, `source type "flatpak" is not known to this dotf`) {
		t.Errorf("Plan = %+v, want a skip noting the unknown type", p)
	}
	if !strings.Contains(out.String(), p.Note) {
		t.Errorf("Install's warning %q does not carry the plan's note %q", out.String(), p.Note)
	}
}

// sudo -n is the only escalation: it fails instead of prompting. When it fails
// because a password is needed, that is its own outcome (a skip naming the exact
// command to run), and the tools after it still converge.
func TestInstallSystem_NeedsSudoIsNamedAndDoesNotFailTheRun(t *testing.T) {
	w := newWorld("apt-get", "sudo")
	w.failRun = fmt.Errorf("exit status 1") // sudo -n refused; w.sudoOK stays false
	in := w.installer("linux")
	res, err := in.Install(ghTool())
	if err != nil || res != Skipped {
		t.Fatalf("Install = %v, %v; want Skipped, nil", res, err)
	}
	if want := "gh: needs sudo; run: sudo apt-get install -y --no-remove gh"; !strings.Contains(w.out.String(), want) {
		t.Errorf("output %q lacks %q", w.out.String(), want)
	}
	if len(w.ran) != 1 || w.ran[0][1] != "-n" || w.ran[0][len(w.ran[0])-1] != "gh" {
		t.Errorf("ran %v, want one `sudo -n` install attempt, no prompt, and no index refresh sudo would refuse", w.ran)
	}
	if strings.Contains(w.out.String(), "index refresh failed") {
		t.Errorf("a refresh sudo would refuse is the install's needs-sudo line, not a warning of its own: %q", w.out.String())
	}
}

// A fresh machine's apt index is stale: the runner image's lists named a
// libgit2 build its mirror had already dropped, so `apt-get install eza` failed
// with a 404 and converge stopped on its first apt entry (#2013 X1, run
// 38107367419). One refresh per run, before the first install and never on a
// run with nothing to install.
func TestInstallSystem_AptRefreshesItsIndexOnceBeforeTheFirstInstall(t *testing.T) {
	w := newWorld("apt-get")
	w.root = true
	in := w.installer("linux")
	parallel := Tool{Name: "parallel", Profile: "full", Source: Source{Type: "system", Apt: "parallel"}}
	for _, tool := range []Tool{ghTool(), parallel} {
		if res, err := in.Install(tool); err != nil || res != Installed {
			t.Fatalf("Install(%s) = %v, %v", tool.Name, res, err)
		}
	}
	var ran []string
	for _, argv := range w.ran {
		ran = append(ran, strings.Join(argv, " "))
	}
	want := "apt-get update; apt-get install -y --no-remove gh; apt-get install -y --no-remove parallel"
	if got := strings.Join(ran, "; "); got != want {
		t.Errorf("ran %q, want %q", got, want)
	}

	w.ran = nil
	if res, err := w.installer("linux").Install(ghTool()); err != nil || res != Skipped || len(w.ran) != 0 {
		t.Errorf("a present package: Install = %v, %v, ran %v; want Skipped and no refresh", res, err, w.ran)
	}
}

// A refresh that fails for another reason (a third-party source that 404s, no
// network for the lists) warns and installs from the index as it is: the
// install's own failure, if any, is the error, so one broken source does not
// block every package.
func TestInstallSystem_AFailedIndexRefreshWarnsAndStillInstalls(t *testing.T) {
	w := newWorld("apt-get")
	w.root = true
	w.failIndex = fmt.Errorf("exit status 100")
	res, err := w.installer("linux").Install(ghTool())
	if err != nil || res != Installed {
		t.Fatalf("Install = %v, %v; want Installed, nil", res, err)
	}
	if want := "apt index refresh failed (apt-get update: exit status 100)"; !strings.Contains(w.out.String(), want) {
		t.Errorf("output %q lacks %q", w.out.String(), want)
	}
}

// A plan asks the apply's own classifier, so it never promises an install the
// apply would skip for want of a password: a converge probe re-plans after the
// apply, and an `install` row there would fail every unattended run.
func TestPlanSystem_NeedsSudoIsPlannedAsTheApplySkipsIt(t *testing.T) {
	w := newWorld("apt-get", "sudo")
	p := w.installer("linux").Plan(ghTool())
	if p.Action != PlanNeedsSudo || p.Note != "run: sudo apt-get install -y --no-remove gh" || p.Package != "gh" {
		t.Errorf("Plan = %+v, want needs-sudo naming the command", p)
	}
	if len(w.ran) != 0 {
		t.Errorf("a plan ran %v", w.ran)
	}

	in := w.installer("linux")
	in.Plan(ghTool())
	in.Plan(ghTool())
	asked := 0
	for _, q := range w.queries {
		if q[0] == "sudo" {
			asked++
		}
	}
	if asked != 2 { // the first Plan above, and this installer's one question
		t.Errorf("sudo asked %d times across three plans on two installers, want 2", asked)
	}

	root := newWorld("apt-get")
	root.root = true
	if p := root.installer("linux").Plan(ghTool()); p.Action != PlanInstall || len(root.queries) != 1 {
		t.Errorf("as root: Plan = %+v, queries %v; want install and no sudo query", p, root.queries)
	}
}

// A failure that is not sudo's refusal stays an error, so needs-sudo cannot
// hide a broken install.
func TestInstallSystem_AptFailureWithWorkingSudoIsAnError(t *testing.T) {
	w := newWorld("apt-get", "sudo")
	w.sudoOK = true
	w.failRun = fmt.Errorf("exit status 100")
	_, err := w.installer("linux").Install(ghTool())
	if err == nil || !strings.Contains(err.Error(), "sudo -n apt-get install -y --no-remove gh") {
		t.Errorf("want an error carrying the command, got %v", err)
	}
}

// The combined command is the per-package one with every package on it, and
// nothing at all for no packages (#2308).
func TestSudoInstallCommand(t *testing.T) {
	if got := SudoInstallCommand(nil); got != "" {
		t.Errorf("no packages: got %q, want \"\"", got)
	}
	if got, want := SudoInstallCommand([]string{"gh", "parallel"}), "sudo apt-get install -y --no-remove gh parallel"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	w := newWorld("apt-get", "sudo")
	if got, want := "run: "+SudoInstallCommand([]string{"gh"}), w.installer("linux").Plan(ghTool()).Note; got != want {
		t.Errorf("one package: combined %q drifts from the per-package note %q", got, want)
	}
}
