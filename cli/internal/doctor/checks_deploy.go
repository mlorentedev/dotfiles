package doctor

import (
	"fmt"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// checkSymlinks reproduces healthcheck section 4: the dotfiles symlinks resolve.
// A real-file-where-a-symlink-was-expected still PASSes (the deploy strategy
// moved to copy for some paths, ADR-012).
func checkSymlinks(sys *System, rep *Report) {
	rep.Section("Key symlinks")
	home := sys.home()
	win := sys.GOOS == "windows"
	// POSIX shell rc files have no Windows equivalent (pwsh uses $PROFILE) — skip
	// them there instead of reporting a false "missing".
	posixOnly := map[string]bool{
		".zshrc": true, ".bashrc": true,
		".zsh/aliases.zsh": true, ".zsh/functions.zsh": true,
	}
	for _, rel := range []string{
		".dotfiles", ".zshrc", ".bashrc",
		".zsh/aliases.zsh", ".zsh/functions.zsh", ".ssh/config",
	} {
		if win && posixOnly[rel] {
			rep.Skip(rel + " (POSIX-only; Windows uses $PROFILE)")
			continue
		}
		p := filepath.Join(home, rel)
		switch {
		case isSymlink(p) && pathExists(p):
			rep.Pass(rel + " symlink valid")
		case isSymlink(p):
			rep.Fail(rel + " symlink broken (dangling)")
		case pathExists(p):
			rep.Pass(rel + " exists (not a symlink)")
		default:
			rep.Fail(rel + " missing: " + p)
		}
	}
}

// checkVault reproduces healthcheck section 7's read-only PRESENCE checks only.
// Deep vault health (the vault-health.sh invocation and the obsidian-linter
// lintOnSave assertion) is deliberately NOT ported here — it routes to the
// future `dotf vault` (ADR-021), per the proposal's scope.
func checkVault(sys *System, rep *Report) {
	rep.Section("Knowledge vault (presence)")
	// VAULT_PATH is the canonical seam (ADR-025); the old VAULT_DIR name is gone.
	// The generated paths file sets VAULT_PATH, so the hardcoded default below is
	// only a last resort for a machine that never ran `dotf env generate`.
	vault := sys.env("VAULT_PATH", filepath.Join(sys.home(), "Projects", "knowledge"))

	presence := func(ok bool, okMsg, failMsg string) {
		if ok {
			rep.Pass(okMsg)
		} else {
			rep.Fail(failMsg)
		}
	}
	presence(isDir(vault), "vault directory exists ("+vault+")", "vault directory missing: "+vault)
	presence(isDir(filepath.Join(vault, ".obsidian")), ".obsidian/ configured", ".obsidian/ directory missing")
	presence(pathExists(filepath.Join(vault, ".obsidian", "types.json")), "types.json present", "types.json missing (property schema)")
	presence(sys.has("obsidian"), "Obsidian CLI in PATH", "Obsidian CLI not in PATH")
	for _, d := range []string{"00_meta", "10_projects", "40_resources"} {
		presence(isDir(filepath.Join(vault, d)), "vault directory: "+d+"/", "vault directory missing: "+d+"/")
	}
}

// checkPathFiles verifies the deployed paths.sh/paths.ps1 (ADR-025) match a
// fresh resolution of env-contract.json + machine.json. Drift means a path was
// changed in the contract or the per-machine override but `dotf env generate`
// was never re-run — the same copy-with-drift-assertion discipline as ADR-012.
func checkPathFiles(sys *System, cfg *Config, rep *Report) {
	// ADR-025
	rep.Section("Generated path files")
	if cfg.ContractPath == "" {
		rep.Skip("env-contract.json not found — path-file drift check skipped")
		return
	}
	home := sys.home()
	out := envpkg.DefaultOutput(sys.GOOS, sys.env("DOTFILES_DIR", filepath.Join(home, ".dotfiles")))
	if !pathExists(out) {
		rep.Warn(filepath.Base(out) + " not generated — run `dotf env generate`")
		return
	}
	res, err := envpkg.Generate(envpkg.Options{
		ContractPath: cfg.ContractPath,
		MachinePath:  envpkg.MachinePath(home),
		GOOS:         sys.GOOS,
		Home:         home,
		Output:       out,
		Check:        true,
	})
	if err != nil {
		rep.Warn("path-file drift check failed: " + err.Error())
		return
	}
	if res.Drifted {
		rep.Fail(filepath.Base(out) + " is stale — run `dotf env generate` (" + out + ")")
	} else {
		rep.Pass(filepath.Base(out) + " up to date (" + out + ")")
	}
}

// checkSecrets reproduces healthcheck section 8 over the registry SSOT: every
// age-backed secrets/registry.yaml entry resolves to an existing *.secret.age,
// and no orphan .age file lacks a registry entry. With fix, it prunes orphans
// from the deploy mirror (see pruneOrReportOrphans).
//
// Entries() returns a TAGGED UNION, not a list of age sources: #606 taught it to
// emit bw-backed secrets too, because the Loader dispatches on Backend. Only the
// age backends populate File. An earlier revision of this comment claimed Entries
// skipped bw entries and the loop asserted a blob for every one of them — for a bw
// entry that resolves to sensitive/.secret.age (empty base name), which cannot
// exist, so every bw secret read as a missing blob AND poisoned `referenced` with
// "", making every migrated secret's surviving DR blob read as an orphan. It stayed
// invisible while the registry held no bw entries and became 56 FAILs the day 28
// were migrated (#961, #965). Dispatch on the tag; do not infer it from File.
func checkSecrets(sys *System, cfg *Config, rep *Report, fix bool) {
	rep.Section("Secrets integrity")
	secretsDir := filepath.Join(cfg.DotfilesDir, "sensitive")

	reg, err := loadRegistry(cfg)
	if err != nil {
		rep.Fail("secrets/registry.yaml not found or invalid")
		return
	}
	rep.Pass("secrets/registry.yaml exists")

	referenced := map[string]bool{}
	for _, e := range reg.Entries(sys.home()) {
		display := e.Var
		if e.IsFile {
			display = e.Var + " [file]"
		}
		// Only bw is exempt, and it is named explicitly rather than the age
		// backends being whitelisted: age-offline is a backend too (the floor
		// plane), and a whitelist would silently stop checking any backend added
		// later. Unknown tags keep asserting — the check errs toward checking.
		if e.Backend == secrets.BackendBW {
			// Its live tier is proven by [Bitwarden reach], which exercises the
			// token. This section is about the age store, which a bw secret has
			// no declared entry in.
			rep.Pass(fmt.Sprintf("%s -> bw:%s (age store not asserted)", display, e.Item))
			continue
		}
		if e.Backend == secrets.BackendFileAuthority {
			if e.Dest == "" {
				rep.Fail(fmt.Sprintf("%s -> (no path) file-authority missing path", display))
				continue
			}
			if pathExists(e.Dest) {
				checkFileAuthorityMode(rep, e, display, fix)
			} else {
				rep.Fail(fmt.Sprintf("%s -> %s (file-authority missing on disk)", display, e.Dest))
			}
			continue
		}
		referenced[e.File] = true
		if pathExists(filepath.Join(secretsDir, e.File+".secret.age")) {
			rep.Pass(fmt.Sprintf("%s -> %s.secret.age", display, e.File))
		} else {
			rep.Fail(fmt.Sprintf("%s -> %s.secret.age (missing)", display, e.File))
		}
	}

	var orphans []string
	for _, f := range secretBlobCandidates(secretsDir) {
		base, complete := strings.CutSuffix(f, ".secret.age")
		if !complete || !referenced[base] {
			orphans = append(orphans, f)
		}
	}
	pruneOrReportOrphans(sys, cfg, rep, orphans, fix)
}

// checkFileAuthorityMode holds a root that is on disk to its declared mode, as
// `dotf secrets verify` does: a key restored with a plain copy keeps the copy's
// mode, 0644 under the default umask, and every account on the machine can then
// read the key that decrypts every age secret (#2203). With fix it sets the
// declared mode; the key's content is never read.
func checkFileAuthorityMode(rep *Report, e secrets.Entry, display string, fix bool) {
	err := secrets.CheckFileAuthorityMode(e)
	if err == nil {
		rep.Pass(fmt.Sprintf("%s -> %s (file-authority on disk)", display, e.Dest))
		return
	}
	if !fix {
		rep.Fail(fmt.Sprintf("%s -> %s (run: dotf doctor --fix)", display, err))
		return
	}
	if rerr := secrets.RepairFileAuthorityMode(e); rerr != nil {
		rep.Fail(fmt.Sprintf("%s -> %s, and setting the declared mode failed: %v", display, err, rerr))
		return
	}
	rep.Fix(fmt.Sprintf("%s -> %s: set to the declared mode", display, err))
}

// secretBlobCandidates lists the top-level files in the secrets dir that hold
// age-encrypted secret material: *.secret.age, and the *.secret.age.tmp.* that
// a shell-era writer left behind when interrupted (one on msi, 2026-05-11). A
// partial write is never claimed by an entry. Machine-local files (env-mapping.conf,
// README.md) and the dr/ escrow never match.
func secretBlobCandidates(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !e.Type().IsRegular() {
			continue
		}
		if strings.HasSuffix(name, ".secret.age") || strings.Contains(name, ".secret.age.tmp.") {
			out = append(out, name)
		}
	}
	return out
}

