package doctor

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

// checkSymlinks reproduces healthcheck section 4: the dotfiles symlinks resolve,
// plus the two claude-mem install-state assertions (BUG-014 membership, BUG-015
// hook-path resolution). A real-file-where-a-symlink-was-expected still PASSes
// (the deploy strategy moved to copy for some paths, ADR-012).
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
	checkClaudeMem(sys, rep)
}

// checkClaudeMem ports the BUG-014 + BUG-015 assertions: claude-mem is a
// registered plugin and its hook path resolves to a runnable plugin dir. Both
// SKIP when Claude Code never ran (no installed_plugins.json), which is the
// expected state on a clean box or CI runner (WIN-004).
func checkClaudeMem(sys *System, rep *Report) {
	installedJSON := filepath.Join(sys.home(), ".claude", "plugins", "installed_plugins.json")

	if !pathExists(installedJSON) {
		rep.Skip("claude-mem install state — installed_plugins.json missing (Claude Code never ran)")
		rep.Skip("claude-mem hook path — installed_plugins.json missing (probe n/a)")
		return
	}

	if fileContains(installedJSON, "claude-mem@thedotmack") {
		rep.Pass("claude-mem@thedotmack installed (BUG-014)")
	} else {
		rep.Fail("claude-mem@thedotmack NOT in installed_plugins.json — re-run setup (BUG-014)")
	}

	if p := resolveClaudeMemHook(sys); p != "" {
		rep.Pass("claude-mem hook path resolves to: " + p + " (BUG-015)")
	} else {
		rep.Fail("claude-mem hook path resolution FAILED — run claude-mem-heal.sh (BUG-015)")
	}
}

// resolveClaudeMemHook reproduces the healthcheck.sh path-resolution cascade:
// PLUGIN_ROOT override, then the newest versioned cache dir, then the
// marketplace plugin dir. The first candidate carrying both runner scripts
// wins; "" means none did.
func resolveClaudeMemHook(sys *System) string {
	c := sys.env("CLAUDE_CONFIG_DIR", filepath.Join(sys.home(), ".claude"))

	var candidates []string
	if pr := sys.Getenv("PLUGIN_ROOT"); pr != "" {
		candidates = append(candidates, pr)
	}
	candidates = append(candidates, newestVersionDirs(filepath.Join(c, "plugins", "cache", "thedotmack", "claude-mem"))...)
	candidates = append(candidates, filepath.Join(c, "plugins", "marketplaces", "thedotmack-claude-mem", "plugin"))

	for _, r := range candidates {
		r = strings.TrimRight(r, "/")
		q := r
		if isDir(filepath.Join(r, "plugin", "scripts")) {
			q = filepath.Join(r, "plugin")
		}
		if pathExists(filepath.Join(q, "scripts", "bun-runner.js")) &&
			pathExists(filepath.Join(q, "scripts", "worker-service.cjs")) {
			return q
		}
	}
	return ""
}

// newestVersionDirs returns the numeric-prefixed subdirectories of parent,
// newest (by mtime) first — the `ls -dt .../[0-9]*/` the twin used to prefer
// the most recently installed plugin version.
func newestVersionDirs(parent string) []string {
	matches, _ := filepath.Glob(filepath.Join(parent, "[0-9]*"))
	type dirMod struct {
		path string
		mod  int64
	}
	var dirs []dirMod
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil || !fi.IsDir() {
			continue
		}
		dirs = append(dirs, dirMod{m, fi.ModTime().UnixNano()})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].mod > dirs[j].mod })
	out := make([]string, len(dirs))
	for i, d := range dirs {
		out[i] = d.path
	}
	return out
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

