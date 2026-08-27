package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// checkPiExtensions reports hand-wired pi extensions that shadow a package the
// manifest installs (AI-030 / #1243).
//
// THE DEFECT THIS EXISTS FOR, measured 2026-08-26. `pi -p` exited 1 with:
//
//	Error: Failed to load extension ".../npm/node_modules/pi-subagents/index.ts":
//	       Tool "subagent" conflicts with /home/manu/.pi/agent/extensions/subagent/index.ts
//
// pi did not start at all. Two `subagent` tools existed: one from
// `npm:pi-subagents@0.56.0`, which ai/pi/packages.json declares and setup
// reconciles, and one hand-symlinked on 2026-08-09 into pi's OWN bundled
// examples directory under a specific nvm node version. The hand-wired copy
// won, and the declared one is the one that never loaded.
//
// WHY THE EXISTING VERIFICATION COULD NOT SEE IT. AI-030's reconcile proves the
// `packages` array of the live settings.json converges on the manifest, and
// specs/AI-030-pi-packages-manifest/verify-reconcile.sh drives the real block to
// prove it. Both are correct and both are blind here: a package can be
// installed, declared, and counted while a file elsewhere stops it loading.
// Counting declarations is not observing effect — the failure this repository
// has now catalogued seven times inside this one spec family.
//
// THE RULE, and why it is this rule rather than a name list. A shadow is an
// entry under ~/.pi/agent/extensions/ that resolves, through a symlink, into a
// node_modules tree. That shape means exactly one thing: someone linked a
// package's own source or example code in by hand. Nothing reproduces it — not
// setup, not the manifest, not a fresh machine — and it is pinned to whichever
// node version happened to be active the day it was made.
//
// It is deliberately NOT "any extension the manifest does not declare".
// ~/.pi/agent/extensions/ also holds files written by other tools (three
// orca-*.ts appeared there while this check was being written), and claiming
// the whole directory for the manifest would report a live external writer as
// drift. The symlink-into-node_modules rule catches the defect and leaves
// genuinely hand-authored extensions alone.
func checkPiExtensions(sys *System, cfg *Config, rep *Report, fix bool) {
	rep.Section("pi extensions")

	agentDir := filepath.Join(sys.home(), ".pi", "agent")
	extDir := filepath.Join(agentDir, "extensions")

	entries, err := os.ReadDir(extDir)
	if err != nil {
		if os.IsNotExist(err) {
			rep.Skip("no ~/.pi/agent/extensions — pi has no hand-placed extensions to shadow anything")
			return
		}
		rep.Warn(fmt.Sprintf("cannot read %s: %v", extDir, err))
		return
	}

	installed := installedPiPackages(agentDir)

	var shadows []piShadow
	for _, e := range entries {
		name := e.Name()
		dir := filepath.Join(extDir, name)
		for _, link := range symlinksInto(dir, "node_modules") {
			shadows = append(shadows, piShadow{
				extension: name,
				link:      link.path,
				target:    link.target,
				// A shadow is CONFIRMED when a package of the same name is also
				// installed: that is the pair that races for one tool name. An
				// unconfirmed one is still unreproducible, but it is not
				// currently breaking anything, so it warns rather than fails.
				collides: installed[name],
			})
		}
	}

	if len(shadows) == 0 {
		rep.Pass(fmt.Sprintf("no hand-wired extension shadows a manifest package (%d extension entries, %d packages declared)",
			len(entries), piPackagesManifest(sys, cfg)))
		return
	}

	sort.Slice(shadows, func(i, j int) bool { return shadows[i].link < shadows[j].link })

	for _, s := range shadows {
		detail := fmt.Sprintf("%s -> %s", short(sys, s.link), short(sys, s.target))
		if !s.collides {
			rep.Warn(fmt.Sprintf(
				"hand-wired extension link, not reproduced by setup: %s\n"+
					"    Nothing installs this on a fresh machine, and it is pinned to one node version.", detail))
			continue
		}
		rep.Fail(fmt.Sprintf(
			"extension %q shadows the installed package of the same name: %s\n"+
				"    pi refuses to start when both provide the same tool. Re-check with: pi -p 'ok'", s.extension, detail))
	}

	if !fix {
		if anyCollides(shadows) {
			rep.Info("re-run with --fix to remove the hand-wired links and let the declared package load")
		}
		return
	}

	repairPiShadows(sys, rep, shadows)
}