// pruneOrReportOrphans FAILs every age blob no registry entry claims, and with
// fix removes it from the deploy mirror when that is safe.
//
// An unclaimed blob used to degrade to one WARN once any secret had migrated, on
// the belief that such blobs were "the ADR-028 floor" (#971). They are not: the
// floor is the escrow, a verified export of the whole vault (ADR-028 §5), and
// §3 gates age-file retirement on exactly that. Per-secret blobs exist only for
// age-offline entries, which claim theirs. CLI-036 retired the other 31.
//
// The prune follows #802's decision (doctor --fix prunes; setup only copies) and
// refuses unless the checkout resolveRepoDir found is plausibly the dotfiles one:
// it must differ from the mirror and hold both secrets/registry.yaml and the DR
// escrow. resolveRepoDir proves only "a git checkout"; against an unrelated repo
// every mirror blob would look deleted (see checkHarnessMirrorOrphans). A blob
// the checkout still holds is not pruned either, since setup would copy it back.
func pruneOrReportOrphans(sys *System, cfg *Config, rep *Report, orphans []string, fix bool) {
	if len(orphans) == 0 {
		return
	}
	mirrorDir := filepath.Join(cfg.DotfilesDir, "sensitive")
	repo := resolveRepoDir(sys)
	refusal := ""
	switch {
	case repo == "":
		refusal = "no dotfiles checkout resolved"
	case filepath.Clean(repo) == filepath.Clean(cfg.DotfilesDir):
		refusal = "the mirror is the checkout; git rm it there"
	case !pathExists(filepath.Join(repo, "secrets", "registry.yaml")) ||
		!pathExists(filepath.Join(repo, "sensitive", "dr", "bitwarden-export.age")):
		refusal = "checkout " + repo + " lacks the registry or the DR escrow"
	}
	for _, name := range orphans {
		switch {
		case refusal != "":
			rep.Fail("orphan: " + name + " (no registry entry; not pruned: " + refusal + ")")
		case pathExists(filepath.Join(repo, "sensitive", name)):
			rep.Fail("orphan: " + name + " (no registry entry; " + checkoutCopyRemedy(sys, repo, name) + ")")
		case !fix:
			rep.Fail("orphan: " + name + " (no registry entry, gone from the checkout — run: dotf doctor --fix)")
		default:
			if err := os.Remove(filepath.Join(mirrorDir, name)); err != nil {
				rep.Fail("failed to prune orphan: " + name + " (" + err.Error() + ")")
			} else {
				rep.Fix("pruned orphan secret blob: sensitive/" + name)
			}
		}
	}
}

