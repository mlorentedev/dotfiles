package doctor

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
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
	rep.Section("Generated path files (ADR-025)")
	if cfg.ContractPath == "" {
		rep.Skip("env-contract.json not found — path-file drift check skipped")
		return
	}
	home := sys.home()
	out := envpkg.DefaultOutput(runtime.GOOS, sys.env("DOTFILES_DIR", filepath.Join(home, ".dotfiles")))
	if !pathExists(out) {
		rep.Warn(filepath.Base(out) + " not generated — run `dotf env generate`")
		return
	}
	res, err := envpkg.Generate(envpkg.Options{
		ContractPath: cfg.ContractPath,
		MachinePath:  envpkg.MachinePath(home),
		GOOS:         runtime.GOOS,
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
// and no orphan .age file lacks a registry entry.
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
func checkSecrets(sys *System, cfg *Config, rep *Report) {
	rep.Section("Secrets integrity")
	secretsDir := filepath.Join(cfg.DotfilesDir, "sensitive")

	reg, err := loadRegistry(cfg)
	if err != nil {
		rep.Fail("secrets/registry.yaml not found or invalid")
		return
	}
	rep.Pass("secrets/registry.yaml exists")

	referenced := map[string]bool{}
	migrated := 0
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
			migrated++
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
				rep.Pass(fmt.Sprintf("%s -> %s (file-authority on disk)", display, e.Dest))
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

	ageFiles, _ := filepath.Glob(filepath.Join(secretsDir, "*.secret.age"))
	var unreferenced []string
	for _, f := range ageFiles {
		base := strings.TrimSuffix(filepath.Base(f), ".secret.age")
		if !referenced[base] {
			unreferenced = append(unreferenced, base)
		}
	}
	reportUnreferencedBlobs(rep, unreferenced, migrated)
}

// reportUnreferencedBlobs decides the severity of age blobs no registry entry
// claims. The answer depends on whether migration has happened at all.
//
// With no bw-backed secrets, an unclaimed blob is unambiguously a leftover: FAIL,
// naming it, as before. Once secrets have migrated the claim stops being decidable
// — `migrate` drops the `age:` line (registry_write.go), so the surviving DR-floor
// blob of a migrated secret is indistinguishable from a genuine leftover, and the
// names cannot be correlated either (OPENAI_API_KEY's blob is chatgpt.api-key).
// Those blobs ARE the ADR-028 floor for the live tier, so calling them orphans
// invites deleting the recovery path for exactly the secrets that just moved.
//
// So it degrades to one WARN that says the set is unclassifiable, rather than N
// FAILs asserting something it cannot know. It regains its teeth — per blob, as a
// failure — once the registry records a DR pointer for migrated secrets (#971).
func reportUnreferencedBlobs(rep *Report, unreferenced []string, migrated int) {
	if len(unreferenced) == 0 {
		return
	}
	if migrated == 0 {
		for _, base := range unreferenced {
			rep.Fail("orphan: " + base + ".secret.age (no registry entry)")
		}
		return
	}
	rep.Warn(fmt.Sprintf(
		"%d age blob(s) claimed by no registry entry, and %d secret(s) have migrated to bw — "+
			"migrate drops the `age:` pointer, so a surviving DR floor cannot be told from a "+
			"leftover here. Do not bulk-delete: some are the ADR-028 floor. Tracked as #971",
		len(unreferenced), migrated))
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

// checkTmux reproduces healthcheck section 9: tmux is installed and ~/.tmux.conf
// matches the repo source (copy-deploy drift check, ADR-012).
func checkTmux(sys *System, cfg *Config, rep *Report) {
	rep.Section("tmux")
	if sys.GOOS == "windows" {
		rep.Skip("tmux (Linux-only by design; use WSL if needed)")
		return
	}
	if !sys.has("tmux") {
		rep.Fail("tmux not installed (run: sudo apt install -y tmux)")
		return
	}
	ver := "unknown" // tmux uses `-V`, not the conventional `--version`
	if v, err := sys.CommandOutput("tmux", "-V"); err == nil {
		ver = strings.TrimSpace(v)
	}
	rep.Pass("tmux installed: " + ver)

	src := filepath.Join(cfg.DotfilesDir, "tmux.conf")
	dst := filepath.Join(sys.home(), ".tmux.conf")
	switch {
	case pathExists(src):
		switch {
		case !pathExists(dst):
			rep.Fail(".tmux.conf missing at " + dst + " (run setup)")
		case isSymlink(dst):
			rep.Fail(".tmux.conf is a symlink (expected a regular copy — run setup)")
		case filesEqual(src, dst):
			rep.Pass(".tmux.conf deployed (matches repo)")
		default:
			rep.Fail(".tmux.conf has drifted from " + src + " (edit in repo + run setup)")
		}
	case pathExists(dst):
		rep.Pass(".tmux.conf exists (source unavailable for drift check)")
	default:
		rep.Fail(".tmux.conf missing (run setup)")
	}
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
		ver := trailingVersion(sys, "opencode", "--version")
		rep.Pass("opencode in PATH: " + ver)
		matchPinFrom(rep, "opencode", ver, catalogPin(sys, cfg, "opencode"), "packages.json")
	case isExecFile(opencodeBin):
		// The retired curl-script channel (ADR-036) left a copy behind and
		// nothing on PATH resolves: the rc files no longer add ~/.opencode/bin.
		rep.Fail("opencode not on PATH; a legacy curl-script copy sits at " + opencodeBin + " — run `dotf tools install opencode` and delete the legacy copy (ADR-036)")
	default:
		rep.Fail("opencode missing (run `dotf tools install opencode`)")
	}

	// opencode config + $schema.
	cfgPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	switch {
	case !pathExists(cfgPath):
		rep.Fail("opencode.jsonc missing: " + cfgPath + " (run setup)")
	case fileContains(cfgPath, `"$schema":`):
		rep.Pass("opencode.jsonc deployed with $schema declaration")
	default:
		rep.Fail("opencode.jsonc missing $schema declaration (re-run setup to redeploy)")
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
		ver := trailingVersion(sys, "pi", "--version")
		rep.Pass("pi in PATH: " + ver)
		matchPin(rep, "pi", ver, cfg.Versions["PI_VERSION"])
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
// (file/source/requires_command) — the four instruction files
// compile-harness.sh --deploy copies verbatim (HARNESS-058/#828). Kept here
// rather than parsed from the manifest because the doctor binary must run
// with no repo/vault present at all; the manifest is only reachable once a
// repo IS found, at which point this list and the manifest are asserted in
// sync by TestCheckInstructionDrift_MatchesManifest.
var deployedInstructionTargets = []struct{ homeRel, repoRel, requiresCommand string }{
	{".claude/CLAUDE.md", "ai/claude/CLAUDE.md", ""},
	{".config/opencode/AGENTS.md", "AGENTS.md", ""},
	{".pi/agent/AGENTS.md", "AGENTS.md", ""},
	{".copilot/copilot-instructions.md", "ai/copilot/copilot-instructions.md", "copilot"},
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
	home := sys.home()
	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — instruction-file drift check skipped")
		return
	}
	checked, drift := 0, 0
	for _, tgt := range deployedInstructionTargets {
		if tgt.requiresCommand != "" && !sys.has(tgt.requiresCommand) {
			continue
		}
		deployed := filepath.Join(home, filepath.FromSlash(tgt.homeRel))
		source := filepath.Join(repo, filepath.FromSlash(tgt.repoRel))
		if !pathExists(deployed) || !pathExists(source) {
			continue // not deployed here — not drift
		}
		dc, err1 := os.ReadFile(deployed)
		sc, err2 := os.ReadFile(source)
		if err1 != nil || err2 != nil {
			continue
		}
		checked++
		if stripHarnessRegions(string(dc)) != stripHarnessRegions(string(sc)) {
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

// Harness marker-region delimiters, mirrored from scripts/compile-harness.sh's
// BEGIN_PREFIX/END_MARKER and AGENT_BEGIN_PREFIX/AGENT_END_MARKER constants. A
// drift test (TestHarnessMarkerConstants) asserts they stay byte-identical.
const (
	harnessBeginPrefix       = "<!-- BEGIN HARNESS GENERATED"
	harnessEndMarker         = "<!-- END HARNESS GENERATED -->"
	agentPresenceBeginPrefix = "<!-- BEGIN HARNESS AGENT-PRESENCE"
	agentPresenceEndMarker   = "<!-- END HARNESS AGENT-PRESENCE -->"
)

// stripHarnessRegions removes every harness-managed marker region (both the
// GENERATED and AGENT-PRESENCE kinds) from content, mirroring
// compile-harness.sh's region_content in reverse (strip instead of extract).
//
// Also drops the single blank line immediately preceding a BEGIN marker:
// both inject_agent_presence and replace_region's append branch write a
// region as "\n" + BEGIN + body + END + "\n" (compile-harness.sh), so an
// appended region always leaves that blank separator behind in the deployed
// file with nothing to match it in the un-appended repo source. Without
// dropping it, checkInstructionDrift reported drift on every target
// immediately after a clean --deploy — caught in review before merge.
//
// Line endings are normalised first. A deployed copy written by a Windows
// tool arrives CRLF while the repo source is LF (`.gitattributes` `*.md
// eol=lf`), and with a bare "\n" split every line kept a trailing "\r": the
// END marker never matched, so the strip swallowed the rest of the file, and
// every other line differed anyway — a drift FAIL no redeploy could clear
// (WIN-008/#1289). EOL is not content; the writer is fixed separately, and
// this keeps the comparator honest about the next writer that is not.
func stripHarnessRegions(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	skip, endMarker := false, ""
	dropPrecedingBlank := func() {
		if n := len(out); n > 0 && out[n-1] == "" {
			out = out[:n-1]
		}
	}
	for _, l := range lines {
		if skip {
			if l == endMarker {
				skip = false
			}
			continue
		}
		switch {
		case strings.HasPrefix(l, harnessBeginPrefix):
			dropPrecedingBlank()
			skip, endMarker = true, harnessEndMarker
		case strings.HasPrefix(l, agentPresenceBeginPrefix):
			dropPrecedingBlank()
			skip, endMarker = true, agentPresenceEndMarker
		default:
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

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
		rep.Skip("harness drift gate is Linux-only; Windows deploys committed records, no --check port yet (CLI-035)")
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
	rep.Fail("deployed skill path(s) are symlinks (must be hard copies — BUG-100):")
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
		rep.Fail("master mcp_config.json is a symlink (BUG-100 regression — recursion risk)")
	case !isValidJSON(master):
		rep.Fail("master mcp_config.json at " + master + " is invalid JSON")
	default:
		rep.Pass("master mcp_config.json is a real file with valid JSON")
	}

	if isDir(configDir) {
		if links := findSymlinks([]string{configDir}); len(links) > 0 {
			rep.Fail("symlinks found under ~/.gemini/config/ (BUG-100 regression): " + strings.Join(links, ", "))
		} else {
			rep.Pass("no symlinks under ~/.gemini/config/ (BUG-100 guard)")
		}
	}
}

// trailingVersion returns the last whitespace field of the first line of
// `name <arg>` output (the awk '{print $NF}' idiom), or "unknown" on error.
func trailingVersion(sys *System, name, arg string) string {
	out, err := sys.CommandOutput(name, arg)
	if err != nil {
		return "unknown"
	}
	first := out
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		first = out[:i]
	}
	fields := strings.Fields(first)
	if len(fields) == 0 {
		return "unknown"
	}
	return fields[len(fields)-1]
}

// matchPin compares an installed version against a versions.conf pin: empty pin
// → SKIP, equal → PASS, drift → WARN (never a FAIL — a pinned-tool drift is
// advisory, exactly as healthcheck treated it).
func matchPin(rep *Report, tool, installed, pin string) {
	matchPinFrom(rep, tool, installed, pin, "versions.conf")
}

// matchPinFrom is matchPin with the pin's source named: packages.json is the
// SSOT for catalog tools (ADR-036), versions.conf for the rest.
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

// checkDeployDrift ports the standalone diff-check twin (healthcheck §11): for
// every git-tracked file under the managed allowlist, byte-compare the repo copy
// against the deployed ~/.dotfiles copy. Drift means the repo was edited without
// re-running setup, so every shell still reads the stale deploy-dir copy. A
// missing repo / deploy-dir / non-git repo is a SKIP (the shell twin's exit 2),
// because `dotf doctor` legitimately runs where one side is absent (CI, fresh box).
func checkDeployDrift(sys *System, cfg *Config, rep *Report) {
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
	if !isDir(filepath.Join(repo, ".git")) {
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
		if rel == "" || !isManagedDeployPath(rel) {
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
			rep.Fail("drift: " + rel + " — repo differs from deploy-dir (run setup to refresh ~/.dotfiles)")
			drift++
		}
	}
	if drift == 0 {
		rep.Pass(fmt.Sprintf("repo and deploy-dir agree (%d managed files checked)", checked))
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

// isManagedDeployPath reports whether a git-tracked repo path is one setup copies
// into the deploy-dir. It MUST mirror the copy block in setup-linux.sh (and the
// Windows guards in setup-windows.ps1); diff-check kept them in sync by comment,
// and this port inherits that coupling (CLI-019 follow-up: a grep-guard test).
func isManagedDeployPath(rel string) bool {
	switch rel {
	case "versions.conf", ".zshrc", ".bashrc", ".profile", ".gitconfig", "tmux.conf":
		return true
	}
	for _, prefix := range []string{".zsh/", "ssh/", "scripts/", "sensitive/"} {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}