type piShadow struct {
	extension string
	link      string
	target    string
	collides  bool
}

func anyCollides(ss []piShadow) bool {
	for _, s := range ss {
		if s.collides {
			return true
		}
	}
	return false
}

type symlink struct{ path, target string }

// symlinksInto returns the symlinks directly under dir whose resolved target
// contains a path segment equal to segment. Resolution is EvalSymlinks so a
// chain of links is followed to whatever it really ends at; a link that dangles
// is reported on its raw target instead, because a broken hand-wired link is
// still a hand-wired link.
func symlinksInto(dir, segment string) []symlink {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []symlink
	for _, e := range entries {
		if e.Type()&os.ModeSymlink == 0 {
			continue
		}
		p := filepath.Join(dir, e.Name())
		target, err := filepath.EvalSymlinks(p)
		if err != nil {
			if target, err = os.Readlink(p); err != nil {
				continue
			}
		}
		if slices.Contains(strings.Split(filepath.ToSlash(target), "/"), segment) {
			out = append(out, symlink{path: p, target: target})
		}
	}
	return out
}

// installedPiPackages reports which packages are unpacked under
// ~/.pi/agent/npm/node_modules, keyed by the name a tool namespace would use.
// `pi install` unpacks there as well as writing the settings.json array, so this
// reads what is ON DISK rather than what is declared — the whole point of the
// check is that the two can disagree.
//
// A scoped package (@scope/pi-thing) is keyed on its unscoped half, and every
// key is also recorded with a leading "pi-" stripped, because the extension
// directory is named for the tool ("subagent") while the package is named
// "pi-subagents". Matching is therefore generous by design: this map only
// decides FAIL versus WARN, and both dispositions name the same file.
func installedPiPackages(agentDir string) map[string]bool {
	root := filepath.Join(agentDir, "npm", "node_modules")
	found := map[string]bool{}

	record := func(name string) {
		if name == "" {
			return
		}
		found[name] = true
		if trimmed, ok := strings.CutPrefix(name, "pi-"); ok {
			found[trimmed] = true
			found[strings.TrimSuffix(trimmed, "s")] = true
		}
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return found
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "@") {
			scoped, err := os.ReadDir(filepath.Join(root, e.Name()))
			if err != nil {
				continue
			}
			for _, s := range scoped {
				record(s.Name())
			}
			continue
		}
		record(e.Name())
	}
	return found
}

// short renders a path under $HOME as ~/... so the diagnostic is readable and
// carries no machine-specific prefix (ADR-025: never print a literal home).
func short(sys *System, p string) string {
	home := sys.home()
	if home != "" && strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + string(filepath.Separator) + p[len(home)+1:]
	}
	return p
}

// piPackagesManifest is the declaration setup reconciles against. Read only to
// report how many packages are at stake when a shadow is found. Checkout
// first, mirror second — the ADR-030 precedence every other registry read
// here follows: the deploy mirror never carried ai/pi/, so this reported
// "0 packages declared" as a PASS on Windows (WIN-007/#1288).
func piPackagesManifest(sys *System, cfg *Config) int {
	path := filepath.Join(cfg.DotfilesDir, "ai", "pi", "packages.json")
	if repo := resolveRepoDir(sys); repo != "" {
		if p := filepath.Join(repo, "ai", "pi", "packages.json"); pathExists(p) {
			path = p
		}
	}
	doc, err := os.ReadFile(path) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return 0
	}
	var parsed struct {
		Packages []json.RawMessage `json:"packages"`
	}
	if json.Unmarshal(doc, &parsed) != nil {
		return 0
	}
	return len(parsed.Packages)
}