// checkoutCopyRemedy words the FAIL for an orphan the checkout still holds.
// Existence is not tracked state (#1793): a committed blob needs `git rm` in a
// PR, an untracked leftover needs deleting, and `git rm` fails on the second.
// When git cannot answer, the line says so instead of guessing either way.
func checkoutCopyRemedy(sys *System, repo, name string) string {
	where := repo + "/sensitive"
	if sys.CommandOutputDir == nil {
		return "still present in " + where + ": git rm it if tracked, delete it if not"
	}
	out, err := gitIn(sys, repo, "ls-files", "--", ":(literal)sensitive/"+name)
	switch {
	case err != nil:
		return "still present in " + where + ": git rm it if tracked, delete it if not"
	case strings.TrimSpace(out) != "":
		return "still committed in " + where + ", git rm it there"
	default:
		return "an untracked copy sits in " + where + " and setup copies it back: delete it there, then run dotf doctor --fix"
	}
}

// loadRegistry reads and parses secrets/registry.yaml under the dotfiles dir.
// Shared by checkSecrets and githubPATSecrets (both consume the mapping SSOT).
func loadRegistry(cfg *Config) (*secrets.Registry, error) {
	raw, err := os.ReadFile(filepath.Join(cfg.DotfilesDir, "secrets", "registry.yaml"))
	if err != nil {
		return nil, err
	}
	return secrets.ParseRegistry(raw)
}

// checkTmux reproduces healthcheck section 9: tmux is installed. ~/.tmux.conf is
// the `tmux` deploy entry's, compared by checkDeployManifest.
func checkTmux(sys *System, rep *Report) {
	rep.Section("tmux")
	if sys.GOOS == "windows" {
		rep.Skip("tmux (not available on Windows; use WSL if needed)")
		return
	}
	if !sys.has("tmux") {
		rep.Fail("tmux not installed (run: " + tmuxInstall(sys.GOOS) + ")")
		return
	}
	ver := "unknown" // tmux uses `-V`, not the conventional `--version`
	if v, err := sys.CommandOutput("tmux", "-V"); err == nil {
		ver = strings.TrimSpace(v)
	}
	rep.Pass("tmux installed: " + ver)
}

// tmuxInstall is the not-installed remedy. tmux is class 3 (ADR-044): no
// cross-OS channel ships it, so the remedy is the OS's package manager.
func tmuxInstall(goos string) string {
	if goos == "darwin" {
		return "brew install tmux"
	}
	return "sudo apt install -y tmux"
}

// checkOpenCode reproduces healthcheck section 10: opencode + pi are installed,
// on PATH, version-matched, and have their deployed config.
func checkOpenCode(sys *System, cfg *Config, rep *Report) {
	rep.Section("OpenCode + pi")
	home := sys.home()

	// opencode binary + version.
	opencodeBin := filepath.Join(home, ".opencode", "bin", "opencode")
	switch {
	case sys.has("opencode"):
		ver := semverOf(sys, "opencode")
		rep.Pass("opencode in PATH: " + ver)
		matchPinFloorFrom(rep, "opencode", ver, catalogPin(sys, cfg, "opencode"), "packages.json")
	case isExecFile(opencodeBin):
		// The retired curl-script channel (ADR-036) left a copy behind and
		// nothing on PATH resolves: the rc files no longer add ~/.opencode/bin.
		rep.Fail("opencode not on PATH; a legacy curl-script copy sits at " + opencodeBin + " — run `dotf tools install opencode` and delete the legacy copy")
	default:
		rep.Fail("opencode missing (run `dotf tools install opencode`)")
	}

	// opencode config + $schema.
	cfgPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	switch {
	case !pathExists(cfgPath):
		rep.Fail("opencode.jsonc missing: " + cfgPath + " (run: dotf deploy opencode)")
	case fileContains(cfgPath, `"$schema":`):
		rep.Pass("opencode.jsonc deployed with $schema declaration")
	default:
		rep.Fail("opencode.jsonc missing $schema declaration (run: dotf deploy opencode)")
	}

	// pi binary + version. pi is optional → SKIP when truly absent, but FAIL when
	// it is configured (~/.pi present) yet unreachable on PATH — the Orca / GUI
	// per-node-version PATH trap: pi installed under an nvm node version not on
	// the current PATH. The ~/.local launcher (see setup-linux.sh) is the
	// durable fix; this guard makes the trap loud instead of a misleading SKIP.
	piLocalBin := filepath.Join(home, ".local", "bin", "pi")
	piConfigured := pathExists(filepath.Join(home, ".pi", "agent", "models.json"))
	switch {
	case sys.has("pi"):
		ver := semverOf(sys, "pi")
		rep.Pass("pi in PATH: " + ver)
		matchPinFloor(rep, "pi", ver, cfg.Versions["PI_VERSION"])
	case isExecFile(piLocalBin):
		rep.Fail("pi exists at " + piLocalBin + " but not in PATH (reload shell)")
	case piConfigured:
		rep.Fail("pi configured (~/.pi present) but not on PATH — installed under a node version not on this PATH; re-run setup to install into ~/.local")
	default:
		rep.Skip("pi not installed (run setup, or npm i -g --ignore-scripts --prefix ~/.local @earendil-works/pi-coding-agent)")
	}

	// pi models.json + secret substitution state.
	models := filepath.Join(home, ".pi", "agent", "models.json")
	switch {
	case !pathExists(models):
		rep.Skip("pi models.json not deployed at " + models)
	case !fileContains(models, "{env:"):
		rep.Pass("pi models.json secret substituted (no {env:} placeholder left)")
	default:
		ageKey := sys.env("AGE_KEY_PATH", filepath.Join(home, ".config", "age", "key.txt"))
		if pathExists(ageKey) {
			rep.Fail("pi models.json has an unresolved {env:...} placeholder (re-run setup)")
		} else {
			rep.Skip("pi models.json substitution — age identity absent ({env:} resolves at runtime)")
		}
	}

	checkShadowedCatalogTools(sys, cfg, rep)

	// Launchability. Everything above is a static predicate — files and PATH —
	// and every one of them was green on a box where `pi` could not start
	// (WIN-012/#1293). The shell wrappers (profile.ps1, .zshrc, .bashrc) run
	// both agents under `dotf secrets run`, which resolves their keys BEFORE
	// exec'ing the binary; while those keys live in a locked Bitwarden vault the
	// resolution fails and the agent never launches. Report the precondition the
	// wrappers actually depend on, not a proxy for it.
	if piConfigured || pathExists(cfgPath) {
		reportAgentLaunchability(sys, rep)
	}
}

