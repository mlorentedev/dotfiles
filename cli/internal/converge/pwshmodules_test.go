package converge

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// galleryPwsh is a pwsh whose CurrentUser scope holds the modules in
// installed, as name@version. An install adds what it is asked for unless
// brokenInstall is set, an install that reports success and leaves nothing.
type galleryPwsh struct {
	installed     map[string]bool
	brokenInstall bool
	installs      int
}

func (g *galleryPwsh) run(name string, args ...string) ([]byte, error) {
	if name != "pwsh" || len(args) != 4 {
		return nil, errors.New("galleryPwsh: unexpected " + name)
	}
	var out strings.Builder
	for l := range strings.SplitSeq(args[3], "\n") {
		f := strings.Split(l, "'")
		switch {
		case strings.HasPrefix(l, "try { Import-Module"):
			if !g.installed[f[1]+"@"+f[3]] {
				fmt.Fprintf(&out, "MISSING %s\n", f[1])
			}
		case strings.HasPrefix(l, "Install-Module"):
			g.installs++
			if !g.brokenInstall {
				g.installed[f[1]+"@"+f[3]] = true
			}
		}
	}
	return []byte(out.String()), nil
}

const pwshVersions = "# pwsh-module: PSScriptAnalyzer\nPSSCRIPTANALYZER_VERSION=1.25.0\n# pwsh-module: Pester\nPESTER_VERSION=6.2.0\n"

func pwshEnv(t *testing.T, versions string) Env {
	t.Helper()
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{"versions.conf": versions})
	return Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "darwin"}
}

// From a pwsh with an older analyzer and no Pester: the plan names both and
// installs nothing; the apply installs both, the probe passes, and the next
// plan has nothing to do (PLAT-003 AC1, AC3).
func TestPwshModules_InstallsWhatIsNotAtItsPin(t *testing.T) {
	g := &galleryPwsh{installed: map[string]bool{"PSScriptAnalyzer@1.24.0": true}}
	step, env := pwshModules{run: g.run, has: func(string) bool { return true }}, pwshEnv(t, pwshVersions)

	plan, err := step.Reconcile(env, true)
	if err != nil || plan.Changes != 2 || g.installs != 0 || !strings.Contains(plan.Detail, "PSScriptAnalyzer@1.25.0, Pester@6.2.0") {
		t.Fatalf("plan = %+v, %v, %d installs", plan, err, g.installs)
	}
	if res, err := step.Reconcile(env, false); err != nil || res.Changes != 2 {
		t.Fatalf("apply = %+v, %v", res, err)
	}
	if err := step.Probe(env); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	if again, err := step.Reconcile(env, true); err != nil || again.Changes != 0 {
		t.Errorf("second plan = %+v, %v; want converged", again, err)
	}
}

func TestPwshModules_OnlyTheMissingModuleIsInstalled(t *testing.T) {
	g := &galleryPwsh{installed: map[string]bool{"PSScriptAnalyzer@1.25.0": true}}
	step := pwshModules{run: g.run, has: func(string) bool { return true }}
	if res, err := step.Reconcile(pwshEnv(t, pwshVersions), false); err != nil || res.Changes != 1 || g.installs != 1 {
		t.Errorf("apply = %+v, %v, %d installs; want Pester alone", res, err, g.installs)
	}
}

func TestPwshModules_ProbeFailsWhenAnInstallLeftNothing(t *testing.T) {
	g := &galleryPwsh{installed: map[string]bool{}, brokenInstall: true}
	step, env := pwshModules{run: g.run, has: func(string) bool { return true }}, pwshEnv(t, pwshVersions)
	if _, err := step.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := step.Probe(env); err == nil || !strings.Contains(err.Error(), "Pester@6.2.0") {
		t.Errorf("probe = %v, want it to name what is still missing", err)
	}
}

func TestPwshModules_SkippedWithoutPwsh(t *testing.T) {
	step := pwshModules{run: (&galleryPwsh{}).run, has: func(string) bool { return false }}
	res, err := step.Reconcile(pwshEnv(t, pwshVersions), true)
	if err != nil || res.Skip == "" {
		t.Errorf("got %+v, %v; want a skip", res, err)
	}
}

func TestPwshModules_NothingMarkedIsConverged(t *testing.T) {
	g := &galleryPwsh{installed: map[string]bool{}}
	step := pwshModules{run: g.run, has: func(string) bool { return true }}
	res, err := step.Reconcile(pwshEnv(t, "PESTER_VERSION=6.2.0\n"), false)
	if err != nil || res.Changes != 0 || g.installs != 0 {
		t.Errorf("got %+v, %v, %d installs", res, err, g.installs)
	}
}

func TestPwshModules_AMalformedMarkerFails(t *testing.T) {
	step := pwshModules{run: (&galleryPwsh{}).run, has: func(string) bool { return true }}
	if _, err := step.Reconcile(pwshEnv(t, "# pwsh-module: Pester\n"), true); err == nil {
		t.Error("a marker over no pin must fail the step, not converge nothing")
	}
}
