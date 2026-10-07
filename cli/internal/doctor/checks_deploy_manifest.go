package doctor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// checkDeployManifest reports (AI-039, #1322) whether every entry of
// ai/deploy.json is installed as the manifest says. It asks the deploy package
// for its plan — the same compare `dotf deploy` runs, so the two can never
// disagree about what "in sync" means — and never for a deploy: PlanConfig
// touches nothing, which matters because the old flow staged before it
// compared and a diagnostic built on it would have created ~/.copilot/ while
// asking whether ~/.copilot/settings.json was in sync.
//
// Two kinds of entry are not compared, and the PASS line says how many: a
// rendered one (its installed content is only known after `secrets render`,
// which needs the daemon; doctor stays read-only, though an absent rendered
// destination is still a WARN) and one whose `requires`
// command is absent (deploy skips it too, and a row for a tool the box does
// not carry is a WARN no remedy can clear — #843). Drift is a WARN, not a
// FAIL: a tool that co-owns its file (Copilot's `/model` writes `model`) may
// legitimately have moved a managed key, and the remedy is one command. A file
// whose content is in sync but whose mode deploy would fix is drift too, from
// deploy.ModeDrift, the predicate deploy runs (#1664).
//
// It also reports a directory the manifest deploys a private file into that
// is open to group or others (checkPrivateDeployDirs), which --fix tightens.
func checkDeployManifest(sys *System, rep *Report, fix bool) {
	rep.Section("Deployed agent configs (ai/deploy.json)")

	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — deploy manifest check skipped")
		return
	}
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(deploy.ManifestRel))) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		rep.Warn("cannot read " + deploy.ManifestRel + ": " + err.Error())
		return
	}
	man, err := deploy.ParseManifest(raw)
	if err != nil {
		rep.Fail(err.Error())
		return
	}

	home := sys.home()
	inSync, drifted, notCompared := 0, 0, 0
	for _, c := range man.Configs {
		if !c.AppliesOn(sys.GOOS) || (c.Requires != "" && !sys.has(c.Requires)) {
			notCompared++
			continue
		}
		if c.Render {
			// Its content is only comparable after `secrets render`, but its
			// existence is not: an absent destination is a deploy that never
			// ran, and a SKIP there let pi on a fresh Mac start with no models
			// and a doctor that read healthy (#2100).
			dst, err := deploy.ExpandDst(c.Dst, home, env.ResolvePath)
			switch {
			case err != nil:
				rep.Warn(fmt.Sprintf("%s: %v (run: dotf deploy %s)", c.Name, err, c.Name))
				drifted++
			case !pathExists(dst):
				rep.Warn(fmt.Sprintf("%s: %s not deployed (run: dotf deploy %s)", c.Name, dst, c.Name))
				drifted++
			case warnModeDrift(rep, c, dst):
				// The mode needs no render to read, and deploy fixes it on
				// this path too.
				drifted++
			default:
				notCompared++
			}
			continue
		}
		p, err := deploy.PlanConfig(c, repo, home, env.ResolvePath)
		if err != nil {
			rep.Warn(fmt.Sprintf("%s: %v (run: dotf deploy %s)", c.Name, err, c.Name))
			drifted++
			continue
		}
		if p.Symlink {
			// Its bytes may match: name the link, or the operator diffs two
			// identical files and stops trusting the line (#2054).
			rep.Warn(fmt.Sprintf("drift: %s — %s is a symlink, not the file %s deploys (run: dotf deploy %s)", c.Name, p.Dst, c.Src, c.Name))
			drifted++
			continue
		}
		if p.Changed {
			rep.Warn(fmt.Sprintf("drift: %s — %s is not what %s deploys (run: dotf deploy %s)", c.Name, p.Dst, c.Src, c.Name))
			drifted++
			continue
		}
		if p.ModeDrift {
			mode, _ := c.FileMode() // PlanConfig has already parsed it
			rep.Warn(modeDriftLine(c.Name, p.Dst, mode))
			drifted++
			continue
		}
		inSync++
	}
	if drifted == 0 {
		rep.Pass(fmt.Sprintf("%d deployed config(s) in sync with %s (%d not compared: rendered, or tool absent)", inSync, deploy.ManifestRel, notCompared))
	}
	checkPrivateDeployDirs(sys, man, rep, fix)
}

// warnModeDrift reports a rendered destination whose mode deploy would fix,
// through the same predicate deploy uses (deploy.ModeDrift), and says whether
// it did.
func warnModeDrift(rep *Report, c deploy.Config, dst string) bool {
	mode, err := c.FileMode()
	if err != nil {
		rep.Warn(fmt.Sprintf("%s: %v", c.Name, err))
		return true
	}
	_, drift, err := deploy.ModeDrift(dst, mode)
	switch {
	case err != nil:
		rep.Warn(fmt.Sprintf("%s: mode on %s: %v (run: dotf deploy %s)", c.Name, dst, err, c.Name))
		return true
	case drift:
		rep.Warn(modeDriftLine(c.Name, dst, mode))
		return true
	}
	return false
}

// modeDriftLine names the declared mode, not "too open": the predicate also
// fires for a missing owner bit and, on Windows, for an inherited DACL.
func modeDriftLine(name, dst string, mode os.FileMode) string {
	return fmt.Sprintf("drift: %s — %s does not carry its declared mode %04o (run: dotf deploy %s)", name, dst, mode.Perm(), name)
}

// checkPrivateDeployDirs reports a directory that holds a private deployed file
// but grants group or others access: a ~/.ssh left 0755 from before #2051.
// Deploy creates such a directory 0700 and never changes one that exists,
// because tightening a directory on every deploy is a decision the operator
// should see. --fix makes it. Windows has no POSIX mode bits to check.
func checkPrivateDeployDirs(sys *System, man *deploy.Manifest, rep *Report, fix bool) {
	if sys.GOOS == "windows" {
		return
	}
	applies := func(c deploy.Config) bool {
		return c.AppliesOn(sys.GOOS) && (c.Requires == "" || sys.has(c.Requires))
	}
	dirs, err := man.PrivateDirs(sys.home(), env.ResolvePath, applies)
	if err != nil {
		rep.Warn("private deploy directories: " + err.Error())
		return
	}
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		perm := info.Mode().Perm()
		if perm&0o077 == 0 {
			continue
		}
		if !fix {
			rep.Warn(fmt.Sprintf("%s is %04o but holds a private deployed file; want 0700 (run: dotf doctor --fix)", dir, perm))
			continue
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			rep.Fail(fmt.Sprintf("could not tighten %s to 0700: %v", dir, err))
			continue
		}
		rep.Fix(fmt.Sprintf("tightened %s from %04o to 0700 (it holds a private deployed file)", dir, perm))
	}
}