// reportAgentLaunchability reports whether the agent wrappers' key resolution
// can succeed right now. WARN, never FAIL: every reboot locks the vault, and a
// doctor that is red on every fresh boot is one nobody reads. Silent while no
// registry entry resolves through Bitwarden — the keys then come from the age
// floor, which needs no daemon — and while the registry is unreadable, which
// checkBitwardenReach already reports with its own reason.
func reportAgentLaunchability(sys *System, rep *Report) {
	live, err := sys.BWBackedSecrets()
	if err != nil || live == 0 {
		return
	}
	st, err := sys.BWServeStatus()
	if err != nil {
		return // checkBWServeDaemon reports the unreadable state
	}
	if st == "unlocked" {
		rep.Pass("agent wrappers can resolve their keys (bw serve daemon unlocked)")
		return
	}
	rep.Warn("pi/opencode wrappers will refuse to launch: their keys resolve through Bitwarden and no unlocked bw serve daemon is reachable — run `dotf secrets unlock` (once per boot; the daemon then serves every terminal)")
}

// checkHarnessDrift reproduces healthcheck section 11's harness/skill-record
// drift gate and the symlink-free-skills invariant. The repo↔deploy-dir drift
// half of §11 (the standalone diff-check twin) is a separate section,
// checkDeployDrift (CLI-019).
func checkHarnessDrift(sys *System, cfg *Config, rep *Report, fix bool) {
	rep.Section("Harness + skill drift")
	checkCompileHarnessDrift(sys, cfg, rep)
	checkHarnessMirrorOrphans(sys, cfg, rep, fix)
	checkDeployedSkillSymlinks(sys, cfg, rep)
	checkInstructionDrift(sys, rep)
	checkDeployedDoctrine(sys, cfg, rep)
	checkTriggerTargets(sys, cfg, rep)
}

var enforcedRegionMarkers = map[string]string{
	"no-attribution":      "No AI attribution",
	"english-only":        "English only",
	"no-phase-references": "No internal phase/milestone references",
	"no-auto-merge":       "Auto-merge is forbidden",
	// The rule's own opening line, not the phrase "Definition of Done" — which
	// appears nowhere in this record. Its only occurrence in the enforced set was
	// inside pr-stewardship's provenance blockquote ("It elaborates Definition of
	// Done §4 …"), so this check verified one region by finding another's
	// meta-text, and broke the moment #1181 compacted those blockquotes out of
	// the capped payload while the doctrine itself was entirely intact.
	// TestEveryDoctrineMarkerIsInItsOwnRecord keeps every marker honest.
	"definition-of-done": "Working code is not a finished change",
	"pr-stewardship":     "What binds is the disposition",
	"pr-sizing":          "Atomic PRs, ~300 LOC hard cap",
}

var deployedDoctrineTargets = []struct {
	homeRel string
	regions []string
}{
	{
		homeRel: ".gemini/GEMINI.md",
		regions: []string{"no-attribution", "english-only", "no-phase-references", "no-auto-merge", "definition-of-done", "pr-stewardship", "pr-sizing"},
	},
	{
		homeRel: ".codex/AGENTS.md",
		regions: []string{"no-attribution", "english-only", "no-phase-references", "no-auto-merge", "definition-of-done", "pr-stewardship", "pr-sizing"},
	},
	{
		homeRel: ".claude/CLAUDE.md",
		regions: []string{"no-attribution", "english-only", "no-phase-references", "no-auto-merge", "definition-of-done", "pr-stewardship"},
	},
}

// checkDeployedDoctrine asserts that every enforced doctrine region declared in the
// harness manifest actually survived deployment to runtime files in $HOME (HARNESS-074/#1035).
func checkDeployedDoctrine(sys *System, cfg *Config, rep *Report) {
	home := sys.home()
	checked, failures := 0, 0

	for _, tgt := range deployedDoctrineTargets {
		deployedPath := filepath.Join(home, filepath.FromSlash(tgt.homeRel))
		if !pathExists(deployedPath) {
			continue
		}

		contentBytes, err := os.ReadFile(deployedPath)
		if err != nil {
			continue
		}
		content := string(contentBytes)
		checked++

		for _, regionID := range tgt.regions {
			marker, ok := enforcedRegionMarkers[regionID]
			if !ok {
				continue
			}
			if !strings.Contains(content, marker) {
				rep.Fail(fmt.Sprintf("enforced region %q missing from deployed %s (run: compile-harness.sh --deploy)", regionID, tgt.homeRel))
				failures++
			}
		}
	}

	if checked == 0 {
		rep.Skip("no deployed doctrine payloads found to verify")
		return
	}
	if failures == 0 {
		rep.Pass(fmt.Sprintf("deployed doctrine payloads contain all enforced regions (%d surfaces verified)", checked))
	}
}