// checkSecrets reproduces healthcheck section 8: every env-mapping.conf entry
// resolves to an existing *.secret.age, and no orphan .age file lacks a mapping.
func checkSecrets(sys *System, cfg *Config, rep *Report) {
	rep.Section("Secrets integrity")
	secretsDir := filepath.Join(cfg.DotfilesDir, "sensitive")
	mapping := filepath.Join(secretsDir, "env-mapping.conf")

	raw, err := os.ReadFile(mapping)
	if err != nil {
		rep.Fail("env-mapping.conf not found")
		return
	}
	rep.Pass("env-mapping.conf exists")

	referenced := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		varName, val, _ := strings.Cut(line, "=")
		varName = strings.TrimSpace(varName)
		val = strings.TrimSpace(val)

		var fname, display string
		if strings.HasPrefix(varName, "@") {
			fname, _, _ = strings.Cut(val, ">") // @VAR=file>dest  → file
			display = strings.TrimPrefix(varName, "@") + " [file]"
		} else {
			fname = val
			display = varName
		}
		referenced[fname] = true

		if pathExists(filepath.Join(secretsDir, fname+".secret.age")) {
			rep.Pass(fmt.Sprintf("%s -> %s.secret.age", display, fname))
		} else {
			rep.Fail(fmt.Sprintf("%s -> %s.secret.age (missing)", display, fname))
		}
	}

	ageFiles, _ := filepath.Glob(filepath.Join(secretsDir, "*.secret.age"))
	for _, f := range ageFiles {
		base := strings.TrimSuffix(filepath.Base(f), ".secret.age")
		if !referenced[base] {
			rep.Fail("orphan: " + base + ".secret.age (no mapping)")
		}
	}
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
		matchPin(rep, "opencode", ver, cfg.Versions["OPENCODE_VERSION"])
	case isExecFile(opencodeBin):
		rep.Fail("opencode binary exists at " + opencodeBin + " but not in PATH (reload shell)")
	default:
		rep.Fail("opencode binary missing (run setup)")
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
}

// checkHarnessDrift reproduces healthcheck section 11's harness/skill-record
// drift gate and the symlink-free-skills invariant. The repo↔deploy-dir drift
// half of §11 (the standalone diff-check twin) is a separate section,
// checkDeployDrift (CLI-019).
func checkHarnessDrift(sys *System, cfg *Config, rep *Report) {
	rep.Section("Harness + skill drift")
	scriptsDir := filepath.Join(cfg.DotfilesDir, "scripts")

	compile := filepath.Join(scriptsDir, "compile-harness.sh")
	switch {
	case !isExecFile(compile):
		rep.Skip("compile-harness.sh not found at " + compile)
	default:
		if _, err := sys.CommandOutput("bash", compile, "--check"); err == nil {
			rep.Pass("harness blocks + skill records match their source-of-record (no drift)")
		} else {
			rep.Fail("harness/skill drift (run: compile-harness.sh --refresh, then re-deploy)")
		}
	}

	// Deployed skill paths must be regular copies, never symlinks (BUG-100).
	home := sys.home()
	skillDirs := []string{
		filepath.Join(home, ".claude", "skills"),
		filepath.Join(home, ".config", "opencode", "commands"),
		filepath.Join(home, ".gemini", "skills"),
		filepath.Join(home, ".gemini", "prompts"),
	}
	var present []string
	for _, d := range skillDirs {
		if isDir(d) {
			present = append(present, d)
		}
	}
	if len(present) == 0 {
		rep.Skip("no deployed skill paths found (run setup to deploy skills)")
		return
	}
	links := findSymlinks(present)
	if len(links) == 0 {
		rep.Pass("deployed skills are regular copies (no symlinks in skill paths)")
		return
	}
	rep.Fail("deployed skill path(s) are symlinks (must be hard copies — BUG-100):")
	for _, l := range links {
		rep.Fail("  " + l)
	}
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
	if strings.HasPrefix(agyData, "/") {
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
	switch {
	case pin == "":
		rep.Skip(tool + " version not pinned in versions.conf — match not verified")
	case installed == pin:
		rep.Pass(fmt.Sprintf("%s version matches versions.conf (%s)", tool, pin))
	default:
		rep.Warn(fmt.Sprintf("%s version drift: installed=%s pinned=%s", tool, installed, pin))
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