// deployedInstructionTargets mirrors harness/manifest.json's agents.presence[]
// (agent/file/source/requires_command) — the four instruction files
// compile-harness.sh --deploy copies verbatim (HARNESS-058/#828). Kept here
// rather than parsed from the manifest because the doctor binary must run
// with no repo/vault present at all; the manifest is only reachable once a
// repo IS found, at which point this list and the manifest are asserted in
// sync by TestCheckInstructionDrift_MatchesManifest.
var deployedInstructionTargets = []struct{ agent, homeRel, repoRel, requiresCommand string }{
	{"claude", ".claude/CLAUDE.md", "ai/claude/CLAUDE.md", ""},
	{"opencode", ".config/opencode/AGENTS.md", "AGENTS.md", ""},
	{"pi", ".pi/agent/AGENTS.md", "AGENTS.md", ""},
	{"copilot", ".copilot/copilot-instructions.md", "ai/copilot/copilot-instructions.md", "copilot"},
}

// checkInstructionDrift reports (AC2 of HARNESS-058/#828) a deployed
// instruction file that has drifted from its repo source — never silently.
// Comparison strips both harness marker-region kinds (the enforced-pattern
// GENERATED region and the AGENT-PRESENCE region) from each side first: the
// GENERATED region is baked into the repo source by --refresh so it is
// identical on both sides already, and the AGENT-PRESENCE region (plus,
// for copilot, the skill-catalog GENERATED region) is injected into the
// DEPLOYED copy only, after the copy — a naive byte-compare would false-fail
// immediately after a clean --deploy.
//
// A target with requiresCommand set is skipped entirely when that command is
// absent, mirroring deploy_instructions' own gate: a leftover
// copilot-instructions.md on a machine that never had `copilot` installed is
// never written by --deploy, so comparing it is not "drift" — it is a FAIL no
// remedy can ever clear, the exact #843 signal-rot this session exists to
// kill.
func checkInstructionDrift(sys *System, rep *Report) {
	missing := checkInstalledAgentsHaveInstructions(sys, rep)
	home := sys.home()
	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — instruction-file drift check skipped")
		return
	}
	checked, drift := 0, missing
	for _, tgt := range deployedInstructionTargets {
		if tgt.requiresCommand != "" && !sys.has(tgt.requiresCommand) {
			continue
		}
		deployed := filepath.Join(home, filepath.FromSlash(tgt.homeRel))
		source := filepath.Join(repo, filepath.FromSlash(tgt.repoRel))
		if !pathExists(deployed) || !pathExists(source) {
			continue // a missing file was judged above; this compares contents
		}
		dc, err1 := os.ReadFile(deployed)
		sc, err2 := os.ReadFile(source)
		if err1 != nil || err2 != nil {
			continue
		}
		checked++
		// Trailing newlines are not content either: the LF writer on Windows
		// ends a file with exactly one, while a source may end with a blank
		// line, and that alone read as drift on the CI runner (#1308).
		if !harness.DeployedMatchesSource(string(dc), string(sc)) {
			rep.Fail("stale: " + tgt.homeRel + " has drifted from " + tgt.repoRel + " (run: compile-harness.sh --deploy)")
			drift++
		}
	}
	if checked == 0 {
		rep.Skip("no deployed instruction files found to compare")
		return
	}
	if drift == 0 {
		rep.Pass(fmt.Sprintf("deployed instruction files match their repo source (%d checked)", checked))
	}
}

// checkInstalledAgentsHaveInstructions fails each installed agent whose
// instruction file is missing: it runs on its harness defaults, which is how AI
// attribution reached commits on a fresh machine (F-060, #2016). It needs only
// $HOME and PATH, because doctor runs with no checkout at all. An agent that is
// not installed is not a failure (#843). Returns the number of failures.
func checkInstalledAgentsHaveInstructions(sys *System, rep *Report) int {
	failed := 0
	for _, tgt := range deployedInstructionTargets {
		if !sys.has(tgt.agent) || pathExists(filepath.Join(sys.home(), filepath.FromSlash(tgt.homeRel))) {
			continue
		}
		rep.Fail(tgt.agent + " is installed but " + tgt.homeRel + " is missing (" + instructionsRemedy(sys.GOOS) + ")")
		failed++
	}
	return failed
}

// instructionsRemedy names what deploys the instruction files on goos: the
// records-harness reconciler covers linux and darwin, and Windows keeps its
// setup-script copies until the Go deploy of these files ships (PLAT-001b).
func instructionsRemedy(goos string) string {
	if goos == "windows" {
		return "re-run setup-windows.ps1"
	}
	return "run: dotf converge"
}

// Harness marker-region delimiters, owned by package harness (DeployedMatchesSource).
// TestHarnessMarkerConstants pins them against scripts/compile-harness.sh.
const (
	harnessBeginPrefix       = harness.GeneratedBeginPrefix
	harnessEndMarker         = harness.GeneratedEndMarker
	agentPresenceBeginPrefix = harness.PresenceBeginPrefix
	agentPresenceEndMarker   = harness.PresenceEndMarker
)

// checkHarnessMirrorOrphans detects harness/{skills,agents} records present in
// the deploy mirror (cfg.DotfilesDir) with no counterpart in the repo — the gap
// BUG-058/#843 describes: the repo->mirror copy (`dotf harness mirror`, and the
// setup-linux.sh bash block before it) is copy-only, so a record deleted from
// the repo survives in the mirror forever and keeps failing
// checkCompileHarnessDrift, which runs FROM the mirror. Per #802's decided
// semantic (doctor --fix prunes; setup only copies/warns) — generated records
// are prunable automatically here, unlike sensitive/*.secret.age.
//
// This applies on every OS. It used to early-return on Windows on the belief
// that Windows had no repo/mirror split; it did — setup-windows.ps1's
// `$DotfilesDest` is `~/.dotfiles`, the very dir cfg.DotfilesDir resolves to —
// it just never received harness/ (WIN-007/#1288). The "mirror IS the
// checkout" case is the guard below, wherever it occurs.
func checkHarnessMirrorOrphans(sys *System, cfg *Config, rep *Report, fix bool) {
	repo := resolveRepoDir(sys)
	if repo == "" || filepath.Clean(repo) == filepath.Clean(cfg.DotfilesDir) {
		return // no checkout found, or the "mirror" IS the checkout — nothing to compare
	}

	orphans := 0
	for _, sub := range []string{"skills", "agents"} {
		mirrorDir := filepath.Join(cfg.DotfilesDir, "harness", sub)
		if !isDir(mirrorDir) {
			continue
		}
		entries, err := os.ReadDir(mirrorDir)
		if err != nil {
			continue
		}
		repoDir := filepath.Join(repo, "harness", sub)
		if !isDir(repoDir) {
			// resolveRepoDir's DOTFILES_REPO_DIR/cwd-git-root cascade proves
			// only "a git checkout", not "the dotfiles checkout" (no such
			// validation exists — docs/lessons.md, the resolveRepoDir
			// test-isolation lesson). If it resolved to an unrelated repo
			// lacking this subtree entirely, every mirror entry would look
			// orphaned and --fix would delete the whole harness/<sub> tree.
			// Refuse to compare rather than risk that.
			rep.Skip("repo has no " + filepath.Join("harness", sub) + " — orphan comparison skipped (wrong checkout resolved?)")
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || isDir(filepath.Join(repoDir, e.Name())) {
				continue
			}
			orphans++
			rel := filepath.Join("harness", sub, e.Name())
			target := filepath.Join(mirrorDir, e.Name())
			if !fix {
				rep.Fail("orphan mirror record: " + rel + " (no repo counterpart — run: dotf doctor --fix)")
				continue
			}
			if err := os.RemoveAll(target); err != nil {
				rep.Fail("failed to prune orphan mirror record: " + rel + " (" + err.Error() + ")")
			} else {
				rep.Fix("pruned orphan mirror record: " + rel)
			}
		}
	}
	if orphans == 0 {
		rep.Pass("harness mirror has no orphan records")
	}
}

// checkCompileHarnessDrift runs the compile-harness --check drift gate. It is
// Linux-only: compile-harness.sh is the Linux generation engine, so on Windows
// (which deploys committed records via Deploy-SkillRecord and has no --check
// port yet — CLI-035) it SKIPs with the platform reason rather than the
// misleading "not found" of a mirror that never holds the script (BUG-052).
func checkCompileHarnessDrift(sys *System, cfg *Config, rep *Report) {
	compile := filepath.Join(cfg.DotfilesDir, "scripts", "compile-harness.sh")
	switch {
	case sys.GOOS == "windows":
		// CLI-035
		rep.Skip("harness drift gate is Linux-only; Windows deploys committed records, no --check port yet")
	case !isExecFile(compile):
		rep.Skip("compile-harness.sh not found at " + compile)
	default:
		if _, err := sys.CommandOutput("bash", compile, "--check"); err == nil {
			rep.Pass("harness blocks + skill records match their source-of-record (no drift)")
		} else {
			rep.Fail("harness/skill drift (run: compile-harness.sh --refresh, then re-deploy)")
		}
	}
}

// skillSymlinkRoot is a deployed-skill path this repo's harness manages,
// paired with how to recover the skill NAME from a symlink found under it.
type skillSymlinkRoot struct {
	dir      string
	fileMode bool // true: <name>.md files (opencode commands, agy prompts); false: <name>/ dirs (claude/agy skills)
}

// checkDeployedSkillSymlinks enforces the BUG-100 invariant — deployed skill
// paths this repo manages must be regular copies, never symlinks — but only
// for names the harness actually manages (a `harness/skills/<name>` record
// exists). Foreign tools legitimately symlink their OWN skills into the same
// directories: pi's installer links sibling skills from `~/.agents/skills`
// (documented exclusion, specs/archive/AI-022-pi-harness-slot), and Orca does
// the same for `~/.claude/skills` (e.g. computer-use, orca-cli). Flagging
// those fights another tool's filesystem layout — the exact class of bug
// BUG-100 was about in the first place — so a symlink at an unmanaged name is
// silent here, mirroring compile-harness.sh's warn_unmanaged_output policy on
// the deploy side.
func checkDeployedSkillSymlinks(sys *System, cfg *Config, rep *Report) {
	home := sys.home()
	roots := []skillSymlinkRoot{
		{filepath.Join(home, ".claude", "skills"), false},
		{filepath.Join(home, ".config", "opencode", "commands"), true},
		{filepath.Join(home, ".gemini", "skills"), false},
		{filepath.Join(home, ".gemini", "prompts"), true},
	}
	managed := managedSkillNames(sys, cfg)

	var present, flagged []string
	for _, r := range roots {
		if !isDir(r.dir) {
			continue
		}
		present = append(present, r.dir)
		for _, l := range findSymlinks([]string{r.dir}) {
			if managed[skillNameForSymlink(r, l)] {
				flagged = append(flagged, l)
			}
		}
	}
	if len(present) == 0 {
		rep.Skip("no deployed skill paths found (run setup to deploy skills)")
		return
	}
	if len(flagged) == 0 {
		rep.Pass("deployed skills are regular copies (no symlinks at managed skill names)")
		return
	}
	// BUG-100
	rep.Fail("deployed skill path(s) are symlinks (must be hard copies):")
	for _, l := range flagged {
		rep.Fail("  " + l)
	}
}

// managedSkillNames is the union of harness/skills/ record names from the
// deploy mirror and (when resolvable) the repo checkout — either one may be
// what actually rendered the deployed copy, depending on how the machine last
// deployed.
func managedSkillNames(sys *System, cfg *Config) map[string]bool {
	names := map[string]bool{}
	add := func(recdir string) {
		entries, err := os.ReadDir(recdir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				names[e.Name()] = true
			}
		}
	}
	add(filepath.Join(cfg.DotfilesDir, "harness", "skills"))
	if repo := resolveRepoDir(sys); repo != "" {
		add(filepath.Join(repo, "harness", "skills"))
	}
	return names
}

// skillNameForSymlink recovers the skill name a symlink found under root.dir
// belongs to: the file's basename minus ".md" for command/prompt renders, or
// the first path segment below root.dir for skill renders (covers both a
// symlinked SKILL.md one level in and a symlinked skill directory itself).
func skillNameForSymlink(root skillSymlinkRoot, symlinkPath string) string {
	if root.fileMode {
		return strings.TrimSuffix(filepath.Base(symlinkPath), ".md")
	}
	rel, err := filepath.Rel(root.dir, symlinkPath)
	if err != nil {
		return ""
	}
	if i := strings.IndexRune(rel, filepath.Separator); i >= 0 {
		return rel[:i]
	}
	return rel
}

// findSymlinks returns every symlink found anywhere under the given roots.
func findSymlinks(roots []string) []string {
	var links []string
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				links = append(links, p)
			}
			return nil
		})
	}
	return links
}

// checkAntigravity reproduces healthcheck section 12: when the agy CLI is
// present, its endpoint/data/MCP-config invariants hold and no symlinks lurk
// under ~/.gemini/config (BUG-100). Absent agy → the whole section SKIPs.
func checkAntigravity(sys *System, rep *Report) {
	rep.Section("Antigravity CLI health")
	if !sys.has("agy") {
		rep.Skip("agy not in PATH")
		return
	}

	endpoint := sys.env("ANTIGRAVITY_ENDPOINT", "https://cloudcode-pa.googleapis.com")
	if endpoint == "https://cloudcode-pa.googleapis.com" {
		rep.Pass("ANTIGRAVITY_ENDPOINT set to production")
	} else {
		rep.Fail("ANTIGRAVITY_ENDPOINT is not production: " + endpoint)
	}

	agyData := sys.env("AGY_APP_DATA", filepath.Join(sys.home(), ".gemini", "antigravity-cli"))
	// filepath.IsAbs, not HasPrefix(_, "/"): an absolute Windows path
	// (C:\Users\...\.gemini\antigravity-cli) is not '/'-rooted, so the POSIX-only
	// check false-FAILed it whenever agy was on PATH on Windows (#691 / C20).
	if filepath.IsAbs(agyData) {
		rep.Pass("AGY_APP_DATA is absolute")
	} else {
		rep.Fail("AGY_APP_DATA is relative or unset: " + agyData)
	}

	geminiHome := sys.env("GEMINI_HOME", filepath.Join(sys.home(), ".gemini"))
	configDir := filepath.Join(geminiHome, "config")
	master := filepath.Join(configDir, "mcp_config.json")
	switch {
	case !pathExists(master):
		rep.Fail("master mcp_config.json missing at " + master + " (run setup)")
	case isSymlink(master):
		// BUG-100 regression
		rep.Fail("master mcp_config.json is a symlink (recursion risk) (run setup)")
	case isEmptyFile(master):
		// agy writes an empty file on its first run, before setup has written
		// the master (measured on the Mac bring-up, #2013). Setup overwrites
		// it, so the remedy is the same as for a missing one.
		rep.Fail("master mcp_config.json at " + master + " is empty — agy creates it on first run (run setup)")
	case !isValidJSON(master):
		rep.Fail("master mcp_config.json at " + master + " is invalid JSON (run setup)")
	default:
		rep.Pass("master mcp_config.json is a real file with valid JSON")
	}

	if isDir(configDir) {
		if links := findSymlinks([]string{configDir}); len(links) > 0 {
			// BUG-100 regression
			rep.Fail("symlinks found under ~/.gemini/config/ (recursion risk): " + strings.Join(links, ", "))
		} else {
			// BUG-100 guard
			rep.Pass("no symlinks under ~/.gemini/config/")
		}
	}
}

// matchPin compares an installed version against a versions.conf pin: empty pin
// → SKIP, equal → PASS, drift → WARN (never a FAIL — a pinned-tool drift is
// advisory, exactly as healthcheck treated it).
// semverOf is the doctor-side face of tools.ProbeVersion: the first semver in
// `<name> --version`, through the System seam so tests inject the banner.
// trailingVersion (last token of the first line) is what reported "locked."
// for opencode on the Windows work box (AI-034/#1294); "unknown" when the
// tool prints no version at all, so the report line stays readable.
func semverOf(sys *System, name string) string {
	v := tools.ProbeVersion(name, func(n string, args ...string) ([]byte, error) {
		out, err := sys.CommandOutput(n, args...)
		return []byte(out), err
	})
	if v == "" {
		return "unknown"
	}
	return v
}

func matchPin(rep *Report, tool, installed, pin string) {
	matchPinFrom(rep, tool, installed, pin, "versions.conf")
}

// matchPinFrom is matchPin with the pin's source named: packages.json is the
// SSOT for catalog tools (ADR-036), versions.conf for the rest.
//
// Exact match by design: for the tools that call this (golangci-lint, age),
// nothing reconciles them toward the pin on a floor policy, so any drift —
// ahead or behind — is worth a WARN. golangci-lint in particular needs this:
// CI lints at the exact pinned version, so a locally-newer binary still
// diverges from CI (BUG-071).
func matchPinFrom(rep *Report, tool, installed, pin, source string) {
	switch {
	case pin == "":
		rep.Skip(tool + " version not pinned in " + source + " — match not verified")
	case installed == pin:
		rep.Pass(fmt.Sprintf("%s version matches %s (%s)", tool, source, pin))
	default:
		rep.Warn(fmt.Sprintf("%s version drift: installed=%s pinned=%s (%s)", tool, installed, pin, source))
	}
}

func matchPinFloor(rep *Report, tool, installed, pin string) {
	matchPinFloorFrom(rep, tool, installed, pin, "versions.conf")
}

// matchPinFloorFrom is matchPinFrom for tools whose installer reconciles on a
// floor policy — `dotf tools install`'s decideAction (install.go, "the pin is
// a MINIMUM, not an exact match... an exact-match reconcile would wrongly
// downgrade") and pi's own setup-{linux,windows} install block (REFACTOR-013).
// Doctor must agree with the installer about what the pin means: an exact-match
// WARN here fires every time the tool's own upstream ships a patch between two
// pin bumps, which the installer will never revert — chasing it with a pin bump
// is a treadmill, not a signal (observed: PI_VERSION bumped to "the version
// actually installed" in f1c20cd, then drifted again).
func matchPinFloorFrom(rep *Report, tool, installed, pin, source string) {
	switch {
	case pin == "":
		rep.Skip(tool + " version not pinned in " + source + " — match not verified")
	case installed == pin:
		rep.Pass(fmt.Sprintf("%s version matches %s (%s)", tool, source, pin))
	case atLeast(installed, pin):
		rep.Pass(fmt.Sprintf("%s %s meets the %s pin %s", tool, installed, source, pin))
	default:
		rep.Warn(fmt.Sprintf("%s version drift: installed=%s pinned=%s (%s)", tool, installed, pin, source))
	}
}

// checkDeployDrift ports the standalone diff-check twin (healthcheck §11): for
// every git-tracked file in the deploy-dir set (harness.IsDeployDirPath, the
// same set the mirror copies), byte-compare the repo copy against the deployed
// ~/.dotfiles copy. Drift means the repo moved and nothing re-mirrored it, so
// every shell still reads the stale deploy-dir copy. A
// missing repo / deploy-dir / non-git repo is a SKIP (the shell twin's exit 2),
// because `dotf doctor` legitimately runs where one side is absent (CI, fresh box).
func checkDeployDrift(sys *System, cfg *Config, rep *Report, fix bool) {
	rep.Section("Repo↔deploy-dir drift")

	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — set DOTFILES_REPO_DIR or run from a checkout")
		return
	}
	deploy := cfg.DotfilesDir
	if !isDir(deploy) {
		rep.Skip("deploy-dir absent: " + deploy + " (run setup)")
		return
	}
	// pathExists, not isDir: in a linked worktree .git is a file naming the
	// gitdir, and the check skipped every worktree as "not a git repo".
	if !pathExists(filepath.Join(repo, ".git")) {
		rep.Skip("not a git repo: " + repo)
		return
	}

	out, err := sys.CommandOutput("git", "-C", repo, "ls-files")
	if err != nil {
		rep.Warn("git ls-files failed in " + repo + ": " + err.Error())
		return
	}

	drift, checked := 0, 0
	for _, rel := range strings.Split(out, "\n") {
		rel = strings.TrimSpace(rel)
		if rel == "" || !harness.IsDeployDirPath(rel) {
			continue
		}
		repoFile := filepath.Join(repo, filepath.FromSlash(rel))
		deployFile := filepath.Join(deploy, filepath.FromSlash(rel))
		// Compare only files present on BOTH sides — a repo file not yet deployed
		// (or a deploy-only leftover) is not "drift", matching diff-check's
		// existence guards.
		if !pathExists(repoFile) || !pathExists(deployFile) {
			continue
		}
		checked++
		if !filesEqual(repoFile, deployFile) {
			rep.Fail("drift: " + rel + " — repo differs from deploy-dir (run `dotf converge` to refresh it)")
			drift++
		}
	}
	if drift == 0 {
		rep.Pass(fmt.Sprintf("repo and deploy-dir agree (%d managed files checked)", checked))
	}
	checkDeployDirLeftovers(sys, rep, repo, deploy, fix)
}

// checkDeployDirLeftovers is the other half of the drift check, the one it
// skips by comparing only files on both sides: a file the checkout deleted that
// the deploy dir still has (#2266). harness.Mirror prunes these on every setup
// and converge; this reports them on a box that has not mirrored since, and
// --fix removes them through the same function. An orphan git never tracked is
// only named, because absence from the checkout does not make it garbage (#802).
func checkDeployDirLeftovers(sys *System, rep *Report, repo, deploy string, fix bool) {
	git := func(dir string, args ...string) (string, error) {
		return sys.CommandOutput("git", append([]string{"-C", dir}, args...)...)
	}
	o, err := harness.ScanOrphans(repo, deploy, git)
	if err != nil {
		rep.Warn("deploy-dir leftover scan failed: " + err.Error())
		return
	}
	for _, rel := range o.Unknown {
		rep.Warn(rel + " is in the deploy dir but not in the checkout, and git history does not show it deleted — left in place; move it out of " + deploy + " if it is yours")
	}
	if o.Skipped != "" {
		rep.Warn("deploy-dir leftovers not classified: " + o.Skipped)
	}
	for _, rel := range o.Unreadable {
		rep.Warn(rel + " in the deploy dir could not be read, so it was not checked for leftovers")
	}
	switch {
	case len(o.Deleted) == 0:
		if len(o.Unknown)+len(o.Unreadable) == 0 {
			rep.Pass("deploy dir holds no leftovers of files the checkout deleted")
		}
	case !fix:
		for _, rel := range o.Deleted {
			rep.Fail("leftover: " + rel + " — deleted from the checkout, still in the deploy dir (run: dotf doctor --fix, or dotf converge)")
		}
	default:
		if err := harness.PruneOrphans(deploy, o.Deleted); err != nil {
			rep.Fail("failed to prune deploy-dir leftovers: " + err.Error())
			return
		}
		for _, rel := range o.Deleted {
			rep.Fix("pruned deploy-dir leftover: " + rel)
		}
	}
}

// resolveRepoDir locates the dotfiles checkout: DOTFILES_REPO_DIR when it points
// at a real directory, else the git root walked up from the current directory.
// "" means neither resolved (caller SKIPs). The shell twin used
// DOTFILES_REPO_DIR → parent-of-script; Go has no script dir, so it walks up.
func resolveRepoDir(sys *System) string {
	if r := sys.Getenv("DOTFILES_REPO_DIR"); r != "" && isDir(r) {
		return r
	}
	if wd, err := os.Getwd(); err == nil {
		if root, err := findRepoRoot(wd); err == nil {
			return root
		}
	}
	return ""
}
