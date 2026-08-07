---
id: "dotfiles-lessons"
type: lesson
status: active
tags: [project, dotfiles, lessons]
created: "2025-12-15"
owner: manu
---

# Dotfiles - Lessons Learned

> Patterns learned, mistakes avoided, and best practices discovered during development.
>
> **Protocol**: Update after every correction or important discovery.

---

## Format

```markdown
### [YYYY-MM-DD] Title

**Context**: What I was doing

**Problem**: What went wrong or what I discovered

**Solution**: How I resolved it

**Rule**: Pattern to follow going forward
```

---

## Entries

### [2025-12-15] echo -e breaks in zsh

**Context**: Shell scripts used `echo -e "\033[32mDone\033[0m"` for colored output

**Problem**: zsh does not support the `-e` flag on `echo`. Color codes printed as literal text instead of being interpreted.

**Solution**: Replace all `echo -e` with `printf '%b'` which handles escape sequences portably.

**Rule**: Never use `echo -e`. Always use `printf '%b' "..."` for colored or escaped output.

### [2025-12-15] &>/dev/null is not POSIX

**Context**: Scripts used `&>/dev/null` to suppress both stdout and stderr

**Problem**: `&>` is a bash-ism. Not POSIX-compliant and can fail in strict zsh configurations or when scripts are sourced in unexpected contexts.

**Solution**: Replace all `&>/dev/null` with `>/dev/null 2>&1` across every script.

**Rule**: Always use `>/dev/null 2>&1` for output suppression. Never use `&>`.

### [2025-12-15] ((count++)) exits with code 1 when count is 0

**Context**: Counter variables used `((count++))` inside scripts with `set -e`

**Problem**: In bash, `((0))` evaluates to false and returns exit code 1. When `count=0`, `((count++))` evaluates the pre-increment value (0), triggering `set -e` to abort the script.

**Solution**: Replace `((count++))` with `count=$((count + 1))`. The `$((...))` form always returns exit code 0.

**Rule**: Never use `((...))` for arithmetic that might evaluate to 0 under `set -e`. Use `var=$((expr))` assignment form instead.

### [2025-12-15] ${BASH_SOURCE[0]} is empty in zsh

**Context**: Scripts used `${BASH_SOURCE[0]}` to determine their own file path for relative directory resolution

**Problem**: zsh does not populate `BASH_SOURCE`. Scripts that relied on it for path resolution silently got empty strings, breaking relative path calculations.

**Solution**: Use `${BASH_SOURCE[0]:-$0}` which falls back to `$0` (populated by zsh) when `BASH_SOURCE` is empty.

**Rule**: Always use `${BASH_SOURCE[0]:-$0}` when a script needs to know its own path. Test path resolution in both bash and zsh.

### [2026-02-26] Claude Code auto-memory path encoding

**Context**: Needed to symlink auto-memory directories from dotfiles repo to `~/.claude/projects/*/memory/`

**Problem**: Claude Code encodes project paths by replacing all `/` with `-`. The path `/home/manu/Projects/dotfiles` becomes `-home-manu-Projects-dotfiles`. This encoding is deterministic but machine-specific (depends on username and directory structure).

**Solution**: Compute the encoded path dynamically in setup scripts: `printf '%s' "$project_path" | sed 's|/|-|g'`. Symlink from encoded path to vault directory (`~/Projects/knowledge/10_projects/<project>/memory/`).

**Rule**: When symlinking Claude Code internal paths, always compute the encoding at setup time — never hardcode the encoded path. Memory lives in the vault (not in the dotfiles repo) following the Neural Hive protocol: code in git, knowledge in vault.

### [2026-02-26] set -u requires ${1:-} for optional positional parameters

**Context**: Adding `set -euo pipefail` to all standalone scripts. Several scripts used bare `$1` in argument parsing (e.g., `case "$1" in`, `[[ -z "$1" ]]`).

**Problem**: `set -u` treats unset variables as errors. When a script is invoked with no arguments, `$1` is unbound and the script aborts before reaching its usage message. Tests that checked "shows usage with no args" all failed.

**Solution**: Use `${1:-}` (default to empty string) wherever a positional parameter might be unset. For scripts with a `case "$1"` dispatch, assign to a local variable first: `ACTION="${1:-}"` then `case "$ACTION" in`.

**Rule**: When using `set -u`, always guard optional positional parameters with `${1:-}` or `${1:-default}`. The `$#` check (`[ $# -ne 2 ]`) before access is also safe. Only bare `$1` inside a `while [[ $# -gt 0 ]]` loop is inherently safe.

### [2026-02-26] ${VAR:-fallback} pattern for sourced config files

**Context**: Tool versions (Java 21.0.4, Go 1.26.0, etc.) were hardcoded in both `.zshrc` and `.bashrc` — 12 duplicated strings with no single source of truth.

**Problem**: Updating a tool version required editing two files. A missed edit caused silent path mismatches. Also, on a fresh machine before running setup, `versions.conf` might not exist yet, and the shell would fail to set up PATH correctly.

**Solution**: Created `versions.conf` at repo root (KEY=VALUE, no export, no quotes — simultaneously sourceable by bash/zsh and parseable by PowerShell). Shell RCs source it with a guard (`[[ -f ... ]] && . ...`) then use `${JAVA_VERSION:-21.0.4}` to construct paths, providing a safe fallback if the file is missing.

**Rule**: When sourcing external config files that may not exist, always: (1) guard the source with a file-existence check, (2) use `${VAR:-default}` for every variable consumed from the config. This ensures the shell works on fresh machines before setup runs. The config file should use bare `KEY=VALUE` format (no `export`, no quotes) for maximum cross-tool compatibility.

### [2026-02-27] Always edit the repo copy, never the deployed system copy

**Context**: Adding an `obsidian --no-sandbox` alias to `.zshrc` and `.bashrc`

**Problem**: Edited `~/.dotfiles/.zshrc` and `~/.dotfiles/.bashrc` (the deployed system copy) first, instead of `~/Projects/dotfiles/.zshrc` and `~/Projects/dotfiles/.bashrc` (the canonical repo). Changes in `~/.dotfiles/` are not tracked by git and will be overwritten on next sync.

**Solution**: Always edit files in `~/Projects/dotfiles/` (the repo), then commit there. The sync/install script deploys repo → `~/.dotfiles/`.

**Rule**: The canonical source of truth for dotfiles is `~/Projects/dotfiles/`. The `~/.dotfiles/` directory is a deployment target — never edit it directly. Flow: edit repo → commit → sync to system.

### [2026-02-28] grep -c with 0 matches outputs "0" AND exits with code 1

**Context**: Writing `dedup_current_date()` in `knowledge-crystallize.sh`. Used `count=$(grep -c 'pattern' "$file" 2>/dev/null || echo 0)` to count matching lines.

**Problem**: `grep -c` with zero matches prints `"0"` to stdout AND exits with code 1 (not found). The `|| echo 0` then fires, appending a SECOND `"0"` to stdout. Result: `count="0\n0"` — a multi-line string. Downstream `[ "$count" -le 1 ]` fails with "integer expression expected".

**Solution**: Use assignment outside the subshell: `count=$(grep -c 'pattern' "$file" 2>/dev/null) || count=0`. The `$()` captures grep's stdout (`"0"`). If grep exits 1, `|| count=0` overwrites the variable. Either way, `count` is a single clean integer.

**Rule**: Never use `|| echo VALUE` to provide a default for a command that BOTH outputs and exits non-zero on "empty" results (grep -c, wc, etc.). Use `command || var=default` (assignment form) to overwrite after the subshell, not append.

### [2026-02-28] Claude Code path encoding: Linux vs Windows differ

**Context**: Implementing `--all` auto-discovery in `knowledge-crystallize.sh/.ps1`. Both scripts need to decode `~/.claude/projects/<encoded>/` back to real project paths.

**Problem**: The encoding differs by OS. Linux uses `tr '/' '-'` on the full absolute path (leading `/` becomes leading `-`). Windows uses `.Replace('\', '-').Replace(':', '')` which strips the drive colon (no leading dash). A decode strategy that works on one OS breaks on the other.

**Solution**: Two-stage decode in each script: (1) simple character substitution + `Test-Path`/`-d` existence check (handles 95% of cases where dir names have no dashes); (2) filesystem walk under `$HOME`/`$env:USERPROFILE` up to depth 5, encode each directory, compare (handles project names with dashes like `kasa-provisioner`). The walk is O(dirs under HOME, depth ≤ 5) — fast enough in practice.

**Rule**: When round-tripping Claude Code project paths: Linux encodes with leading `-` (from `/`), Windows strips `:` so no leading char. Keep OS-specific encode/decode functions separate. Always test with a project whose name contains a dash.

### [2026-02-28] bash set -e does not exit on [ with integer error when in && chain

**Context**: In `knowledge-crystallize.sh`, `[ "$count" -le 1 ] && return 0` was at the top of `dedup_current_date`. When `count="0\n0"` (bug above), `[` exited with code 2 (error).

**Problem**: Expected `set -euo pipefail` to abort the script on the `[` error. Instead, the script continued. Bash's `set -e` doesn't always trigger on `[` exits with code 2 inside `&&` chains — the `&&` compound absorbs the error differently than a simple command.

**Solution**: Fixed the root cause (`grep -c` bug). Also added `|| log_warning "..."` in `run_all` around `process_project` so one bad project can't kill the batch run.

**Rule**: Don't rely on `set -e` to catch every `[` evaluation error — especially inside `&&`/`||` compounds. Validate that numeric variables are actually integers before using them in arithmetic comparisons. Wrap batch loops with per-item `|| warn` to isolate failures.

### [2026-02-27] Claude Code SessionStart hook for vault health context

**Context**: `vault-health.sh` was created to report Obsidian vault health (orphans, unresolved links, frontmatter coverage) but required manual invocation.

**Problem**: Claude had no automatic awareness of vault health state at session start. Users had to remember to run the script and paste results.

**Solution**: Created `claude-session-start.sh` as a Claude Code `SessionStart` hook. The hook detects if CWD is inside an Obsidian vault (walks up directories looking for `.obsidian/`), runs `vault-health.sh` if found, and returns health summary as `additionalContext` via the hook JSON output format. Registered in `~/.claude/settings.json` under `hooks.SessionStart`.

**Rule**: Claude Code hooks live in `~/.claude/settings.json` (global scope). Scripts they invoke live in dotfiles (`~/.dotfiles/scripts/`). On new machines: (1) deploy dotfiles (gets the script), (2) add the hook entry to `~/.claude/settings.json`. The hook must tolerate Obsidian GUI being down (exit code 2 from vault-health.sh) and non-vault directories (exit 0 silently).

### [2026-03-10] Aider requires Python 3.12 — audioop removed in 3.13

**Context**: Installing aider-chat via `uv tool install` on a system with Python 3.13.

**Problem**: Python 3.13 removed the `audioop` stdlib module. Aider depends on `pydub` which imports `audioop` at module load time. The error manifests as `ModuleNotFoundError: No module named 'audioop'` on any aider command, even `--version`.

**Solution**: Pin Python 3.12 in the install command: `uv tool install --python 3.12 aider-chat`. Both `setup-linux.sh` and `setup-windows.ps1` use this pinned version.

**Rule**: When installing Python tools that depend on deprecated/removed stdlib modules, always pin the Python version explicitly in `uv tool install --python X.Y`. Check release notes for stdlib removals before upgrading the pinned version.

### [2026-03-12] Single-quoted shell strings prevent variable expansion in JSON

**Context**: `setup-linux.sh` built a JSON hook entry with `HOOK_ENTRY='{"command":"$HOME/.dotfiles/scripts/..."}'`. The literal `$HOME` was written into `settings.json`.

**Problem**: Claude Code reads the hook path as-is. The literal string `$HOME/...` is not a valid path, so the SessionStart hook silently failed on every fresh install since it was added.

**Solution**: Replaced string concatenation with `jq -n --arg cmd "$HOME/.dotfiles/scripts/claude-session-start.sh" '{"command":$cmd}'`. Shell expands `$HOME` in the argument, `jq` handles JSON escaping safely.

**Rule**: Never embed shell variables inside single-quoted JSON strings. Use `jq -n --arg` to build JSON with dynamic values — it handles both variable expansion and proper JSON escaping.

### [2026-03-12] grep -c '.' counts 1 on empty input (newline matches dot)

**Context**: `vault-health.sh` counted orphan/dead-end links with `echo "$output" | grep -c '.'`. When output was empty, the count should be 0.

**Problem**: `echo ""` emits a newline character. The regex `.` matches a newline in this pipeline context, so `grep -c '.'` returns 1 instead of 0. Every vault health check reported at least 1 orphan even on a clean vault.

**Solution**: Changed all `grep -c '.'` to `grep -c '[^[:space:]]'` which only counts lines with visible characters.

**Rule**: When counting non-empty lines from command output, use `grep -c '[^[:space:]]'` not `grep -c '.'`. The dot regex matches newlines from `echo` even when the content is empty.

### [2026-03-12] Plaintext secrets must never touch disk — pipe to age directly

**Context**: `secrets_add` and `secrets_rotate` in `load-secrets.sh` wrote the secret value to a plaintext file, encrypted it with age, then deleted the plaintext.

**Problem**: Between write and delete, the secret exists unencrypted on disk. On SSDs with wear leveling or systems with filesystem journaling, `rm` doesn't guarantee data erasure. An interrupted script leaves the plaintext file permanently.

**Solution**: Piped the value directly to age via stdin: `printf '%s' "$value" | age_encrypt "$encrypted_file" "$key_path"`. No temporary plaintext file is ever created.

**Rule**: Never write secrets to disk before encryption. Always pipe directly to the encryption tool's stdin. This eliminates both the crash-window vulnerability and the data-remanence risk.

### [2026-03-12] Uninitialized variable under set -u in conditional-only assignment

**Context**: `claude-session-start.sh` used `VAULT_NAME` which was only assigned inside a vault-detection `if` block.

**Problem**: Under `set -euo pipefail`, referencing `VAULT_NAME` after the conditional would abort the script with "unbound variable" when no vault was detected — the variable was never defined in that code path.

**Solution**: Initialize `VAULT_NAME=""` at the top of the script, before any conditionals. The variable is always bound regardless of which branch executes.

**Rule**: When using `set -u`, any variable assigned inside a conditional block must also be initialized at a wider scope (script top or function top). If only one branch of an `if/else` assigns the variable, the other branch leaves it unbound.

### [2026-03-12] \s is not POSIX — use `[[:space:]]` in bash regex

**Context**: `utils.sh` used `\s` in `[[ =~ ]]` regex patterns inside `load_env_file()` and `debug_print_env()`.

**Problem**: Bash's `[[ =~ ]]` uses POSIX Extended Regular Expressions (ERE), which do not define `\s` as a whitespace shorthand. `\s` is a Perl-Compatible Regular Expression (PCRE) extension. On some bash versions/platforms, `\s` silently fails to match whitespace, causing functions to behave incorrectly without any error message.

**Solution**: Replace all `\s` with the POSIX character class `[[:space:]]`, which is universally supported in ERE and functionally equivalent.

**Rule**: Never use `\s`, `\d`, `\w` or other PCRE shorthands in bash `[[ =~ ]]` or `grep -E` patterns. Use POSIX character classes instead: `[[:space:]]`, `[[:digit:]]`, `[[:alnum:]]`. Only `grep -P` (PCRE mode) supports the shorthand forms.

### [2026-03-12] Stray bare word causes silent set -e abort

**Context**: `github-secrets-manager.sh` had an accidental bare word `tmp` on its own line, immediately before a valid `tmp=$(create_temp_file "ssh_key")` assignment.

**Problem**: Under `set -e`, bash interprets a bare word as a command to execute. `tmp` is not a valid command, so it exits non-zero, and `set -e` aborts the script immediately. The SSH_PRIVATE_KEY_BASE64 branch would always fail silently with no error message — the "command not found" error went to stderr but was easily missed.

**Solution**: Delete the stray bare-word line. The variable assignment on the next line was already correct.

**Rule**: Under `set -e`, any bare word on its own line is treated as a command. A stale variable name from a copy-paste or incomplete edit becomes an invisible script-killer. Always review diffs for orphaned identifiers — they compile silently but crash at runtime.

### [2026-03-16] ShellCheck treats "# shellcheck" comments as directives

**Context**: `setup-linux.sh` had a comment `# shellcheck (shell script linter)` describing the tool being installed.

**Problem**: ShellCheck parses any comment starting with `# shellcheck` as a directive (like `# shellcheck disable=SC2012`). The parenthesized description text is not valid directive syntax, causing SC1073/SC1072 parse errors that halt further checking of the entire file.

**Solution**: Capitalize or rephrase the comment to avoid the `# shellcheck` prefix, e.g., `# ShellCheck (shell script linter)` or `# Install shellcheck`.

**Rule**: Never start a comment with the literal text `# shellcheck` unless it is an actual ShellCheck directive. The tool intercepts any comment matching that prefix. Use capitalization (`# ShellCheck`) or different phrasing to describe the tool in human-readable comments.

### [2026-03-25] Secrets mapping and file inventory must be reconciled automatically

**Context**: `sensitive/` contained 35 encrypted `.secret.age` files but only 17 had entries in `env-mapping.conf`. 14 were app-specific secrets (mlorentedev) that didn't belong in dotfiles at all.

**Problem**: `load-secrets.sh` silently skipped missing files (`file_exists || return 1`) and never checked for orphans. No automated way to detect mapping↔file drift. Secrets accumulated without cleanup.

**Solution**: (1) Added `log_warning` on missing files and orphan detection to `secrets_load()` — runs passively at every shell startup. (2) Added healthcheck section 8/8 that validates every mapping entry has a file and every file has a mapping. (3) Classified secrets: personal cross-machine credentials stay, app-specific envs move to project SOPS.

**Rule**: Any system with a mapping file (config ↔ resource) needs bidirectional reconciliation: mapping→resource (missing?) and resource→mapping (orphan?). Run both checks automatically at load time (passive warning) and in CI/healthcheck (active audit). Silent failure on missing resources is a bug, not a feature.

### [2026-03-18] cp fails when source and destination resolve to the same file via symlink

**Context**: `setup-linux.sh` used `safe_copy` (which wraps `cp`) to deploy `.gitconfig` from `$DOTFILES_DIR/.gitconfig` to `$HOME/.gitconfig`.

**Problem**: `~/.gitconfig` was already a symlink pointing to `~/.dotfiles/.gitconfig` (the same file). `cp` cannot copy a file over a symlink that resolves to the same source — it fails silently or with an error. This broke the "Setting up Git configuration" step on re-runs.

**Solution**: Replaced `safe_copy` with `ln -sf` for `.gitconfig`, consistent with how `.zshrc` and `.profile` are already deployed. `ln -sf` silently replaces any existing symlink. Updated `verify-setup.bats` to assert symlink behavior instead of regular file.

**Rule**: When deploying dotfiles, prefer `ln -sf` over `cp` for idempotency. A symlink can always be replaced atomically, while `cp` fails when the destination is a symlink to the source. Use the same deployment mechanism (symlink) for all managed dotfiles to avoid inconsistent behavior on re-runs.

### [2026-03-26] PSScriptAnalyzer fails on non-ASCII chars outside here-strings

**Context**: Added `-WorkSdk` mode to `init-project.ps1` with em dashes (`—`) in `Write-Host` strings and an arrow (`→`) in a `Write-Success` call. Also added an em dash in a comment line.

**Problem**: PSScriptAnalyzer flags any non-ASCII character that appears in `.ps1` files outside of here-strings (`@"..."@`). The lint-powershell CI job (`Invoke-ScriptAnalyzer -Severity Error,Warning`) caught it and exited 1. Syntax check (test 84) passed because PowerShell parses non-ASCII fine — only PSScriptAnalyzer rejects them.

**Solution**: Replace `—` with `-` and `→` with `->` in string literals, `Write-*` calls, and comments. Non-ASCII inside `@"..."@` here-strings is fine (pre-existing `→` on line 288 was already passing).

**Rule**: In `.ps1` files, keep all non-here-string code (comments, regular quoted strings, `Write-*` calls) to pure ASCII. Em dashes, arrows, and similar typography must live only inside `@"..."@` blocks (template content). When adding display text to a PowerShell script, use ASCII punctuation: `-` not `—`, `->` not `→`.

### [2026-03-29] File deployment requires delete-then-copy, not additive-only copy

**Context**: Skills ecosystem overhaul deleted 9 skill directories from the dotfiles repo. Setup scripts (`setup-linux.sh`, `setup-windows.ps1`) deployed skills by copying source to destination.

**Problem**: The copy loop only added new/updated files — it never removed entries at the destination that no longer existed in the source. After deleting 9 skills from the repo, those stale skill directories persisted at all deployment targets (`~/.claude/skills/`, Gemini prompts dir) indefinitely.

**Solution**: Changed both setup scripts to a three-step sync pattern: (1) enumerate destination entries, (2) delete any entry not present in the source, (3) copy all source entries. Linux uses `basename` + `[ -d ]` checks; Windows uses `$item.BaseName` + `Test-Path`.

**Rule**: Any file deployment pipeline that copies a directory must also handle deletions. Additive-only copy creates ghost artifacts that accumulate silently. The canonical pattern is: enumerate destination → diff against source → delete orphans → copy current. Apply this to skills, configs, prompts, or any mirrored directory.

### [2026-03-25] Config deployment guards vs tool installation guards

**Context**: Made Gemini config deployment conditional on `command -v gemini`. CI integration tests failed because the Docker container doesn't have gemini installed.

**Problem**: Conflated two concerns: (1) installing a tool's dependencies (needs the tool ecosystem present) and (2) deploying config files (just copying markdowns). The guard prevented harmless config from being deployed, breaking CI and also preventing pre-deployment on machines where the tool will be installed later.

**Solution**: Removed the CLI guard from config deployment. Config files are always deployed. Guards remain only around actual tool installation commands (e.g., `gh extension install github/gh-copilot`).

**Rule**: Separate "deploy config" from "install tool". Config file deployment (copying markdown, YAML, JSON) is always safe and should run unconditionally. Only guard commands that install binaries, extensions, or packages. A machine without the CLI benefits from pre-deployed config — it's ready when the tool arrives.

### [2026-05-08] Self-heal third-party plugin breakage at SessionStart

**Context**: `thedotmack/claude-mem` shipped v12.7.4 and v13.0.0 to the marketplace with two independent bugs that prevent the `mcp-search` MCP server and worker from starting on a fresh install: (1) `.mcp.json` embeds `${_R%/}` shell parameter expansion which Claude Code's MCP loader misreads as a missing env var (upstream #2385), and (2) v13.0.0's `bun.lock` and shipped `node_modules/` omit the `zod` dep declared in `package.json`, so the worker crashes with `Cannot find module 'zod/v3'`.

**Problem**: A manual edit of the cached `.mcp.json` plus a manual `npm install zod` fixes both, but `/plugin update` (or any reinstall) wipes the workaround. Documenting the manual steps in a vault troubleshooting note relies on me remembering to re-run them, which violates the "automate, don't instruct" standing order. Pinning to v10.6.3 loses upstream fixes and eventually stops working when the marketplace stops serving the old version. Forking the plugin is heavyweight for two trivial patches.

**Solution**: Encoded both fixes into `scripts/claude-mem-heal.sh` (POSIX `sh`, idempotent: `grep -F '${_R%/}'` to detect bug 1, `[ -d node_modules/zod ]` to detect bug 2; silent on healthy installs). Wired into `claude-session-start.sh` so it runs on every session start before vault detection. Heal output (when something was actually fixed) is surfaced via `additionalContext` so the user sees what was repaired. Iterates over all cached versions plus the marketplace fallback copy. Cost on healthy installs: <50ms (just two filesystem checks per cached version).

**Rule**: When a third-party tool ships a broken artifact and the fix is small and idempotent, encode it as a self-heal script wired into the relevant lifecycle hook (SessionStart, shell init, install verification). Three properties are required: idempotent (re-running on healed state is a no-op), silent on success (don't pollute every session with status spam), and surfacing only when it acts (so the user knows the workaround fired). Document the bug, the fix, and the retire-when criteria in `50-troubleshooting/`. Promote to a `00_meta/patterns/` pattern only after the second occurrence — one instance is a workaround, two is a pattern.


### [2026-05-11] tmux clipboard needs an external bridge — and that bridge is display-server-specific

**Context**: After enabling `set -g mouse on` and `mode-keys vi`, selections inside tmux still did not appear in the system clipboard. `Ctrl+V` outside tmux pasted stale content.

**Problem**: tmux's `copy-pipe` writes to its own internal buffer, not to the OS clipboard. The buffer is only exposed externally if you pipe the selection through an out-of-process tool. That tool is also display-server-specific:

- X11 → `xclip` (or `xsel`)
- Wayland → `wl-copy` (from `wl-clipboard`)
- macOS → `pbcopy` (stdlib, but irrelevant for this Linux-only setup)

`xclip` does not work on a pure Wayland session. `wl-copy` does not work on X11.

**Solution**: Pipe selections to `xclip` via `copy-pipe-and-cancel` and install `xclip` as a system package (added a warning block to `setup-linux.sh`, matching the existing `tmux` pattern — no sudo from the script). Bindings live in `tmux.conf`:

```tmux
bind -T copy-mode-vi y                 send-keys -X copy-pipe-and-cancel 'xclip -selection clipboard -in'
bind -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel 'xclip -selection clipboard -in'
```

**Rule**: When migrating display server, the clipboard bridge must change in lockstep:
- X11 → keep `xclip`.
- Wayland → swap to `wl-copy` in `tmux.conf` **and** install `wl-clipboard` instead of `xclip`.

Check current server with `echo $XDG_SESSION_TYPE`. Full operational walkthrough lives in [`runbooks/guide-tmux.md`](runbooks/guide-tmux.md) under the "Copy / paste" section.


### [2026-05-11] Editing a dotfile in the repo does not take effect until `setup-linux.sh` runs

**Context**: After editing `tmux.conf` in `~/Projects/dotfiles/` to add clipboard bindings (`copy-pipe-and-cancel` piped to `xclip`), tmux still behaved like the old config. Mouse selection produced nothing in the system clipboard.

**Problem**: The dotfiles repo uses a **two-tier deploy**, not a direct symlink to the repo:

```
~/Projects/dotfiles/<file>   ← canonical, git-tracked (you edit here)
        │ safe_copy in setup-linux.sh
        ▼
~/.dotfiles/<file>           ← deploy-dir intermediate (what the symlink resolves to)
        ▲ ln -sf in setup-linux.sh
~/.<file>                    ← active symlink in $HOME
```

The symlink `~/.tmux.conf → ~/.dotfiles/tmux.conf` was intact, but `~/.dotfiles/tmux.conf` was stale — `grep -c "copy-pipe-and-cancel" ~/.tmux.conf` returned `0` despite the repo file having the bindings. tmux was reading the old version of the file.

This applies to **every** dotfile deployed via `safe_copy` in `setup-linux.sh`: `tmux.conf`, `.gitconfig`, `.zshrc`, `.bashrc`, etc.

**Solution**:

```sh
cd ~/Projects/dotfiles
./setup-linux.sh                  # refreshes ~/.dotfiles/<file> from the repo
tmux source-file ~/.tmux.conf     # for tmux: reload running session
```

For other dotfiles, the appropriate reload command (e.g. `exec zsh`, `source ~/.zshrc`) after the redeploy.

**Rule**: After editing **any** file in `~/Projects/dotfiles/` that's tracked by `setup-linux.sh`, the change is not live until both:
1. `./setup-linux.sh` runs (refreshes the deploy-dir middle layer).
2. The relevant tool reloads its config.

Verification check before claiming the change is live:

```sh
grep -c "<new content>" ~/.<file>   # should match what's in the repo file
```

Cross-ref: [`runbooks/guide-tmux.md`](runbooks/guide-tmux.md) documents the same flow under "How the config gets deployed".

### [2026-05-12] Env-vs-disk drift after secret mutation
**Context:** While diagnosing dotfiles issue #7 (https://github.com/mlorentedev/dotfiles/issues/7) where `secrets_rotate` appeared to silently fail to update the encrypted file. Investigation showed the encrypted .age file WAS being updated correctly on disk (mtime + SHA256 confirmed change after rotate), but every consumer of the secret saw the old value.
**Problem:** load-secrets.sh exports `$VAR` once at shell startup. `secrets_rotate` updated the on-disk .age file but did NOT re-export `$VAR` in the current shell. Any subsequent read of `$VAR` (gh CLI, curl, scripts, even `secrets_show` without `--raw`) returned the cached old value, indistinguishable from a real failure. This wasted ~30min of debugging adding instrumentation to age_encrypt before realizing the encryption was fine. The issue was env-vs-disk drift between shell state and disk state, with no automatic reconciliation. Compounded by: `_secrets_sync_to_repo` silently no-op'd when `DOTFILES_REPO_DIR` was unset, so the user's primary verification step (git status in the repo) could miss real updates. Also the project lacked a `secrets_remove` function, so deletion was a manual three-step (edit mapping + rm .age + manual sync) that bypassed audit logging.
**Solution:** After any mutating secret operation (add/rotate/remove/add_file), auto-update the current shell so `$VAR` matches disk immediately — eliminate the manual `secrets_refresh` step that users forget. Concretely: rotate/add do `export_var "$var" "$value"`; remove does `unset_var`; add_file calls `secrets_refresh` (file deployment is non-trivial). Make `_secrets_sync_to_repo` warn loudly to stderr when no repo is resolved, and add an auto-detect default of `$HOME/Projects/dotfiles` when `DOTFILES_REPO_DIR` is unset. Add a missing `secrets_remove VAR_NAME [--yes]` that handles plain + file secrets, syncs deletions to repo, and updates audit log. Generalizes to: any system where the source of truth (disk) and the working copy (env, cache, deployed file) can diverge MUST either auto-reconcile after mutation OR loudly warn — silent drift is the worst failure mode because it produces false bug reports and wastes investigation time.
**Tags:** `#secrets` `#shell` `#env-drift` `#ssot` `#false-positive-debugging`

### [2026-05-12] git log --pretty=format: drops last commit silently in while-read pipelines
**Context:** Building scripts/changelog-gen.sh to bucket commits by Conventional Commit type. Read `git log --no-merges --pretty=format:'%cs|%h|%s'` via `while IFS='|' read -r date hash subject; do ...; done < <(git log ...)`. All commits except the very oldest one made it into the output. Tests caught it: the assertion that "feat(core): initial commit" appeared under "## Features" failed.</context>
<parameter name="problem">`git log --pretty=format:'...'` writes the format string between commits but does NOT terminate the LAST commit with a newline (matches the docstring of `format:` in `git log` man page). When piped into a `while read` loop, `read` returns non-zero on the unterminated final line, so the loop exits BEFORE processing it. Result: the oldest commit is silently dropped. Easy to miss — the output looks correct, just one entry shorter.</problem>
<parameter name="solution">Use `--pretty=tformat:'...'` instead of `--pretty=format:'...'`. The `t` prefix means "terminator" — git appends a newline after every commit, including the last. The `while read` loop then sees the newline and processes the final entry. Alternatively, use the canonical idiom `while read || [[ -n "$line" ]]; do ...; done` to consume the unterminated line, but `tformat:` is cleaner because it fixes the data, not the consumer. Generalizes: any `while read` loop fed by a tool with optional trailing newlines (printf, awk, custom scripts) needs either the `|| [[ -n "$var" ]]` guard or the producer to always terminate. When in doubt, write a test that asserts the FIRST and LAST records make it through.</solution>
<parameter name="tags">["bash", "git", "shell-pipelines", "off-by-one", "test-discovery"]

### [2026-05-15] MCP transport state and daemon state can disagree per-conversation

**Context:** Mid-session on a fresh Windows 11 laptop, after rejecting the very first `mcp__hive__session_briefing` call in the permission prompt. Every subsequent Hive tool call returned `MCP error -32000: Connection closed`, then `No such tool available`. Spent a few minutes assuming the Hive server had crashed.

**Problem:** `claude mcp list` in a separate terminal reported `hive: uvx hive-vault - ✓ Connected`. The daemon process was healthy; only the current conversation's handle to it was dead. There is no in-session command to re-attach. Filed as [mlorentedev/hive#75](https://github.com/mlorentedev/hive/issues/75); see also [`troubleshooting/hive-mcp-rejection-disconnect.md`](troubleshooting/hive-mcp-rejection-disconnect.md).

**Solution:** Two layers. Operationally: always accept the first MCP tool call in a fresh conversation; you can deny later ones safely. If the transport is already poisoned, restart the conversation — that is the only recovery. Diagnostically: when an MCP-backed tool stops working mid-session, the very first signal to capture is `claude mcp list` in a separate terminal. If the daemon is `✓ Connected` but the conversation cannot reach the tools, you have a session-state vs daemon-state divergence — not a server crash, not a config error, just a per-conversation transport handle that Claude Code does not recover on its own. The fallback is filesystem reads of the vault for the rest of the session.

**Rule:** Treat "MCP server appears dead" and "MCP server actually died" as two different failures and check them separately. The daemon-side check (`claude mcp list`) is free, takes seconds, and decides between "restart conversation" and "actually investigate the server". Doing this check first reframes the rest of the debugging — and prevents 20 minutes of poking at a healthy server.

**Tags:** `#mcp` `#claude-code` `#hive` `#transport-state` `#diagnostic-first-move`

### [2026-05-15] bash `IFS=$'\t' read` collapses consecutive tabs (whitespace IFS chars never preserve empty fields)

**Context:** Building `scripts/doctor.sh` to iterate `env-contract.json` entries. Used jq's `@tsv` to emit one TSV record per env-var, with `(.required_on // "")` for an optional column, then parsed with `while IFS=$'\t' read -r name required required_on default validation`. On entries where `required_on` was empty (most of them), every subsequent column shifted by one — `$default` got the validation value, `$validation` got an empty string, and the script ran silently wrong for ~10 minutes before a `bash -x` trace pinpointed it. Raw TSV had the right columns (`cat -A` confirmed); the read loop was eating them.

**Problem:** Bash's `read` treats *whitespace* IFS characters specially: even when you explicitly set `IFS=$'\t'`, consecutive tabs are collapsed into a single delimiter (the same rule applies to space and newline). So a TSV row like `name<TAB>true<TAB><EMPTY><TAB>$HOME/.dotfiles<TAB>path_exists` is read as if the empty third column did not exist, shifting every later assignment by one slot. POSIX behaviour, deeply confusing in practice because the documentation calls IFS "the field separator" and a quick reader assumes "tab means tab". The bug is silent: the script never errors, it just operates on wrong data.

**Solution:** Use a *non-whitespace* delimiter for TSV-like output whenever any column can be empty. Switched jq to `... | join("|")` and bash to `IFS='|' read -r ...`. Non-whitespace IFS chars do NOT collapse, so empty fields are preserved exactly. Pipe is safe for our values (paths, version strings, regex patterns) but for arbitrary content prefer the ASCII Unit Separator `$'\x1f'` — chosen by ASCII itself for this purpose, guaranteed absent from any sane string. Either way, **never use `IFS=$'\t'` (or `' '` or `$'\n'`) for `read` if an empty field is even possible**.

**Rule:** When a `read` loop must preserve empty fields, the IFS character must be non-whitespace (`|`, `;`, `:`, or `$'\x1f'`). For jq pipelines: replace `@tsv` with `[...] | join("|")`. For ad-hoc shell: never assume `IFS=$'\t' read` round-trips a TSV with empty columns — it doesn't, and the failure is invisible.

**Tags:** `#bash` `#ifs` `#read` `#tsv` `#silent-failure` `#jq` `#posix-gotcha` 

### [2026-05-16] PSScriptAnalyzer fails on non-ASCII in .ps1 without BOM
**Context:** Editing PowerShell scripts (`setup-windows.ps1`, `powershell/profile.ps1`, `scripts/*.ps1`) in this repo. CI runs PSScriptAnalyzer in the `lint-powershell` job with default rules.
**Problem:** Any non-ASCII character (em dash `—`, en dash `–`, arrows `→`, smart quotes `"" ''`, ellipsis `…`) in a `.ps1` file without a BOM trips the rule `PSUseBOMForUnicodeEncodedFile`, fails the lint-powershell CI job, and blocks the PR. Hit twice in two months: commit `464eecf` (Mar 2026, em dash in `setup-windows.ps1`) and commit `9d284b9` (May 2026 PR #36, em dash in `powershell/profile.ps1`). The bug surfaces only in CI — local edits and grep look fine, the byte sequence is just multi-byte UTF-8.</problem>
<parameter name="solution">Project policy: **ASCII-only in `.ps1` files; do not add a BOM**. Substitutions when writing/editing: em dash -> `--`, arrows -> `->`, smart quotes -> ASCII `'` `"`, ellipsis -> `...`. Pre-commit safety net: `grep -nP '[^\x00-\x7F]' powershell/*.ps1 scripts/*.ps1 setup-windows.ps1` must return zero hits. Comments in `.ps1` are as constrained as code — an em dash in a comment block is enough to fail CI. Note `.sh`, `.md`, and vault files are NOT subject to this constraint.</solution>
<parameter name="tags">["powershell", "ci", "lint", "psscriptanalyzer", "encoding"]
**Solution:** 

### [2026-05-18] Verify post-checks with hardcoded strings rot when the verified file is refactored
**Context:** setup-windows.ps1 had Select-String post-checks (lines 142 and 466) grepping for the literal "CORE PRINCIPLE" in deployed CLAUDE.md/GEMINI.md to confirm the AI-013 pointer-style refactor (2026-05-16) landed correctly. AI-013 actually replaced that content with pointers starting "First, read `AGENTS.md`" -- the string "CORE PRINCIPLE" no longer existed in any deployed file. Setup kept emitting two spurious [ERROR] lines on every run despite the actual Copy-Item succeeding. Discovered empirically on 2026-05-18 during WIN-003 validation re-run.</context>
<problem>Hardcoded verify strings create an invisible coupling between two files (the deploy source and the verifier). When the deploy source is refactored, the verifier silently lies: deploy succeeds (Copy-Item ran), post-check says "failed" (string not found), and the script appears partially broken. Same class as BUG-001 (copilot-instructions verify, fixed in PR #40 with the exact same pattern fix). Both bugs lived in main for weeks -- they were only caught by empirical re-run of setup on a clean machine where someone read the output carefully.</problem>
<solution>Use a durable marker tied to the *file format convention*, not arbitrary body content. For pointer-style files the convention is the first-line marker `'First, read \`AGENTS.md\`'`. Match it with `-SimpleMatch` (PowerShell Select-String) or `grep -F` (POSIX) to avoid regex interpretation of backticks. The marker survives content refactors because it IS the convention, not an arbitrary string in the content. setup-linux.sh already had this; setup-windows.ps1 lagged in 2 places (BUG-002, fixed PR #47). Lock the parity in tests/setup-windows.bats with cross-OS asserts so future drift fails CI not production.</solution>
<tags>["bash", "powershell", "verification", "setup-scripts", "refactor-drift", "parity", "silent-failure"]</tags>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-18] detect-and-act scripts go silently inert when upstream products change their surface
**Context:** dotfiles setup-windows.ps1 and setup-linux.sh detected GitHub Copilot CLI via `gh extension list | grep github/gh-copilot`. GitHub released a new standalone `copilot` CLI (winget GitHub.Copilot, agentic interface closer to Claude Code than to the legacy suggest/explain wrappers). On machines with the new v2 installed and operative, the script logged "extension not installed, skipping" and never deployed ~/.copilot/copilot-instructions.md. The AI-013 pointer-style refactor was functionally inert on those machines. Discovered 2026-05-18 only when the user typed `gh extension install github/gh-copilot` and got "matches built-in alias" error -- revealing both products existed.</context>
<problem>Detect-and-act scripts assume the upstream product's surface stays stable. When the vendor ships a new product with the same conceptual purpose but different binary/CLI/install path, the detection silently misses it. Two failure modes: (1) deploy never fires (config missing); (2) the script LOGS "success" because the detect branch was simply skipped (no fail, no warn -- log says "not installed, skipping"). The longer the gap between vendor change and detection update, the more machines drift silently.</problem>
<solution>Three layers. (1) Detect on the BINARY not on a package-manager extension list: `Get-Command copilot` / `command -v copilot`. Binaries are more stable as a surface than extension manager state. (2) Re-audit upstream products at least once per sprint -- vendor changes are frequent; quarterly audit is too slow. Add to sprint checklists: "any AI/dev-tool integration deployed in past 6 months -- verify upstream hasn't shipped a replacement product." (3) Annotate every detect-and-act block with an inline comment: `# Upstream: https://...  Last validated: YYYY-MM-DD`. Forces the re-audit habit when the comment goes stale.</solution>
**Tags:** `#setup-scripts` `#detect-and-act` `#upstream-drift` `#copilot` `#silent-failure`

### [2026-05-18] When an invariant changes, dead code emerges silently downstream

**Context:** SDD-001 (PR #49) added an unconditional `[sdd]` reminder to `$ContextLines` / `CONTEXT_LINES` at the start of both `claude-session-start.{ps1,sh}`. The new invariant: the context buffer is NEVER empty. Two pre-existing branches gated on the OLD invariant (`if (-not $VaultRoot -and -not $ContextLines) { exit 0 }` and the bash equivalent) became unreachable code -- they fired the `exit 0` branch only when both were empty, which can no longer happen. Also: the claude-mem heal block's `$ContextLines = "..."` overwrite (instead of append) would have wiped the new reminder when heal output existed.

**Problem:** Dead code born from an invariant change is silently broken in a way that compiles and runs but does the wrong thing. Three failure modes: (1) the dead branch is never executed (silent waste), (2) the dead branch IS executed and produces incorrect behavior because its precondition is now impossible (rare, but happens when state is computed by side-effects), (3) downstream blocks that share state with the upstream invariant get clobbered (the claude-mem overwrite case). The bug surfaces only on the next refactor or in production when a corner case finally triggers the dead path.

**Solution:** When changing an invariant (especially one as fundamental as "this buffer is always non-empty"), do a Pre-Flight Audit per the AGENTS.md Socratic Guardrail: grep the entire file for references to the OLD invariant's preconditions (in this case, `(-not $ContextLines)` and `[ -z "$CONTEXT_LINES" ]`) and decide for each: (a) is this block now unreachable? remove it explicitly; (b) does this block read the buffer state and act on it? verify the new invariant doesn't break it; (c) does this block WRITE to the buffer? change overwrite to defensive append. Document the invariant change in a comment at the original assignment site so future readers see "this was made invariant by SDD-001 -- downstream blocks rely on it being non-empty".

**Rule:** Invariant changes are interface changes in disguise. Audit every reader AND writer of the affected state. A grep for the OLD invariant's predicate is the cheap mechanical step that catches the dead-code class. Skipping the audit = future-you debugs a 5-line silent regression for 30 minutes.

**Tags:** `#refactor` `#invariants` `#dead-code` `#pre-flight-audit` `#shell-state`

### [2026-05-18] Bulk-copy operations collide silently with per-file deploy logic

**Context:** SDD-002 (PR #51) introduced a per-file deploy for `ai/claude/settings.json`: read template, substitute `__HOOK_COMMAND__` placeholder, merge with existing target using per-key policy. Both setup scripts also had a pre-existing bulk-copy `Copy-Item ai/claude/* ~/.claude/` (PowerShell) / `cp -rf ai/claude/* ~/.claude/` (bash) that would copy ALL files from the source dir, including the new `settings.json` template -- which contained the literal `__HOOK_COMMAND__` placeholder AND would have wiped the user's customizations.

**Problem:** The bulk-copy + per-file collision is invisible until the per-file logic introduces a placeholder or merge invariant that the bulk-copy can't honor. While both deploys produce "a file at the target", the bulk-copy version produces semantically wrong content (placeholder unsubstituted, user customizations lost). No error fires because both operations succeed at the filesystem level. The bug surfaces only when the user opens the deployed file and finds garbage, OR when downstream logic chokes on the placeholder.

**Solution:** When introducing per-file deploy semantics for a file previously covered by a bulk-copy, ALWAYS add an explicit exclusion to the bulk-copy at the same time. PowerShell: `Copy-Item ... -Exclude 'settings.json'`. POSIX bash: explicit loop with `[ "$(basename "$src")" = "settings.json" ] && continue`. Document the exclusion next to the bulk-copy with the reason ("handled by per-file logic in <function>"). A bats parity assert that grep-checks for the exclusion locks it in.

**Rule:** Per-file deploy + bulk-copy of the parent dir is a guaranteed collision. The first-PR-after-refactor often misses it because the symptom is "file exists at target, looks OK on glance". Pair every new per-file deploy with the exclusion edit in the same atomic PR. Generalizes beyond settings.json: skills, MCP configs, opencode commands, anywhere a curated subset of a directory needs per-file handling within an otherwise-bulk deploy.

**Tags:** `#setup-scripts` `#deploy` `#bulk-copy` `#collision` `#parity` `#shell` `#powershell`

### [2026-05-19] Defensive monitors are not fixes — trigger fix and monitor are siblings, not substitutes

**Context:** dotfiles#33 was the "original" fix for the upstream `anthropics/claude-code#59870` truncation bug — every `claude plugin install` call rewrites `~/.claude/.claude.json` and silently drops subscription metadata (organizationType, organizationRateLimitTier, projects map, onboarding flags), shrinking the file from ~75 KB to ~1.5 KB and forcing re-authentication. The "fix" was an idempotence guard: don't call install if the plugin already appears in `claude plugin list` output. Six months later SDD-021 (2026-05-18) added a session-start canary (`Test-ClaudeJsonSize` in `claude-session-start.{sh,ps1}`) that flags the symptom if it ever recurs, with a 10 KB threshold. Today (2026-05-19) I noticed the canary firing on every session: file at 3444 bytes, re-login prompt in every project.

**Problem:** The idempotence guard had a false negative for `claude-mem@thedotmack` — it never appears in `claude plugin list` output (different marketplace, `@thedotmack` vs `@claude-plugins-official`), so the literal-match check `grep -qF "claude-mem@thedotmack"` returns false on every setup run, triggering one real install call and one real truncation. The SDD-021 canary CORRECTLY surfaced this — the warning text was in the session-start additionalContext I was reading at boot — but the canary is a detector, not a preventer. I had been blaming the recurrence on something I had "just broken" instead of recognising the canary was doing its job and the trigger fix from dotfiles#33 was incomplete.

**Solution:** Add a snapshot/restore wrapper around the install call (BUG-004 / PR pending) as a defense-in-depth layer beneath the existing idempotence guard. The wrapper snapshots `.claude.json` to a tempfile before the install, restores from snapshot in `finally` iff the post-call size dropped >50% from a baseline ≥10 KB (same threshold as SDD-021's canary — single SSOT for "anomalously small"). Now there are THREE layers that have to fail for the user to see re-login: (1) idempotence guard catches the common case, (2) wrapper catches the false-negative case, (3) canary alarms at next session start if both fail.

**Rule:** When you ship a monitor for a bug you "fixed", the monitor firing is evidence the fix was incomplete, not evidence the fix was undone. Before assuming "I broke it again", check whether the monitor was designed for the exact failure mode now appearing. Three-layer thinking: prevention (the trigger fix), detection (the monitor), recovery (the auto-restore). Each layer guards against the others failing. The presence of a monitor does NOT discharge the obligation to find the residual trigger — it just gives you a finite time window before the bug bites the user.

**Tags:** `#defense-in-depth` `#monitoring` `#claude-cli` `#setup-scripts` `#three-layer-thinking` `#upstream-bug`

### [2026-05-19] Wide try/catch misclassifies the error and misleads the next reader

**Context:** SDD-002 (PR #51) wrapped `ConvertFrom-Json -AsHashtable` in `Merge-ClaudeSettings` with `try { ... } catch { Write-Warn "Claude settings template is not valid JSON after placeholder substitution: $_"; return }`. Under Windows PowerShell 5.1 (the default `PowerShell` interpreter on Windows), `-AsHashtable` does not exist — it was added in PowerShell 7.0. The actual exception thrown is `ParameterBindingException` ("A parameter cannot be found that matches parameter name 'AsHashtable'"), NOT a JSON parse error.

**Problem:** The catch was wide and the user-facing message anchored on "not valid JSON". For a debugging human reading the warning, the natural next step is to inspect the template file for JSON syntax errors — which are nonexistent. The actual root cause (PS version mismatch) is buried in the trailing `$_` interpolation that most observers skip. This was the entire surface area of BUG-005: not the missing parameter, but the misleading log line that prevented someone from finding the missing parameter for hours.

**Solution:** Two layers: (1) at the FIX site, replaced the catch by an explicit version check at script entry that re-execs under pwsh (BUG-005 / PR #58). (2) At the PATTERN site, catches should be either narrow (`catch [System.Management.Automation.ParameterBindingException]` for the specific case) OR the user-facing message should NOT assert a cause it cannot verify (write "Claude settings merge failed" + the raw `$_` exception type, not "is not valid JSON"). Better still: use `Test-Json` to check structure before parsing, so a JSON failure is a separate code path from any other parse-pipeline failure.

**Rule:** A wide `catch { ... }` is fine for "swallow and continue"; it is NOT fine when paired with a user-facing message that asserts a cause. If the message names a cause, the catch must be narrow enough that the cause is the only possibility. Otherwise the error message becomes a lie that costs more debugging time than no message at all. Auxiliary lesson: when a script must run under a newer runtime (PS 7+, Python 3.12+, Node 22+), the cheapest portability fix is to detect-and-reexec at the front door (single point of policy) rather than to backfill compat into every helper (N points, easy to miss one).

**Tags:** `#error-handling` `#powershell` `#portability` `#log-quality` `#wide-catch-trap` `#auto-reexec`


### [2026-05-19] Incident → guard pattern (red-team thyself)
**Context:** During a single 2026-05-19 session, Hive's vault_patch MCP wrote the literal 2-character sequence backslash-n into dotfiles/11-tasks.md four separate times instead of interpreting it as a newline. Each occurrence corrupted a markdown bullet list by merging two items into one physical line — invisible in rendered markdown but breaking init-spec.sh's vault-gate grep (which anchors on `^- [ ] **<id>**`) and any downstream line-based parser. The user surfaced the meta-issue: "la red de seguridad tiene que ir mejorándose a sí misma" — the safety net must keep improving itself.</context>
<parameter name="problem">Each corruption was fixed manually with the Edit tool. No guard was added until the 4th occurrence. By then, the bug class had already burned ~10 minutes of cumulative friction. The general failure: when a bug class hits, we tend to fix the immediate symptom and move on instead of adding a CI assertion / health check / parity test that prevents the next occurrence. Three sibling instances in the same session reinforce the pattern: (1) AI-019 missed `.github/copilot-instructions.md` Model Tier section — fixed in SDD-005 with `tests/docs-drift.bats`; (2) BUG-001 + BUG-002 verify-string drift between setup-linux.sh and setup-windows.ps1 — fixed earlier in PR #40 + #47 with bats parity asserts; (3) Hive vault_patch literal `[BS-n]` — fixed in SDD-006 with `scripts/check-md-escapes.sh` + bats. All three are the same meta-pattern: a class of failure recurs because each occurrence was patched without adding the structural guard.
**Problem:** 
**Solution:** **General rule (incident → guard, red-team thyself):** every bug class encountered MUST emit a CI assertion or health check in the SAME PR that fixes the symptom. Three signals that the rule is being violated: (a) you hit the same bug class twice in one session, (b) the fix is "I'll edit the file manually" with no test added, (c) the PR body says "I'll add the guard later". When you hit a bug class for the 2nd time, STOP. Don't fix the symptom — add the guard. The guard prevents the 3rd through Nth occurrence. **Concrete artefacts shipped under this rule in dotfiles:** `tests/docs-drift.bats` (mirror-file parity, SDD-005), `tests/check-md-escapes.bats` + `scripts/check-md-escapes.sh` (vault_patch corruption, SDD-006), `tests/setup-windows.bats` parity asserts (verify-string drift, BUG-002). **Pre-promotion to 00_meta/patterns/:** the pattern is dotfiles-specific for now; promote to a global pattern when a second project applies it.
**Tags:** `#safety-net` `#ci` `#testing` `#vault-patch` `#incident-driven` `#meta-pattern`

### [2026-05-19] Filename glob *.lock does NOT match package-lock.json (basename matters)
**Context:** Building scripts/check-spec-gate.sh for SDD-003: the LOC-exclusion list needed to skip lockfiles. Initial regex used a single glob pattern `*.lock` in a bash `case` statement. A bats test for `package-lock.json` failed because the file ends in `.json`, not `.lock` — the npm convention puts the lock-marker in the middle of the name, not the suffix.</context>
<problem>The naive pattern `*.lock` only matches files whose filename ends in literal `.lock`. It matches `Cargo.lock`, `poetry.lock`, `Pipfile.lock`, `yarn.lock`, `Gemfile.lock` — all the suffix-style conventions. It does NOT match npm's `package-lock.json`, pnpm's `pnpm-lock.yaml`, or Go's `go.sum`. Lockfile filtering with `*.lock` alone produces silent false negatives — exactly the kind of bug that ships green and breaks in production.</problem>
<parameter name="solution">Use basename-aware matching with explicit literals for non-suffix conventions. Pattern from scripts/check-spec-gate.sh:

```bash
_excluded() {
    local path="$1"
    local base="${path##*/}"
    case "$path" in
        tests/*|specs/archive/*) return 0 ;;
        *generated*) return 0 ;;
    esac
    case "$base" in
        *.lock|*.lockb) return 0 ;;
        package-lock.json|pnpm-lock.yaml|go.sum) return 0 ;;
        .gitignore|CHANGELOG.md) return 0 ;;
    esac
    return 1
}
```

Two `case` blocks: first checks path prefixes (tests/, specs/archive/) and substring matches (*generated*); second checks basename for filename conventions. ${path##*/} is bash parameter expansion for basename — no external `basename` call needed.

General rule: when filtering filenames by convention, list the literal exceptions BEFORE relying on a suffix glob. Suffix globs are a lower bound, not a complete set.</parameter>
<parameter name="tags">["bash", "globs", "lockfiles", "ci", "spec-gate"]</parameter>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-19] Numeric bats threshold drift is invisible — comment the bump inline
**Context:** AI-019 (model-tier policy) added a Model Tier subsection to ai/claude/CLAUDE.md, pushing the file from 70 to 78 lines. The existing bats assertion `wc -l < CLAUDE.md -le 70` started failing. Two options: compact existing content to fit under 70, or bump the threshold.</context>
<problem>If you silently bump a numeric threshold in a test (70 → 80 lines, 50 → 100 tests, etc.) without leaving a trace, the next contributor sees the new number and has no way to know whether (a) the threshold is calibrated to real constraints, or (b) it was raised to accommodate scope creep that should have been resisted. Threshold drift is invisible — every bump compounds; six months later, the assertion has become meaningless rubber. The classic "boiled frog" failure mode.</problem>
<parameter name="solution">When raising any numeric threshold in a bats test (or any CI assertion), add an inline comment in the SAME line/block stating: which spec/PR caused the bump, by how much, and the justification. Example from tests/opencode.bats line 148:

```bash
@test "ai/claude/CLAUDE.md is a pointer to AGENTS.md (≤ 80 lines)" {
    # Threshold bumped 70→80 in AI-019 (model-tier overlay added ~8 lines).
    # Future per-agent extensions should justify each bump in the spec.
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/claude/CLAUDE.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/claude/CLAUDE.md") -le 80 ]]
}
```

Now any future contributor reading the test sees: previous threshold, new threshold, what caused the change, and the implicit rule for next-bump justification. The comment also makes the audit trail visible to `git blame` so PR review can challenge unjustified bumps. Apply this to ALL thresholds — function-length linters, coverage minimums, perf budgets, file-size caps.</parameter>
<parameter name="tags">["testing", "bats", "thresholds", "code-review"]</parameter>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-19] "chore: close spec lifecycle" pattern — for features that shipped piecemeal before archive
**Context:** TERM-001-ghostty-bootstrap had its proposal scaffolded on 2026-05-17 but the implementation shipped piecemeal across PR #38 (tmux truecolor) + commit b00353e (full ghostty bootstrap) + commit 7424731 (config translation) before the spec lifecycle was formally closed. By 2026-05-19 the feature was 100% live on main with bats green, but the spec folder still sat in specs/ (not archive/) with tasks.md as a skeleton.</context>
<problem>SDD-001's archive criterion ("move folder to specs/archive/ on merge") is straightforward when a single PR ships the feature. It's awkward when implementation lands across multiple commits over multiple days — there's no single "merge" event to trigger archival, and tasks.md / verification.md don't get filled because the work was done. The risk: spec folders accumulate in active state indefinitely after the feature is shipped, polluting `ls specs/` and breaking the "active spec = WIP" invariant that the spec-gate CI relies on.</problem>
<parameter name="solution">Apply the "chore: close spec lifecycle" pattern: a small atomic PR that introduces ZERO production code changes and only does the archive housekeeping. Three artefacts in this PR: (1) fill tasks.md retroactively as a map from existing artefacts to the original TDD plan; (2) fill verification.md with the evidence map (commit hashes for each AC + bats output); (3) features.json with the harness-facing contract; (4) git mv specs/<id>/ → specs/archive/<id>/.

Example: PR #66 for TERM-001 — branched as `chore/TERM-001-close-spec-lifecycle`, single commit `chore(spec): close TERM-001-ghostty-bootstrap lifecycle`, 6 files changed (3 archive moves + 3 spec-folder additions), 160 insertions / 98 deletions, zero production code.

The pattern is NOT a workaround for SDD discipline. The proposal WAS filled before implementation (2026-05-17). The pattern is the final step that turns "shipped artefacts" into "archived spec + audit trail". The risk to guard against: scope creep — be strict about ZERO production code in close-out PRs.

The PR title prefix `chore(spec):` makes the intent obvious at review time.</parameter>
<parameter name="tags">["sdd", "spec-lifecycle", "housekeeping", "atomic-prs"]</parameter>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-19] JSONC native // comments beat _commentKey JSON convention for documentation
**Context:** AI-019 needed to document the model-tier mapping inside ai/opencode/opencode.jsonc. The proposal weighed two conventions: (a) a `_modelTierComment` JSON key with underscore prefix (convention says parsers ignore it); (b) native JSONC `//` line comments. OpenCode reads the file as JSONC (the file already used `//` for the schema URL + 6 other comment blocks).</context>
<problem>Documenting structured config inside JSON has no native syntax — comments aren't part of the JSON spec. Many projects invent the `_commentKey` convention (`"_comment": "..."`, `"_doc": "..."`). It works in practice because consumers ignore unknown keys, but it has two downsides: (1) it pollutes the parsed JSON namespace, so any tooling that enumerates keys sees noise; (2) underscore-key convention is unofficial — a future JSON schema validator might reject it. JSONC (JSON with Comments) is a different file format where `//` and `/* */` are first-class syntax.</problem>
<parameter name="solution">When the file is `.jsonc` (or any consumer accepts JSONC), prefer native `//` comments over `_commentKey` JSON keys. Rationale: (a) comments stay out of the parsed key namespace; (b) diff-friendly (line comments don't bracket-wrap content); (c) consistent with existing convention in that file. Example from ai/opencode/opencode.jsonc head:

```jsonc
// OpenCode configuration — managed by dotfiles.
// SSOT: ai/opencode/opencode.jsonc (this file).
// Schema: https://opencode.ai/config.json
//
// Model Tier (per AGENTS.md "Model Selection"):
//   Top: opencode-go/deepseek-v4-pro
//   Mid: opencode-go/qwen3.6-plus
//   Low: opencode-go/deepseek-v4-flash
{
  "$schema": "https://opencode.ai/config.json",
  ...
}
```

Risk mitigation: if the consumer ever rejects line comments (unlikely for any tool that accepts the .jsonc extension), fall back to a sibling Markdown file (e.g. ai/opencode/MODEL_TIERS.md) — explicit documentation file, no convention games inside the data.

Inverse: if the file is pure .json (no JSONC support), use a top-of-file `_comment: ` key as a least-bad option. The pollution is real but acceptable when the alternative is no documentation at all.</parameter>
<parameter name="tags">["json", "jsonc", "config", "documentation"]</parameter>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-20] Audit ALL call sites of a vulnerable upstream API when guarding one
**Context:** BUG-004 (PR #57, 2026-05-19) wrapped `claude plugin install` with snapshot/restore against upstream truncation bug `anthropics/claude-code#59870`. One week later (2026-05-20) the user reported `.claude.json` truncated AGAIN after a setup-windows.ps1 run. Diagnosis revealed BUG-004 missed ~9 sibling call sites that go through the same vulnerable deserialize-modify-serialize path: `claude mcp get`, `claude mcp add` (each ~9 times per setup), and `claude plugin list`. BUG-011 (PR #69) had to wrap all of them belatedly.</context>
<parameter name="problem">Fixing one symptom of a CLI-level vulnerability without enumerating siblings creates a guaranteed recurrence. The cost of audit-all-call-sites is small (~30 min grep + wrap); the cost of skipping it is a recurring incident class one week later.
**Problem:** 
**Solution:** When patching ANY guard around a vulnerable upstream API call, MUST in the same PR: (1) grep both setup scripts (sh + ps1) for every invocation of the vulnerable binary/subcommand family; (2) wrap each invocation with the same guard; (3) add bats parity asserts that fail CI if a future call site lands without the guard. The audit step is non-negotiable; "wrap one and move on" is a smell.
**Tags:** `#incident` `#guard-pattern` `#audit-discipline` `#claude-cli` `#cross-os-parity`

### [2026-05-20] Claude Code marketplace dir naming follows GitHub repo, NOT declared name field
**Context:** BUG-012 (PR #70, 2026-05-20). Diagnosing `UserPromptSubmit operation blocked by hook` on Windows. Hook command was claude-mem plugin's `bun-runner.js` discovery script. It searched at `~/.claude/plugins/marketplaces/thedotmack/plugin/scripts/` (the marketplace's declared `name` from `marketplace.json`), but Claude Code had cloned the marketplace under `~/.claude/plugins/marketplaces/thedotmack-claude-mem/` (GitHub repo name `thedotmack/claude-mem` flattened with `-`). When `CLAUDE_PLUGIN_ROOT` is unset/stale (Windows: cache `plugins/cache/thedotmack/claude-mem/` stays empty post-install), the hook falls through to the broken fallback → `exit 1` → blocked.</context>
<parameter name="problem">A plugin can declare `name: "thedotmack"` in `marketplace.json` but Claude Code names the install dir after the GitHub repo (`<owner>-<repo>` in some cases, just `<repo>` in others — naming logic not fully documented). Plugins that hardcode `marketplaces/<declared-name>/plugin/scripts/...` in fallback paths break silently on machines where the env-var path fails and the fallback is consulted. Symptom on Windows is `printf: write error: Permission denied` (Git Bash + Claude Code hook subprocess sandbox quirk); on Linux it's a cleaner `claude-mem: plugin scripts not found`. The cross-OS root cause is identical.
**Problem:** 
**Solution:** Defense in depth: create a junction (Windows, no admin) / symbolic link (Linux) from the declared name to the actual install dir during heal. Guard creation on `source exists AND target absent` for idempotence. Once the link exists, ALL code that hardcodes the declared name (plugin's bundled hooks.json, .mcp.json, etc.) resolves correctly without modifying upstream files. The link survives `/plugin update` operations. Pattern lives in `scripts/claude-mem-heal.{sh,ps1}` as `ensure_marketplace_compat_symlink` / `Repair-MarketplaceCompatJunction`.
**Tags:** `#claude-code` `#plugin-discovery` `#claude-mem` `#cross-os-parity` `#junction-pattern`

### [2026-05-20] vault_patch timeout != patch not applied
**Context:** During WIN-001 (PR #71) and the post-merge vault tick, two `mcp__hive__vault_patch` calls to `10_projects/dotfiles/11-tasks.md` returned `vault_patch timed out after 60s. Server may be under load or a lock is contended; retry shortly.` In BOTH cases the patch had actually committed; the file was already in its new state when the retry ran, and the retry then failed with `patch N: find text not found.` (because the original anchor text no longer existed). Pattern observed twice in the same session.</context>
<parameter name="problem">The naïve retry path on `vault_patch` timeout (call the same patch again with the same find/replace) produces a false `find text not found` failure that masks the fact that the first call succeeded. Wasted ~30s per occurrence verifying after the fact. If the file was not idempotent under the new state, a retry could also corrupt content.
**Problem:** 
**Solution:** On any `vault_patch` timeout, do NOT immediately retry. Instead: (a) `mcp__hive__vault_query` the file to inspect current state, (b) determine if the patch text is already applied (by checking for the post-state content), (c) only retry if the original anchor still exists. This is the same defensive pattern as setup-script idempotence: read-then-write, never blind-write-twice. Treat the Hive timeout response as "outcome unknown" not "outcome failed".
**Tags:** `#hive` `#vault` `#mcp` `#idempotence`

### [2026-05-21] Fix ALL surfaces in the same PR when a bug class spans multiple call sites
**Context:** Today's claude-mem upstream bug cascade. BUG-016 fixed `.mcp.json` cascade-pipe EPIPE race but deferred `hooks.json` as "future BUG-017". Minutes after BUG-016 merged, user hit hooks.json fail → BUG-017 had to ship. Then BUG-017 fixed the pipe but UserPromptSubmit still blocked because hook command lacked `{"continue":true}` directive → BUG-018 narrow (only session-init). Minutes later, Stop hook failed 9x in a row → BUG-018 extended via regex to all 5 hooks. Three deferrals, ~30 min user pain each.
**Problem:** When patching a bug class affecting multiple call sites of the same upstream system, "fix one surface and ship" creates a guaranteed cascade. Each subsequent surface fail is a separate ticket, separate context-switch, separate ~30min of user-visible breakage. The cost of audit-all-surfaces-once is low (10-20 min grep + apply); the cost of skipping is unbounded.
**Solution:** When discovering a bug pattern (e.g. `break; }; done` cascade race; missing `{"continue":true}` directive), the SAME PR that fixes the first surface MUST: (a) `grep -E '<broken-pattern>' <relevant-files>` to enumerate every callsite; (b) apply the same fix or regex-substitute across all of them; (c) bats parity asserts that lock the substitution count. Companion to BUG-011's "audit all call sites of vulnerable upstream API" lesson — applies the same principle one layer down (pattern instead of API).
**Tags:** `#bug-class` `#audit-discipline` `#claude-mem` `#regex-substitution` `#cascade-cost`

### [2026-05-21] The detection probe MUST use the race-free pattern, not the upstream broken one
**Context:** BUG-015 (PR #81, today) added a healthcheck probe to detect when claude-mem's path-resolution cascade fails. The probe used the EXACT SAME `{ printf; ls; printf; } | while ... break` pattern as the upstream hooks it was checking. After BUG-017 patched the upstream hooks with `head -n1`, the probe itself STILL raced — reporting false-positive FAIL because the probe-internal EPIPE fires before resolution completes. BUG-022 had to ship a separate fix to apply the same `head -n1` to the probe.
**Problem:** When writing a detection/observability layer for a known upstream bug pattern, copying the broken pattern verbatim "to faithfully reproduce what the hook does" preserves the same failure mode in the detection layer. The probe inherits the bug it was designed to detect → false-positive FAILs masking real state.
**Solution:** Probes MUST use the canonical race-free version of the pattern being detected. The probe is not the same as the upstream — its job is to ACCURATELY REPORT state, not faithfully reproduce broken behaviour. If the bug-class signature still appears in source (e.g. `break; }; done` in hooks.json), the probe should grep for the broken signature in the FILE (cheap, deterministic) rather than re-execute the broken logic. Cross-OS parity matters: even when the race is silent on Linux (SIGPIPE clean), use the race-free form for consistency.
**Tags:** `#observability` `#detection-layer` `#race-condition` `#healthcheck` `#cross-os-parity`

### [2026-05-21] Healthcheck must validate end-state, not proxy artifacts
**Context:** BUG-014 (PR #75, today). The pre-existing healthcheck assertion for claude-mem (BUG-012 era) checked whether the filesystem JUNCTION existed at `~/.claude/plugins/marketplaces/thedotmack/`. The junction exists if BUG-012's heal ran. But the heal only runs if the marketplace dir is present, and the marketplace dir presence does NOT imply the plugin is installed in `installed_plugins.json` (Claude Code's canonical install record). Result: healthcheck reported `PASS: claude-mem marketplace legacy junction present` while `/mem-search` was unavailable, session-start hook never fired, and `installed_plugins.json` had zero `@thedotmack` entries. False positive that hid the real bug for days.
**Problem:** Asserting a proxy artifact (filesystem junction = consequence of heal) instead of canonical state (installed_plugins.json = source of truth for "is the plugin actually installed") makes the healthcheck unable to detect a whole class of failures. The asymmetry is dangerous: proxy artifacts can exist WITHOUT the canonical state being correct.
**Solution:** Every healthcheck assertion should validate the END-STATE that the user cares about, not a proxy. For "is plugin X installed?" → grep `installed_plugins.json`. For "is service Y running?" → query the service status, not "config file exists". For "is alias Z configured?" → invoke the alias and check exit, not "alias line present in profile". When a proxy is the only available signal, the assertion message should explicitly say "proxy" — and a PRIMARY canonical assertion should come first.
**Tags:** `#healthcheck` `#observability` `#false-positive` `#end-state-vs-proxy`

### [2026-05-21] PowerShell single-quoted strings + grep BRE: backslash counting
**Context:** BUG-017 PR #84 first CI run failed on bats test 471. The grep pattern was `'hooks\\\\hooks\.json'` in single-quoted bash to match a PowerShell single-quoted path `'hooks\hooks.json'`. With 4 backslashes the regex matched `hooks\\hooks.json` (two literal backslashes) — wrong. Reducing to 2 backslashes made it match `hooks\hooks.json` (one literal backslash) — correct.
**Problem:** The "double the backslashes" rule of thumb (count literal `\` then × 2 for regex escape) over-applies when the source string is ALREADY single-quoted bash. In bash single-quotes, every char is literal — no shell-level escape happens. So `'\\'` is exactly 2 chars sent to grep. Grep BRE treats `\\` as one literal `\`. Hence 2 backslashes in bash single-quote == 1 literal backslash matched. Adding 4 backslashes gives 2 literal matches, which over-shoots when the source has only one.
**Solution:** Counting rule for grep BRE inside bash single-quote: target literal backslashes × 2 (NOT × 4). Examples: `'\\'` → matches `\`; `'\\\\'` → matches `\\`; `'\\\\\\\\'` → matches `\\\\`. Tip: extract the pattern into a `pat=$'...'` ANSI-C variable for double-quote interpolation rules instead, OR use `grep -F` (fixed string, no regex) when you don't need wildcards — sidesteps the entire escape rabbit hole.
**Tags:** `#bash` `#powershell` `#grep` `#regex-escape` `#bats`

### [2026-05-21] Heal scripts versioned against the upstream bug class they paper over
**Context:** BUG-016 (PR #83, today). `claude-mem-heal.{sh,ps1}::Repair-McpJson` was authored against v12.7.4's broken `${_R%/}` literal in `.mcp.json` (PR #57, 2026-05-19). v13.0.0+ shipped a different broken pattern: cascading-printf via `sh -c` triggering the EPIPE race. The heal silently no-oped against v13.3.0 installs because the v12.7.4 signature was absent. User hit the upstream MCP failure repeatedly while the heal kept exiting clean — `[claude-mem-heal] .mcp.json already healthy: ...` was a false claim.
**Problem:** Heal scripts are bug-class-specific by design (they paper over a SPECIFIC broken upstream pattern). When the upstream changes its bug pattern (intentionally or accidentally — v12 → v13 shipped different brokenness), the heal's detection regex no longer matches, and the heal becomes a no-op. The script reports "healthy" while the install is broken. Worst: silent failure.
**Solution:** When the upstream version changes AND a bug class is still being papered over, the heal's detection MUST be refreshed in the same investigation that discovers the new pattern. (a) Detect each known broken signature with an OR (don't replace the v12 detection with v13 — keep both, since rollbacks happen). (b) Log which signature was patched so future audits can map heal output → upstream version. (c) Bats parity asserts must lock BOTH detection signatures (each on its own assert) so regressions surface in CI. (d) Add a stronger assertion in healthcheck that validates the canonical fix actually landed (the `head -n1` substring in the patched file), not just that the heal "ran". Companion to BUG-014's end-state-not-proxy lesson.
**Tags:** `#heal-pattern` `#upstream-versioning` `#defensive-scripting` `#claude-mem`

### [2026-05-21] Safety-net fixes must be audited against the same bug-class they paper over
**Context:** BUG-022 (PR #87, 2026-05-21) was a fix for BUG-015's hook-resolution probe. The probe originally re-executed the same `break; }; done` EPIPE-race pattern as the upstream claude-mem hooks. BUG-022 appended `head -n1` after the while-loop to make the pipeline race-free. The same day, hours later, BUG-023 surfaced: the BUG-022 fix STILL raced under `set -euo pipefail` when 2+ candidates matched in cache (this user had 6 versions: 12.7.4 -> 13.3.0). `head -n1` closed the consumer; leftover printfs in the while loop got EPIPE; pipefail propagated 141; `set -e` killed healthcheck.sh mid-section 4/12. setup-linux.sh logged a false-positive WARNING. The fix was a half-fix for the same bug-class.
**Problem:** When a safety-net fix patches a bug-class (race condition, escape bug, error-suppression pattern), it's tempting to believe the patch fully closes the class. But a partial patch can leave a structurally identical sub-case open -- exactly the same bug-class, just in a different scenario count (0-1 matches vs 2+ matches; single producer vs multiple producers; single-arg vs varargs). The next time someone reads the code, the fix looks defensive and complete, masking the residual vulnerability. Cascade-cost lesson applies recursively: every safety-net iteration is itself a candidate for the same audit.
**Solution:** When shipping a fix for a bug-class, before merging, ask: 'could the same bug-class symptom still fire in a different cardinality, a different producer/consumer count, or a different shell mode (pipefail on/off, set -e on/off, strict-mode on/off)?' If the answer is maybe, explicitly enumerate the cases and add a bats parity assert with BOTH positive (new pattern present) AND negative (old broken pattern absent) for each cardinality. Concrete pattern for pipe-with-early-close races: materialize candidates into a variable first, then iterate in pure bash with `break` -- no pipe at all means no consumer-close, no EPIPE, no pipefail propagation. Avoid `done | head -n1` form entirely; it is a half-fix dressed up as a full one. Companion to the 'detection probe must use the race-free pattern' lesson -- applies it recursively to the fix itself.
**Tags:** `#safety-net` `#audit-discipline` `#race-condition` `#bash` `#pipefail` `#cascade-cost` `#claude-mem`

### [2026-05-21] Classify "extract boilerplate" audit findings as bootstrap (chicken-and-egg) vs logic before estimating LOC savings
**Context:** AUDIT-005 (REFACTOR-001 scripts/ audit, 2026-05-21) proposed POLISH-001: extract get_script_dir + utils.sh source-fallback boilerplate to utils.sh, estimating -75-85 LOC reduction. During closeout investigation, the actual numbers came out very differently: the boilerplate is bootstrap code that runs BEFORE utils.sh is sourced. A helper inside utils.sh cannot replace bootstrap that's needed to find utils.sh in the first place — chicken-and-egg. The only extraction path is a new _bootstrap.sh file each script sources via a 1-liner; net ~-10 LOC after counting the bootstrap file content, with +1 file overhead and non-trivial script-loading risk. Decision: WONTFIX.
**Problem:** Audit-style analysis tends to flag "repeated code across N files" without distinguishing whether the repetition is structural (must happen before any extraction target exists) or logical (after extraction target is available). The two have very different extractability and ROI characteristics. Conflating them produces misleadingly high LOC-saving estimates that look like quick wins but require either a new file (with its own ongoing maintenance cost) or sophisticated meta-tricks like self-locating libraries. The audit-005 agent treated the 11-script SCRIPT_DIR pattern + 10-script utils.sh source pattern as if they were ordinary code duplication, but both are bootstrap.
**Solution:** When an audit (your own or an agent's) flags "extract boilerplate" or "shared-pattern extraction", classify EACH proposed extraction before estimating value: (1) Is the repeated code BOOTSTRAP — runs before the would-be helper file is loaded (e.g., resolving where to find the helper)? Inextricable without adding new files or shell meta-tricks. The honest LOC saving is near-zero. (2) Is the repeated code LOGIC — runs after the helper is loaded? Extractable with a normal function. The LOC saving is real. For bootstrap-class patterns, the cost of extraction (new file + ongoing maintenance + loading-order risk) usually exceeds the LOC saving. Document the classification in the audit so future readers don't re-litigate. Concrete test: can the audit explain how the helper itself would be loaded inside the same boilerplate it's trying to replace? If not, it's bootstrap.
**Tags:** `#audit-discipline` `#refactoring` `#bash` `#boilerplate` `#bootstrap` `#chicken-and-egg` `#loc-estimation`

### [2026-05-21] Setup-time mutations to repo-symlinked files create permanent drift false-positives
**Context:** BUG-024 (PR #93). After REFACTOR-001 audit chain closed earlier today, the next ./setup-linux.sh fresh run reported `[12/12] Repo ↔ Deploy-Dir Drift FAIL`. diff-check.sh (PR #10) showed 3 lines drifting in both .bashrc and .zshrc: opencode PATH, project-init alias, and ~/.dotfiles/scripts on PATH.</context>
<parameter name="problem">setup-linux.sh appended those 3 lines to ~/.bashrc and ~/.zshrc via `ensure_line_in_file` *after* symlinking them to the repo (L431-433 opencode PATH, L903-905 project-init alias, L922-924 scripts PATH). Because the rc files are symlinks into the deploy-dir, the appends always wrote through to the deploy-dir copies — making those copies diverge from the (clean) repo source on every fresh setup. The drift detector then flagged it on the very next CI run. The repo source rc files also lacked a trailing newline, which caused the first appended line to concatenate onto `fi`.</problem>
<parameter name="solution">Make the repo the SINGLE writer for any file the drift detector watches. Bake the 3 lines directly into the repo source .bashrc/.zshrc (with trailing newline), delete the 3 `ensure_line_in_file` blocks from setup-linux.sh. `ensure_line_in_file` remains a valid pattern for *external* rc files (files the dotfiles repo does NOT own), but NEVER for files the setup has already symlinked from the repo. Test invariant rewritten in tests/opencode.bats #5 to assert repo-as-SSOT and forbid the old `ensure_line_in_file` pattern for these specific lines. Sibling of `lesson_dotfiles_two_tier_deploy.md` — same family of two-tier-deploy bugs where a writer bypasses the SSOT.</solution>
<parameter name="tags">["drift-detection", "setup-idempotence", "two-tier-deploy", "BUG-024"]
**Problem:** 
**Solution:** 

### [2026-05-21] Byte-equivalence assertions require SCRIPT_DIR control, not just literal diff
**Context:** SDD-004 (PR #97). Claimed acceptance criterion: refactor preserves byte-identical output. First attempt at the assertion ran `git show main:scripts/claude-session-start.sh > $(mktemp /tmp/...sh)` then diffed PRE vs POST. False-positive diff appeared (claude-mem heal block missing in PRE, vault-health line different in PRE).</context>
<parameter name="problem">The `mktemp /tmp/...sh` for PRE put the script in /tmp, which changed its SCRIPT_DIR resolution. The pre-refactor script looked for sibling helpers (`claude-mem-heal.sh`, `vault-health.sh`, `doctor.sh`) in /tmp/, didn't find them, so silently skipped those injectors. POST ran from real `scripts/` directory and found them. The diff was 100% methodology artifact — both versions of the script behaved IDENTICALLY when given identical sibling-script paths. Worse failure mode masked by this: a real refactor regression could be hidden under the same diff noise.</parameter>
<parameter name="solution">For any refactor that asserts byte-identical output via PRE-vs-POST diff: the PRE script copy MUST live in the same directory as POST so `SCRIPT_DIR`-relative lookups (sibling scripts, configs, fixtures) resolve identically. Pattern: write PRE to `<script-dir>/<name>.sh.pre-refactor`, run, diff, delete. Captured in tests/session-start-config.bats #14 (byte-equivalence test) + verification.md row 2. Sibling caveat: if the script's output ALSO includes live state queries (vault-health unresolved-links count changes per minute), pure twice-run-deterministic isn't guaranteed — but PRE-vs-POST at the SAME MOMENT cancels that drift, so the SCRIPT_DIR fix is sufficient. Reframe of R1 from "literal byte-equivalence" to "code-controlled byte-equivalence at fixed SCRIPT_DIR".</parameter>
<parameter name="tags">["byte-equivalence", "testing-methodology", "SDD-004", "verify-before-act", "refactor-safety-net"]
**Problem:** 
**Solution:** 

### [2026-05-25] Batch-scaffold N specs in one PR from a research worktree, defer implementation

**Context:** Did a research worktree comparing 3 reference dotfiles repos (fmontes / holman / mathiasbynens) against this repo. Research surfaced 6 actionable ideas with clear ROI tiering. Two paths forward: (a) open 6 separate PRs, one per spec, paid out over weeks; (b) batch-scaffold all 6 in one PR, defer implementation to per-spec branches.

**Problem:** If each spec gets its own scaffolding PR, the cross-spec dependencies discovered during research (IDEAS-003 enables clean IDEAS-001/002 integration; IDEAS-005 + IDEAS-004 interlock on fresh-machine setup; IDEAS-006 has an abandon gate) get rediscovered or lost across review cycles. Also: 6 small scaffolding PRs each pay the spec-gate / CI / review overhead, but produce zero shippable code. The research doc itself (`research/dotfiles-survey.md`) is the durable index — splitting it across 6 PRs scatters it.

**Solution:** Bundle the research doc + all N spec folders into ONE PR (PR #101, 1,629 LOC, 19 files, 0 code changes). Outcomes: (1) spec-gate workflow PASSED with NO `skip-sdd` label because the new `specs/<id>/` folders ARE the active specs satisfying the gate — confirmed empirically on PR #101; (2) cross-spec dependencies + ordering recommendations live in the PR body as the canonical handoff; (3) implementation work is fully isolated per-spec on `feat/IDEAS-NNN-*` branches with full reviewer attention; (4) the research worktree is disposable after merge. Each spec includes BLOCKER-classified risks, testable AC, features.json skeleton — same shape as SDD-004 so reviewers parse them identically.

**Why:** research outputs go stale fast; batch-capturing them as specs locks in the analytical work before the context window or session memory loses it. **How to apply:** any future research session (compare-X-against-Y, audit-N-instances-of-Z, survey-the-field) → bundle the research doc + every surfaced spec into one scaffold PR before opening implementation branches. Use Tier-1/2/3 + LOC estimate + cross-spec dependencies in the PR body to drive the implementation order.

**Tags:** `#sdd` `#specs` `#research-pattern` `#pr-strategy` `#dotfiles-survey`

### [2026-05-26] Incomplete migration: file rename leaves callers stale
**Context:** SDD-007 renamed GEMINI.md → AGY.md across the dotfiles project. Three separate cleanup PRs (#105, #108, #109) all post-merge found different surfaces still referencing the old name or stale assumptions.
**Problem:** When migrating an identity file (X.md → Y.md), it's tempting to think the work is "rename the file in repo + deploy from new path". But the FILE rename touches 3 distinct surfaces that all need updating in lockstep:

1. **The filesystem**: pre-migration installs accumulate the orphan file. Setup must `rm -f old_name` BEFORE copying new_name, idempotently.
2. **Tooling that READS by name**: healthcheck assertions, init-project copies, bats tests — every callsite that grep'd for "X.md" or did `Test-Path X.md` is now broken or asserting stale state.
3. **The CONTENT inside the new file**: the H1, body text, and embedded references may still describe the old tool. The rename ships a hollow shell unless the migration also refreshes prose.

In SDD-007 specifically: PR #102 did the file rename. PR #105 (orphan + healthcheck refs + init-project refs), PR #108 (stale content inside AGY.md), and PR #109 (hc path mismatch) each unwound one of the three surfaces — three separate post-merge fix PRs for what should have been one cohesive migration.</problem>
<parameter name="solution">For any rename/migration PR going forward, the checklist BEFORE merge is:

```bash
# 1. Find every caller of the old name (every surface)
grep -rIn "OLD_NAME" --exclude-dir={node_modules,.git,specs/archive} .

# 2. Confirm every hit is either:
#    - Updated to NEW_NAME, OR
#    - An intentional historical reference (archive, vault lesson), OR
#    - A cleanup statement (rm -f OLD_NAME with explanatory comment)

# 3. Check the NEW file's content, not just its filename
head -20 path/to/NEW_NAME  # Does the H1 / body still describe OLD?

# 4. For each callsite, ask: "what happens on the next setup run for a
#    user whose machine has the pre-migration state?" — if the answer
#    isn't "the orphan gets removed and the new state is reconciled",
#    the migration is incomplete.
```

Better still: add a `tests/` regression guard that bans the old name in the canonical SSOT (e.g. `! grep -qF '# GEMINI.md' "$DOTFILES_DIR/ai/agy/AGY.md"`). That makes "completion" mechanical rather than reviewer-vigilance.

The pattern is "incomplete migration" — same root cause class as "set -u: requires ${1:-} for optional positional parameters" (one-place fix masking missed-callsite bugs). When you change something, all its callers need to come with you.
**Solution:** 
**Tags:** `#migration` `#refactoring` `#pre-merge-discipline` `#regression-guards` `#dotfiles`

### [2026-05-26] Stop fighting agent filesystem expectations

**Context:** SDD-007 was triggered by BUG-100 — `agy` (Antigravity CLI) v1.0.2 collided with our deploy strategy: agy writes in-place to `~/.gemini/...` paths, but our `setup-linux.sh` had placed symlinks at those paths pointing back to the repo. Symptoms: EEXIST errors (forum #145851), circular link traversal (gemini-cli issue #10960), silent state corruption. The reflex fix would have been "patch each collision case as it surfaces". The root-cause fix was different.

**Problem:** Our deploy strategy was *fighting the agent's expected filesystem layout*, not the agent being buggy. Every new agent we adopt (today: agy, opencode, claude-mem; tomorrow: Cursor, Codex, Devin, whatever) has its own conventions for what it writes-in-place, what it expects to be a regular file, what it tolerates as a symlink. If our deploy strategy hardcodes a single mechanism (symlinks pointing back to the repo), we will hit BUG-100-class incidents N times — once per agent that disagrees with that mechanism.

**Solution:** Default deploy mechanism = **copy** (atomic via `deploy_file` helper, idempotent via `cmp -s`). Symlinks become the *exception*, reserved for: (a) intentional vault↔home bindings (Obsidian vault content → memory paths), (b) secret files needing canonical absolute paths (`~/.ssh/id_ed25519`, `~/.config/age/key.txt`). All other config paths get copies. The "edit-in-`~`-silently-loses-change" failure mode that copies introduce gets neutralized by a `check_deployed` drift assertion in `scripts/healthcheck.sh` — drift surfaces loud red. ADR-012 captures the decision.

**Why:** Symlinks express "this path IS the repo file" — that's a strong claim about filesystem identity that not every consumer of the path respects. Copies express "this path is *a copy of* the repo file at deploy time" — a weaker, more universal claim. The weaker claim is portable across N agents; the stronger one only works as long as all consumers cooperate. SDD-008 will extend this discipline to skills (vault `00_meta/skills/` → render via copy to each agent's native path, no symlinks).

**How to apply:** When adding a new deploy target (any path under `~/.*/` that the repo owns), default to `deploy_file` (copy). Only use symlinks when there's a SPECIFIC, documented reason (cross-system binding that must stay live; secret with hard-coded absolute path requirement). Update the `DEPLOYED_FILES` registry so the drift assertion covers the new path. If the target is consumed by a CLI tool that writes-in-place to its own configuration directory, *always* use copy — don't even debate it.

**Tags:** `#deploy` `#cross-agent` `#filesystem` `#sdd-007` `#bug-100` `#adr-012` `#dotfiles`

### [2026-05-26] pkill -f self-kill: pattern matches the pkill command line itself

**Context:** During a cleanup session uninstalling several AI CLIs (opencode, agy, gemini-cli), I ran `pkill -TERM -f 'opencode|antigravity|/agy\b|gemini-cli'` to terminate any live processes. The shell exited with code 144 (128 + SIGTERM 16) — pkill killed the shell that invoked it.

**Problem:** `pkill -f PATTERN` matches against each process's **full command line** (argv as a single string). The shell running my pkill command had a command line containing the literal string `pkill -TERM -f 'opencode|antigravity|/agy\b|gemini-cli'` — and `opencode` substring of that command line matches the pattern `opencode|...`. So pkill cheerfully matched ITS OWN INVOKING SHELL and signaled it. The shell got SIGTERM mid-command and died. This is silent and confusing: the cleanup appears to have failed (exit 144) but actually nothing was killed except the wrapper itself.

This is a recurring class — anyone writing a script that uses `pkill -f` with a pattern broad enough to match human-readable words will hit this on certain shell invocations (especially when the pattern is constructed dynamically from user input or config). The bash-tool-in-an-AI-loop case is particularly prone because each command is launched in its own subshell whose argv contains the full command string.

**Solution:** Three safer alternatives in order of robustness:

```bash
# 1. Match the basename only (no -f flag) — kills by program name not command line.
#    Best when you know the exact binary name.
pkill -x opencode 2>/dev/null
pkill -x agy 2>/dev/null

# 2. With -f, explicitly exclude $$ (current shell PID).
MYPID=$$
pgrep -f 'opencode|/agy|gemini-cli' | grep -v "^$MYPID$" | xargs -r kill -TERM

# 3. Use a sentinel that the pattern won't match in your own command.
#    Wrap the pattern in [oo]pencode trick used by ps/grep — the brackets aren't
#    literal in regex but the LITERAL string '[o]pencode' doesn't match 'opencode'.
pkill -f '[o]pencode|[a]ntigravity'   # matches OUR processes only
```

**Why:** `pkill -f` is one of those tools where the obvious mental model ("match programs whose name contains X") is wrong — it actually matches "processes whose entire command line contains X", and your own pkill invocation IS such a process. The semantics are documented but counter-intuitive, and the failure mode (exit 144, no error message, your script terminated) gives no clue what happened.

**How to apply:** Default to `pkill -x BASENAME` when you can. Only use `pkill -f PATTERN` when you genuinely need command-line matching (e.g., killing one specific instance of a daemon with distinguishing args), AND in that case either exclude `$$` explicitly OR use the `[X]regex` self-exclusion trick. In any AI-agent loop or wrapper script context, never trust `pkill -f` with broad patterns without one of these guards — the wrapper's own argv WILL match common English words.

**Tags:** `#shell` `#process-management` `#pkill` `#self-kill` `#exit-codes` `#dotfiles` `#ai-loop-gotcha`

### [2026-05-27] MEMORY.md files break YAML parsing with `# currentDate` and `---` separators

**Context**: 23 `MEMORY.md` files across all projects had invalid frontmatter. `vault_health --checks frontmatter` reported errors on every one.

**Problem**: Three distinct YAML-breaking patterns:
1. **`# currentDate` after frontmatter** — YAML sees `# currentDate` as a second document (comments after `---` close the first doc).
2. **`> Updated:` block scalar** — YAML interprets `>` as a folded block scalar, then chokes on the next line (e.g., `**Last task:**` doesn't start with a comment or line break).
3. **`---` in body** — Session handoff blocks use `---` as dividers, which YAML interprets as a new document separator.

**Solution**: Wrap the entire body in a single YAML `content: |` literal block scalar under the frontmatter. **Do NOT use a trailing `---`** — that starts a second document. The file is still valid Markdown (Obsidian renders it correctly). Pattern:
```yaml
---
id: "<project>-memory"
type: memory
status: active
tags: [memory, <project>, claude-code]
content: |
  # Title
  
  > block scalar text
  > more text
  ---
  another section
```
Note: the body's `---` is safe because it's inside the `|` scalar.

**Rule**: When a Markdown file needs both valid YAML frontmatter AND body content that may contain YAML-reserved characters (`>`, `---`, `#` at start of line), wrap the body in a `content: |` literal block scalar. Never use a trailing `---` after frontmatter — it starts a second YAML document. For Obsidian compatibility: the `content` key is ignored by Obsidian's markdown renderer (it only reads frontmatter + body), so the file renders identically.

**Tags:** `#yaml` `#frontmatter` `#obsidian` `#memory` `#validation` `#structural-debt`

### [2026-05-26] PowerShell -replace with [\s\S]*? expands large strings instead of replacing
**Context:** setup-windows.ps1 profile-section block used -replace regex to update dotfiles section in PowerShell profile
**Problem:** PowerShell's -replace operator with [\s\S]*? regex pattern EXPANDS large strings (>10KB) instead of replacing. A profile with 1 marker and 0 errors became 4 markers and 5 errors after a single -replace run, then 30+ markers on subsequent runs. The -replace and [regex]::Replace both failed — same behavior. Root cause: PowerShell regex engine backtracking on large strings with non-greedy [\s\S]*?.
**Solution:** Replace regex-based replace with index-based split/join: use IndexOf() to find marker positions, Substring() to extract before/after, and string concatenation to build the result. Also add -NoNewline to Set-Content to prevent trailing newline drift (2 bytes per run). Verified: 5 consecutive runs = 0 accumulation, 0 parse errors, constant size.
**Tags:** `#powershell` `#regex` `#replace` `#idempotency` `#profile` `#bug`
### [2026-05-27] PowerShell -replace with [\s\S]*? expands large strings instead of replacing

**Context**: `setup-windows.ps1` profile-section block used `-replace` regex to update dotfiles section in PowerShell profile.

**Problem**: PowerShell's `-replace` operator with `[\s\S]*?` regex pattern **expands** large strings (>10KB) instead of replacing. A profile with 1 marker and 0 errors became 4 markers and 5 errors after a single `-replace` run, then 30+ markers on subsequent runs. Both `-replace` and `[regex]::Replace` failed identically — same behavior.

**Solution**: Replace regex-based replace with index-based split/join: use `IndexOf()` to find marker positions, `Substring()` to extract before/after, and string concatenation to build the result. Also add `-NoNewline` to `Set-Content` to prevent trailing newline drift (2 bytes per run). Verified: 5 consecutive runs = 0 accumulation, 0 parse errors, constant size.

**Rule**: Never use PowerShell's `-replace` operator with `[\s\S]*?` on strings >10KB. Use `IndexOf()` + `Substring()` + concatenation instead.
### [2026-05-27] PowerShell -replace with [\s\S]*? expands large strings instead of replacing

**Context**: `setup-windows.ps1` profile-section block used `-replace` regex to update dotfiles section in PowerShell profile.

**Problem**: PowerShell's `-replace` operator with `[\s\S]*?` regex pattern **expands** large strings (>10KB) instead of replacing. A profile with 1 marker and 0 errors became 4 markers and 5 errors after a single `-replace` run, then 30+ markers on subsequent runs. Both `-replace` and `[regex]::Replace` failed identically.

**Solution**: Replace regex-based replace with index-based split/join: use `IndexOf()` to find marker positions, `Substring()` to extract before/after, and string concatenation. Also add `-NoNewline` to `Set-Content` to prevent trailing newline drift (2 bytes per run). Verified: 5 consecutive runs = 0 accumulation, 0 parse errors, constant size.

**Rule**: Never use PowerShell's `-replace` operator with `[\s\S]*?` on strings >10KB. Use `IndexOf()` + `Substring()` + concatenation instead.### [2026-05-27] Obsidian CLI package name @vorillaz/obsidian-cli does not exist on npm

**Context**: Both `setup-linux.sh` and `setup-windows.ps1` attempted `npm install -g '@vorillaz/obsidian-cli'` but the package returns 404 on npm registry.

**Problem**: The Obsidian CLI was referenced with a wrong package name (`@vorillaz/obsidian-cli`) that does not exist on npm. The correct package is `obsidian-cli` (no scope). Both Linux and Windows setup scripts silently failed to install it, causing `FAIL: Obsidian CLI not in PATH` in healthcheck.

**Solution**: Updated both setup scripts to use `npm install -g 'obsidian-cli'`. Updated all bats tests that grep for the old package name.

**Rule**: Always verify npm package names exist before committing them to setup scripts. `npm view <package> name version` before using in automated install.
### [2026-05-27] Split-Path -LiteralPath and -Parent are mutually exclusive parameter sets in PowerShell
**Context:** Fixing setup-windows.ps1 post-run errors. utils.ps1:58 used Split-Path -LiteralPath $Destination -Parent which threw Parameter set cannot be resolved.
**Problem:** Split-Path -LiteralPath and -Parent are mutually exclusive parameter sets in PowerShell 5.1 and 7.x. Using both together throws: "Parameter set cannot be resolved using the specified named parameters."
**Solution:** Split-Path has distinct parameter sets: LiteralPathSet (accepts -LiteralPath WITHOUT -Parent) and ParentSet (accepts -Path WITH -Parent). Replace -LiteralPath with -Path when using -Parent. The fix: $dstDir = Split-Path -Path $Destination -Parent (1 line change).
**Tags:** `#powershell` `#bugfix` `#cross-platform` `#split-path`

### [2026-05-30] Verify a plan's file-map against the code before editing load-bearing scripts
**Context:** SDD-008 skill pipeline (PR #179). The implementation plan specified which loops to remove from setup-linux.sh / setup-windows.ps1 to migrate skill deploy to render-at-deploy ("agy ~L428, claude ~L505").
**Problem:** The plan's file-map was incomplete/wrong: the loop that actually produced the symlinks AC1 forbids was the vault-skill symlink loop (setup-linux.sh:1087 / setup-windows.ps1:594), which the plan did not mention. There were 5 skill-deploy loops per OS, not 2. Editing only the loops the plan named would have left AC1 (zero symlinks) silently failing, and opencode deployed from a separate pipeline (skills-to-opencode.sh -> ai/opencode/commands) the plan also missed.
**Solution:** Before editing, grep the actual code for every site that touches the thing being migrated (here: every loop writing to ~/.claude/skills, ~/.config/opencode/commands, ~/.gemini/*, and every vault-symlink loop) and reconcile that against the plan. Treat a plan's line numbers / file list as a hypothesis to verify, not ground truth. A wrong file-map makes an acceptance criterion fail silently because the obvious edits look complete.
**Tags:** `#sdd` `#verification` `#deploy` `#shell`

### [2026-05-30] A whole-file transform must inspect the data shape before assuming a uniform model
**Context:** SDD-008: migrating skill deploy to render-at-deploy. The render kind 'skill' was first implemented to render only SKILL.md to the agent path.
**Problem:** Two vault skills carry auxiliary files (systematic-debugging has 5: scripts + reference .md; test-driven-development has 1). The old claude deploy did `cp -rf "$skill_dir"*` (whole directory), so a SKILL.md-only render would have silently dropped those reference files — a functional regression invisible to a test that only checks SKILL.md.
**Solution:** Before committing to a transform model, enumerate the inputs (here: `find vault/00_meta/skills -type f ! -name SKILL.md`). For dir-based renders, copy the whole record dir then overlay the rendered SKILL.md; single-file renders (opencode command, agy prompt) legitimately take only SKILL.md. Add a test that asserts an auxiliary file lands at the dir-based target and does NOT at the single-file targets.
**Tags:** `#sdd` `#skills` `#deploy` `#testing`

### [2026-05-30] Don't commit to a shared vault that has another session's staged work — use an isolated worktree
**Context:** SDD-011 close-out. The vault (`~/Projects/knowledge`) was being edited in parallel by another session that had a feature branch (`rfd-001/pdf-modifier-mcp`) checked out with *staged* work (a pollex restructure). I committed a scoped 2-file SDD-011 change there.
**Problem:** A scoped `git commit -- <paths>` correctly isolated my 2 files (it did NOT sweep the other session's staged work), but the commit landed on the *other session's* feature branch. When that session switched branches, the vault working tree reverted and my change was stranded off `master` (the canonical branch). Committing into a repo another agent/human is actively branch-switching is racy and misplaces the work.
**Solution:** When you must read or operate on a specific commit of a repo another session is using, use `git worktree add --detach <tmp> <commit>` — an isolated checkout that shares only the object store, never the live HEAD/index. To place a change on the right branch without disturbing the active checkout: `git worktree add <tmp> master` + `git cherry-pick <commit>`, then `git worktree remove`. Detect the hazard early: a `git status` showing *staged* changes you did not make means another session owns this checkout — do not commit there.
**Tags:** `#vault` `#git` `#worktree` `#parallel-sessions` `#sdd`

### [2026-05-31] A structural integrity guard surfaces latent issues you didn't know you had
**Context:** SDD-012 built a backlog-integrity guard (`check-backlog-integrity.sh`) to stop the dotfiles `11-tasks.md` from drifting (the same ticket listed twice with diverging status).
**Problem:** Run on the real file, the guard flagged not only the expected view-duplication but **9 number collisions** — two *different* tickets sharing one number (`BUG-020-pwsh` vs `BUG-020-splitpath`, `IDEAS-007-promote` vs `IDEAS-007-cross-provider`, ...). Nobody had noticed; enforcing the invariant exposed them. It also forced a guard-design refinement: match by FULL id (slug-aware), not by number, so legitimate-but-untidy reuse becomes an advisory NOTE instead of a false "duplicate" that would demand history-desyncing renumbers.
**Solution:** When you add a structural guard (incident→guard), expect it to surface adjacent latent issues beyond the bug class it targets — budget for that, and decide up front which are hard-fails vs advisories. Match the guard to the real identity of the thing (full id), distinguishing "same thing listed twice" (drift → hard-fail) from "two things share a label" (collision → advisory). An over-strict guard demands harmful fixes.
**Tags:** `#sdd` `#incident-to-guard` `#backlog` `#guard-design`

### [2026-05-31] A SCRIPT_DIR root-resolution fix breaks CWD-pinned fixture tests — add an env-override seam, not a code branch
**Context:** PR #192 changed compile-harness.sh root resolution from `git rev-parse --show-toplevel` (CWD-based) to SCRIPT_DIR-based (`cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.."`), to fix the section-12 healthcheck drift false-fail on the non-git ~/.dotfiles copy-deploy (ADR-012), where `git rev-parse` errors.
**Problem:** 26 compile-harness.bats tests run the REAL script against a throwaway fixture they `cd` into, relying on CWD-based (git-toplevel) root resolution to make the fixture the root. SCRIPT_DIR resolution instead pointed the script at the LIVE repo, so --refresh/--deploy operated on the wrong tree → mass failure. The two goals (robust resolution for the deploy copy vs. arbitrary fixture root for tests) are irreconcilable with a single hard-coded REPO_ROOT. A second, brittle test also broke: a guard that greps healthcheck.sh for an exact failure-message string, after #192 reworded the message.
**Solution:** Introduce an explicit env override: `REPO_ROOT="${HARNESS_REPO_ROOT:-$(SCRIPT_DIR-based)}"` — the SAME idiom the script already uses for VAULT_PATH. Tests `export HARNESS_REPO_ROOT="$REPO"` once in setup() to pin the fixture; production never sets it and keeps the SCRIPT_DIR default. One seam, two correct behaviors, zero production code branch. Generalize the existing override idiom rather than special-casing tests. Corollary: prefer grepping a stable SUBSTRING of a message (or asserting behavior) over an exact full-string match, so message rewording doesn't break the guard.
**Tags:** `#testing` `#bats` `#root-resolution` `#deploy-model` `#env-override` `#compile-harness`

### [2026-05-31] A skip-guarded test is green in CI but a real assertion locally — it can hide a genuine cross-OS parity gap
**Context:** While running the full bats suite locally to verify PR #192, tests/antigravity.bats test 64 ("setup-windows.ps1 syncs Shared Skills to ~/.gemini/skills") FAILED locally, yet the very same commit was GREEN in CI.
**Problem:** The test carries `skip "agy not installed (CI / fresh machine)"`. On CI (fresh runner) agy is absent → the test skips → counts as `ok`. On a dev box with agy installed, the skip condition is false → the real assertion runs → red. The assertion is a static grep proving setup-windows.ps1 contains the `sharedSkillsDir` / "Synced Shared Skills to" sync that setup-linux.sh has; those strings are currently ABSENT from setup-windows.ps1. So a genuine Windows-parity gap (or a stale test left behind when SDD-008's compile-harness --deploy subsumed the old skills-sync block) is invisible in CI and surfaces only locally.
**Solution:** Treat skip-guarded tests as CI-BLIND: green-in-CI does NOT mean the contract holds when the skip condition is "tool not installed on CI". For a cross-OS-parity-of-SOURCE assertion (static grep of a script's text), there is no reason to skip on a missing runtime tool — gate it on file presence, not on `agy` being installed, so CI actually runs it. Reserve skips for tests that exercise the tool's RUNTIME behavior. When such a local-only red appears, record it as an open thread / Windows-empirical ticket rather than assuming the suite is clean.
**Tags:** `#testing` `#ci` `#skip-guard` `#cross-os-parity` `#false-green` `#windows`

### [2026-05-31] Onboarding a junior/remote agent: verify its self-authored docs, enforce boundaries mechanically
**Context:** HERMES-001 — integrating Hermes, a low-capability remote ops agent (Debian 13 on NaN infra, Telegram), into the dotfiles ecosystem via ai/hermes/setup.sh + a curated vault SSOT at 80_agents/hermes-nan/.
**Problem:** The agent's own self-authored vault docs had silently drifted from reality: validate.sh checked filenames (memory.md, skills.md) the folder no longer used (numbered 10-memory.md, 11-skills.md), a constitution AGENTS.md was referenced everywhere but did not exist, and the vault clone lived in ephemeral /tmp so a reboot lost the auto-push git hook. Separately, the write-zone boundary (commit only within 80_agents/) was instruction-only — a junior agent can ignore instructions, and prompt-injection or a bug could push anywhere.
**Solution:** Treat a junior/remote agent's own docs as CLAIMS to verify, not truth. Probe the real box before designing provisioning — config path (/hermes-home/config.yaml, not ~/.hermes/), the commit mechanism (git CLI vs MCP — determines whether git hooks fire), and credential handling (token was embedded in the remote URL) were ALL different from assumptions; one probe round saved a wrong design. Convert soft instruction-only boundaries into MECHANICAL guardrails: once the probe confirmed Hermes commits via git CLI, install local git hooks in its clone — pre-commit rejecting paths outside the write-zone + token-like content, pre-push rejecting non-fast-forward (force) pushes — each with a functional test. Hooks are local to the agent's clone (never tracked/cloned), so they harden one consumer without touching others.
**Tags:** `#hermes` `#agent-onboarding` `#guardrails` `#remote-agent` `#verify-before-trust` `#git-hooks`

### [2026-05-31] A cross-environment SSOT validator must split "content drift" (fail) from "runtime absent off-box" (warn)
**Context:** HERMES-001 Track B. `80_agents/hermes-nan/scripts/validate.sh` checks the vault SSOT for the Hermes agent; AC6 is "vault SSOT consistent, validate.sh green". The script also checked box-only runtime facts (post-commit hook, cron entry, `uvx`).
**Problem:** Those runtime checks only pass on the provisioned Hermes box. As hard failures they made AC6 unprovable anywhere else — the script could never be green from a dev checkout or CI, so "green" had no portable meaning.
**Solution:** Split the exit policy by what the artifact is SSOT for. `fail` (exit 1) = vault SSOT inconsistency (a required file missing/malformed), content the vault owns and that holds everywhere. `warn` (exit 0) = box-runtime advisory (hook/cron/tool), environmental and absent off-box. A green run then means "SSOT internally consistent" regardless of where it runs, and the same script still does the full check on the box. Generalize: a validator that mixes content invariants with environment state must rank them, or it asserts nothing portable.
**Tags:** `#validation` `#ssot` `#hermes` `#exit-codes` `#cross-environment`

### [2026-06-02] A harness consumer spec scoped on a retired axis must be reconciled, not implemented (WORKMODE-001)

**Context:** Picking up the HARNESS-001 backlog, the next planned consumer was WORKMODE-001 (#159) — "the harness adapts its knowledge-SSOT target to repo type (personal → vault, work → repo + Project)". Before implementing, a vault+repo sweep checked whether the work/personal model had already been decided.

**Problem:** It had. `pattern-knowledge-placement` (KPM-001, 2026-05-28) replaced the **work/personal** axis with a **decide-vs-operate-by-layer** axis: build/operate docs → repo `docs/` for *every* placement-model repo; cross-project brain → vault; collaborate → forge. WORKMODE-001's premise ("personal → vault") was the retired axis. Implementing it as written would have hard-coded the obsolete split into the deploy engine — the opposite of the fix. Worse, `AGENTS.md` (the cross-agent SSOT) still carried the old axis in Standing Orders #2/#3/#7 + the Document Dynamic / Lessons sections, contradicting its own Neural Hive section and mis-routing artifacts (the kubelab regression: a personal+placement repo's lesson sent back to the vault). The open reconciliation ticket #197 already named this.

**Solution:** Fuse #197 + #159 into one spec that retires the axis instead of cementing it: make decide-vs-operate primary across all of AGENTS.md, demote work/personal to its one residual (where the cross-project *brain* lives + whether tasks sit on a shared board) as a defaulted `## Knowledge Placement` declaration, re-key tooling guards from "for work projects" to "for any placement-model repo", and ship an incident→guard (`tests/agents-md.bats`) that fails if any build/operate class is routed to the vault again. Verify a held spec's *premise* against later patterns/ADRs before writing code — a spec is a hypothesis with a shelf life, not a work order.

**Tags:** `#sdd` `#knowledge-placement` `#decide-vs-operate` `#agents-md` `#ssot` `#incident-to-guard` `#reconcile-dont-implement`

### [2026-06-01] Archived-spec ≠ issue-complete; verify "high-value open" items against git before implementing
**Context:** 2026-06-01 backlog-reconciliation session. Picked the top "high-value open" backlog items to implement (BUG-024, SDD-009, #156).
**Problem:** All three "high-value open" items were already shipped and merged — never ticked in the vault or closed on GitHub (backlog over-reported pending work ~3×). Worse, when reconciling GH issues I closed #193 (HERMES-001) on the archived-spec signal, but the user flagged the agent box still needs bootstrap/config/backups — the issue tracked broader operational scope than its archived spec, so the close was premature.
**Solution:** Before implementing any "open" backlog item, verify against git/PRs/archived-specs first (extends [[pattern-verify-against-source-of-truth]]). Two reliability tiers: archived-spec = deterministic "this spec shipped" (full-id keyed, dodges ticket-number reuse); PR-title match = advisory/brittle. But archived-spec proves the SPEC merged, NOT that the broader ISSUE/epic is operationally complete — an issue (esp. "add agent X to ecosystem") can outscope its spec. So: tick vault + close issues on archived-spec, but only after confirming the issue's scope == the shipped work; when unsure, ask the owner (the #193 reopen). Also: GH issues with no TICKET-NNN prefix escape id-keyed sweeps (#156 slipped) — a content-level sweep catches those. Mechanical enforcement shipped as scripts/check-backlog-merged.sh (SDD-012b, PR #203).
**Tags:** `#backlog` `#reconciliation` `#verify-before-act` `#drift` `#process`

### [2026-06-01] A held spec can be obsoleted by a later ADR — reconcile+close, don't implement-as-written
**Context:** Picked up IDEAS-007 (#103, filed 2026-05-27) to implement a 4-layer cross-provider agent harness (.agent/<id>/INSTRUCT.md design). Ran verify-before-act against git/specs before coding.
**Problem:** The spec's architecture had already shipped by other means AFTER it was written: ADR-009 (AGENTS.md SSOT) + ADR-010 (parity) + the ai/<provider>/ overlay structure realised Layers 1-2; the L3 registry + runtime discovery mechanism had zero consumer. Implementing the spec literally would have manufactured debt — a binary-name->provider detector nothing calls, plus a churning rename of a deployed convention wired into setup-linux.sh (~6 sites) + healthcheck.
**Solution:** Reconcile, don't re-implement. Produce evidence (audit.json: rule-by-rule classification confirming the split is already correct; reconciliation.md: criterion-by-criterion disposition) and close the issue with that evidence. Reject the no-consumer layers explicitly as YAGNI (Decision Hierarchy: Explicit > Implicit). Spin off the one genuine win found (data-driven setup-linux.sh provider-deploy manifest) as its own deferred ticket to keep the PR atomic. This is the IMPLEMENT-side mirror of feedback_check_existing_artifacts_first (which is propose-side) and another instance of pattern-verify-against-source-of-truth: verify the spec's PREMISE is still valid before writing code, because a later ADR can silently obsolete an open spec.
**Tags:** `#sdd` `#verify-before-act` `#yagni` `#reconciliation`

### [2026-06-02] Broad sed over a backlog ticks substring mentions, not just the entry — anchor to the line-start id
**Context:** Reconciling the vault backlog (11-tasks.md) during a drift cleanup; used sed to tick the shipped IDEAS-007 entry: sed 's/^- \[ \] \(.*IDEAS-007\)/- [x] \1/'.
**Problem:** The pattern matched ANY '- [ ]' line CONTAINING "IDEAS-007", not just its own entry. It also ticked the HARNESS-001 entry (its body references IDEAS-007) and — wrongly — REFACTOR-008, a deferred-with-trigger YAGNI item whose body merely cites the IDEAS-007 reconciliation doc. One factually-incorrect tick (REFACTOR-008 is NOT done). The SDD-012b merged-but-open guard caught the IDEAS-007 case but not the over-ticks.
**Solution:** Anchor backlog seds to the entry's own id at line start — '^- \[ \] \*\*<ID>' — never a bare substring that can appear in prose. ALWAYS verify with `git diff | grep '^[+-]- \['` to see every checkbox the command flipped BEFORE committing, and re-run the integrity/merged-open guard after. Broad find/replace on a human-curated list that mentions ids in prose is a footgun.
**Tags:** `#backlog` `#sed` `#reconciliation` `#footgun` `#verify-the-diff` `#red-team-thyself`

### [2026-06-03] A late push to a PR branch can miss the squash — verify the commit is on the PR head, and each deliverable is on main post-merge
**Context:** AI-020. Opened PR #216 with the decision commit, then pushed a SECOND commit (the migration runbook, AC6) to the same branch. The user clicked "Update branch" in the GitHub UI and squash-merged.
**Problem:** The squash captured only what the PR head had at merge time — the decision + AI-023 scaffold — NOT the runbook. The UI "Update branch" + concurrent merge consolidated a head that raced my runbook push, so the runbook silently never reached main. Nothing errored; AC6 was just absent. Caught only by an explicit post-merge `git cat-file -e origin/main:docs/runbooks/...` check, AFTER I'd already torn down the worktree + branch (`-D`).
**Solution:** After adding a commit to a PR that may merge imminently — especially when the other party is merging concurrently — confirm it actually landed on the PR head (`gh pr view <n> --json commits` shows the SHA) before treating it as shipped. When closing a multi-commit PR, verify each deliverable file exists on `origin/main` post-merge; don't assume the squash took everything. Recovery is cheap because the orphaned commit survives in the local object store (`git cat-file -e <sha>` → cherry-pick onto fresh main), but only if you notice — so delay `git branch -D` of a just-merged branch until the deliverables are confirmed on main.
**Tags:** `#git` `#pr` `#squash-merge` `#verify-before-completion` `#race` `#red-team-thyself`

### [2026-06-07] Re-running a failed Actions run replays the *original commit's* workflow file
**Context:** The "Add to bitácora" workflow had failed repeatedly on `actions/add-to-project@v1` (a tag that does not exist). After pinning `@v1.0.2` on master, the temptation was to re-run the failed runs to confirm the fix.
**Problem:** `gh run rerun <id>` (and the UI "re-run") replays the workflow definition from the **commit the run was originally triggered on**, not from current `main`. Re-running the old failures would have re-executed the broken `@v1` pin and failed again — "proving" nothing and looking like the fix did not work.
**Solution:** Verify a workflow-file fix by triggering a **fresh event** on the patched ref (open/assign a throwaway issue, push a trivial commit), not by re-running historical failures. Delete the throwaway afterward.
**Tags:** `#github-actions` `#ci` `#rerun` `#verify-before-completion`

### [2026-06-07] `gh project --owner` → "unknown owner type" under a fine-grained PAT; a green CI is not a green workflow
**Context:** HARNESS-010's `bitacora-status.yml` used `gh project item-add/item-edit --owner mlorentedev` to move an assigned issue to In Progress. `actionlint`, `test`, and `spec-gate` were all green, and the same commands worked when run with my local `gh` auth.
**Problem:** On the first real assignment the workflow failed at runtime with `unknown owner type`. The `gh project` CLI resolves the owner (user vs org) via an API call the fine-grained `BITACORA_PAT` cannot satisfy — so it works with a local token but not with the workflow's secret. No CI job exercises the workflow with the real secret, so CI was green while the workflow was broken.
**Solution:** Drive Projects v2 from `actions/github-script` (or raw `gh api graphql`) — `addProjectV2ItemById` + `updateProjectV2ItemFieldValue`, the same path `actions/add-to-project` uses — which does not depend on owner-type resolution. And validate any secret-dependent workflow end-to-end with the **real secret** (a throwaway trigger), not local credentials: "mechanism proven locally" ≠ "workflow proven". (Corollary: do not silence a linter false-positive — e.g. `SC2016` on GraphQL `$vars` in single quotes — with scattered `# shellcheck disable`; pick a form the linter understands, like `github-script` for GraphQL.)
**Tags:** `#github-actions` `#projects-v2` `#fine-grained-pat` `#gh-cli` `#verify-with-real-secret` `#no-kludges`

### [2026-06-09] setup-time `compile-harness.sh --refresh` leaves silent drift -- surface it, don't delete it
**Context:** A live `setup-linux.sh` run left `harness/skills/handoff/SKILL.md` modified in the working tree. It read like a parallel session's WIP and derailed reasoning about an unrelated fast-forward pull (OPS-001, #295) -- time was spent treating a designed signal as mystery dirt.
**Problem:** `setup` runs `compile-harness.sh --refresh` whenever the vault is present, regenerating the committed `harness/` records (and the generated blocks in `AGENTS.md` / `ai/claude/CLAUDE.md`) from the vault SSOT. When the vault is ahead, that leaves uncommitted changes -- but setup only logged a success line, so the drift was invisible-as-intent. The instinct ("just delete the `--refresh` from setup to stop the dirt") is a trap: the dotfiles CI has **zero visibility into the private vault** (ADR-013), so `--refresh` is the *only* thing propagating vault skill edits into the committed records. Removing it = silent vault<->repo divergence -- exactly the bug class SDD-008's "the build IS setup" design exists to prevent.
**Solution:** Treat the drift as a signal, not noise. OPS-003 (#307 -> #308): after a successful `--refresh`, setup checks the working tree and, if records changed, **warns loudly** with the file list + the exact commit command -- turning silent drift into an actionable `chore(harness): refresh records from vault` commit. Keep the mechanism; make it announce. The directionality is the design: vault = authoring SSOT, `harness/` = committed cache for offline deploy + CI. Before deleting any setup step that "dirties" the tree, ask what it propagates -- here it was load-bearing.
**Tags:** `#setup` `#harness` `#compile-harness` `#drift` `#vault-ssot` `#generate-and-commit` `#verify-before-act` `#dont-delete-load-bearing-code`

### [2026-06-10] Frontmatter must be strict-YAML clean — the most lenient parser in the fleet is not the contract
**Context:** pi v0.79.1 flagged two deployed skills (`spec`, `architecture-session`) as **Skill conflicts** at startup: "Nested mappings are not allowed in compact mappings". Both had `description:` values containing `: ` sequences (e.g. "Four subcommands: init (...)") without quoting. Claude Code had consumed those SKILL.md files for weeks without complaint.
**Problem:** Cross-agent artifacts (skill frontmatter rendered by the harness to claude/opencode/agy/pi) are parsed by N different YAML implementations. Authoring against the *most lenient* consumer (Claude's parser) lets latent violations accumulate; the contract silently becomes "whatever the loosest parser accepts" until a strict consumer joins the fleet (pi) and surfaces them all at once.
**Solution:** Quote any frontmatter scalar containing `: `, `#`, or leading/trailing specials at the **vault SSOT** (fixed in vault `75fe67c`, propagated via `compile-harness --refresh`, PR #316). Validate with a real YAML parser (`python3 -c "yaml.safe_load(...)"`), not by eyeballing. When a registry serves N consumers, the strictest parser in the fleet defines the format contract — sweep the whole catalog when one file fails, not just the reported one (here a sweep of all 32 skills confirmed only 2 were affected).
**Tags:** `#yaml` `#frontmatter` `#skills` `#cross-agent` `#harness` `#strictest-consumer-wins` `#vault-ssot`

### [2026-06-10] Deploying workflow files to N repos via the contents API: three gotchas the happy path hides
**Context:** OPS-002's `scripts/bitacora-rollout.sh` converges every non-archived, non-fork repo to the bitácora baseline — it pushes the two canonical workflows (`add-to-project.yml`, `bitacora-status.yml`) into each repo via the GitHub contents API and backfills open issues/PRs onto the board. It worked end-to-end against dotfiles/knowledge, then broke in three distinct ways the moment it met the rest of the fleet.
**Problem:** (1) **Protected branches reject a direct `PUT /contents`** with HTTP 409 — repos with branch protection (kubelab, pollex, pdf-modifier-mcp, yt-metrics-cli) cannot take a straight commit to the default branch. (2) **The deployed `add-to-project.yml` ran on fork PRs**, where `pull_request` from a fork has no access to repo secrets, so the job failed secretless on every fork-originated PR (Codex flagged it as P2 on kubelab#237). (3) **Reusing a fixed branch name (`ci/bitacora-workflows`) across runs** meant that after a run's PR was squash-merged, the next run pushed onto the now-diverged stale branch and opened a **conflicting** PR (Manu caught this on pollex/yt/pdf).
**Solution:** All three fixed in the canonical script + propagated to the deployed copies: (1) detect the 409 and **fall back to branch + PR** (with an auto-merge attempt) instead of a direct push (#317); (2) add a **job-level fork guard** (`if: github.event.pull_request.head.repo.fork == false`) so the secret-dependent job is skipped on fork PRs (#320); (3) **force-reset the working branch to base HEAD** at the start of each run so a reused branch never carries stale diff (#322). The convergence mechanism for any later workflow-template change (or PAT rotation) is simply re-running the script — it is idempotent, second run = `0 changes`.
**Tags:** `#github-actions` `#contents-api` `#protected-branches` `#fork-pr-secrets` `#idempotence` `#multi-repo` `#iac` `#bitacora`

### [2026-06-10] Tests aimed at a runner that doesn't exist yet are dead weight — and "if available" guards rot silently
**Context:** WIN-004 (PR #325) added the first `windows-latest` CI job, finally executing `setup-windows.ps1` end-to-end plus the Pester and PowerShell-bats suites. `tests/sdd-009-deploy-time-secrets.Tests.ps1` had declared "WIN-004 will pick this up in CI" in its header and sat unexecuted for two weeks; the bats PSScriptAnalyzer/syntax tests only ran where pwsh happened to exist.
**Problem:** Four consecutive live runs each surfaced a real latent bug that no installed box could reproduce: (1) `$tool.Version` under StrictMode killed installs on clean machines; (2) `DOTFILES_DIR`/`DOTFILES_REPO_DIR` defaults pointed at paths that don't exist on runners; (3) Pester `-Skip` conditions evaluate at discovery, before `BeforeAll` runs; (4) MSYS paths inside quoted pwsh `-Command` strings bypass Git Bash auto-conversion and resolve against the drive root (`D:\d\a\...`). Worse, the knowledge-crystallize analyzer test had escaped its variables (`'\$PS1_SCRIPT'`) and was analyzing an empty path — its catch-block `exit 0` made it pass everywhere, forever, without analyzing anything.
**Solution:** Land the runner with the first test that targets it, not after a backlog of "will be picked up later" suites — every deferred suite is unverified code wearing a green badge. Audit `catch { exit 0 }` / "if available" guards: a test that can't fail is documentation, not verification (the healthcheck variant that exits 1 in its catch is what exposed the path bug). For Git Bash → native pwsh boundaries, convert paths explicitly (`tests/winpath.bash`, `cygpath -w`) — auto-conversion only applies to plain arguments, never inside quoted command strings.
**Tags:** `#ci` `#windows` `#pester` `#bats` `#msys` `#silent-failure` `#dead-tests` `#verify-before-completion`

### [2026-06-12] goreleaser monorepo.tag_prefix is Pro-only — verify paywalled features empirically

**Context**: CLI-001 scaffold (ADR-020): configuring goreleaser for the nested `cli/` Go module with `cli/vX.Y.Z` release tags.

**Problem**: Trained memory and most blog posts present `monorepo.tag_prefix` as a goreleaser feature; it is GoReleaser **Pro**-only. OSS silently treats a prefixed tag as the literal version (`cli/v0.0.1`), and the slash corrupts artifact paths (`dist/dot_cli/v0.0.1_...`).

**Solution**: Exercised the release pipeline locally with a throwaway tag + the OSS binary BEFORE the first real release; switched to plain `v*` tags (the CLI is the repo's only released artifact) and documented the revisit condition in the spec (CLI-001 R2).

**Rule**: Before designing around any third-party tool feature, check the OSS/paid feature split in current docs (Context7) AND exercise the pipeline empirically with a throwaway run. Feature paywalls invalidate trained memory silently.

### [2026-06-12] Non-streaming chat endpoints behind a gateway drop long generations — a client timeout cannot fix a server-side cut

**Context**: CLI-003 `dot review` QA: live review of a real 12KB staged diff through the NaN gateway (`deepseek-v4-flash`, non-streaming chat completions).

**Problem**: The gateway closes long non-streaming responses mid-generation. Reproduced at the 120s client timeout and again at 300s (TCP read died at ~168s) — the cut is provider-side, so no client-side `--timeout` value can help. Hello-world smoke tests never trigger it; the failure only appears at realistic payload sizes.

**Solution**: Kept the 120s default instead of chasing the timeout; documented `--provider openrouter` as the escape hatch for large diffs (same 12KB diff reviewed in ~10s) and recorded the limitation in `cli/README.md`.

**Rule**: QA API integrations with realistic payload sizes, not hello-world ones. When a remote endpoint drops long responses, change the route (streaming, different provider) — not the client timeout.

### [2026-06-13] `gh project item-list` truncates to `--limit` silently — check `totalCount` before asserting absence

**Context**: Verifying whether issues #344/#347/#350 had landed on the bitácora Projects v2 board after an `item-add`. Listed the board's items and grepped for the issue numbers; they were absent from the output, so the working diagnosis became "the add failed, the issues are missing from the board".

**Problem**: `gh project item-list` paginates with a default `--limit` of 30 and returns only that page — items beyond the limit are omitted with no warning and no non-zero exit. The board already held more than 30 items, so the issues were present but off the returned page. A grep over the truncated list "proved" an absence that was really just pagination. The JSON form exposes a `totalCount` that exceeded the returned `items | length`, but the naive list never compared them.

**Solution**: Before concluding an item is absent from a Projects v2 board, reconcile counts: `gh project item-list <n> --owner <o> --format json --limit 1000 | jq '{returned: (.items|length), total: .totalCount}'` and only trust an absence when `returned == total` (page exhausted). Pass a `--limit` >= `totalCount`, or page through, rather than grepping the default page.

**Rule**: Any CLI whose list command has a default `--limit` can answer an existence/absence question *wrongly* once the collection outgrows one page. Treat "not in the output" as "not on this page" until `totalCount` (or exhaustive paging) confirms the page was complete. Silent truncation reads as "covered everything" when it didn't.

### [2026-06-13] A bats teardown's last command classifies even *skipped* tests — never end it with a bare `[ cond ] && cmd`

**Context**: Six `tests/claude-mem-heal-ps1.bats` tests reported as `not ok N … # skip` locally (no `pwsh` installed), which reads as six failures. CI was green because the runner has `pwsh`, so the tests actually run instead of skipping. This failure-shaped skip had previously derailed reasoning about local suite health.

**Problem**: `setup()` calls `skip "pwsh not available"` *before* it assigns `TMP=$(mktemp -d)`. bats still runs `teardown()` after a skipped test, and the teardown's only line was `[ -n "${TMP:-}" ] && rm -rf "$TMP"`. With `TMP` unset, `[ -n "" ]` exits 1, the `&&` short-circuits, and teardown's *last* command therefore exits non-zero — so bats reclassifies the clean skip as `not ok # skip`. The same skip-fragile last-line pattern (`[ -n "$VAR" ] && rm -rf "$VAR"` as the final teardown statement) existed in ~8 test files; it only bites where a test can `skip` before the guarded var is set.

**Solution**: Invert the guard so both branches exit 0: `[ -z "${TMP:-}" ] || rm -rf "$TMP"` (empty → `[ -z ]` true, short-circuits at exit 0; set → runs `rm`, exit 0). Applied across the affected teardowns; the six heal-ps1 entries flip from `not ok # skip` to `ok # skip`.

**Rule**: A teardown's final command determines the test's exit classification — for passing, failing, *and* skipped tests alike. Never end a teardown with a bare `[ cond ] && cmd`; invert to `[ ! cond ] || cmd` or append an explicit `return 0`. A `skip` that fires before `setup()` finishes leaves cleanup vars unset, so the cleanup guard must not itself be able to fail.

### [2026-06-13] Sourced-vs-executed guard: use `(return 0 2>/dev/null)`, not a `BASH_SOURCE`-vs-`$0` compare

**Context**: CLI-009 `scripts/install-dotf.sh` (named `install-dot.sh` until the CLI-010 rename) is both *sourced* (by `setup-linux.sh` and by its bats test) and *executed* directly (standalone `./install-dotf.sh` upgrade). It needs a guard so `install_dotf "$@"` runs only on direct execution, not on source.

**Problem**: The first guard was the common `if [ "${BASH_SOURCE[0]:-$0}" = "$0" ]`. It fired on *source* in the Bash-tool / harness context — `install_dotf` ran at source time with empty `$@`, printing a spurious `no version given` error (and would side-effect under setup). The `${BASH_SOURCE[0]:-$0}` fallback also makes it wrong under zsh: `BASH_SOURCE` is unset there, so the expansion becomes `$0` and the comparison is trivially true whether sourced or executed. A standalone diagnostic with `bash -c '. probe.sh'` showed "not equal" (correct) while the actual harness invocation showed it firing — i.e. the idiom's correctness is context-dependent, which is itself disqualifying for a guard.

**Solution**: `if ! (return 0 2>/dev/null); then install_dotf "$@"; fi`. `return` is only valid in a sourced script (or function), so the subshell exits 0 when sourced and non-zero when executed — a context-independent signal. Verified across `bash -c` source, script-source (the setup path), and direct execution.

**Rule**: To gate a script's "run only when executed directly" block, use `(return 0 2>/dev/null)` (sourced → succeeds, executed → fails), not a `BASH_SOURCE`-vs-`$0` string compare. The string compare aligns the two values in some shells (zsh, where `BASH_SOURCE` is unset) and some harnesses, so it fires on `source` and auto-runs the script's main path as a side effect.

### [2026-06-13] A migration's acceptance guard-grep is the completeness oracle, not the spec's hand-listed targets

**Context**: CLI-005 retired the `init-spec`/`archive-spec` shell twins and repointed every reference to `dotf spec`. The proposal enumerated five repoint targets by hand (AGENTS.md, `agents-md.bats`, `check-spec-gate.sh`, the spec `SKILL.md`, the architecture-map).

**Problem**: The hand-list was incomplete. The acceptance criterion's own guard — `grep -rE 'init-spec|archive-spec'` must return only historical artifacts — surfaced **two more live references the list missed**: a comment in `scripts/check-md-escapes.sh` and two lines in `harness/skills/adversarial-review/SKILL.md` that named `archive-spec.sh` as a command to run. Shipping the spec's list verbatim would have left broken references that no test named.

**Solution**: Treat the acceptance guard-grep as the authority and run it *before* claiming done, then repoint whatever it returns until only provenance (CHANGELOG, ADRs, lessons, `specs/`) remains. The hand-list is a starting hypothesis; the grep is the proof.

**Rule**: When a change's acceptance criterion is "no live reference to X remains except historical," the grep that expresses it is the completeness oracle — not the enumerated edit list in the proposal. Run it as a gate, not an afterthought; it finds the surfaces a human inventory forgets.

### [2026-06-13] Editing a committed render without its source-of-truth is a half-migration that `--refresh` reverts

**Context**: CLI-005 repointed `harness/skills/spec/SKILL.md` and `harness/skills/adversarial-review/SKILL.md` to `dotf spec`. Those files are committed *renders*: `compile-harness.sh` (SDD-008) treats the vault `00_meta/skills/<name>/SKILL.md` as the edit-SSOT and regenerates `harness/skills/` from it via `--refresh`.

**Problem**: Editing only the committed render leaves the vault sources stale. The render is correct for CI `--check`/`--deploy`, but the next `compile-harness.sh --refresh` pulls the unchanged vault source and silently reverts the repoint — a green PR that re-introduces the dead references on the next harness refresh.

**Solution**: Sync the vault sources in lockstep with the render. On the interactive machine, vault edits land on `origin/master` via obsidian-git's periodic auto-commit — just edit the files, no manual git. Verify the vault sources no longer carry the old reference before declaring the migration done.

**Rule**: For any file that is a generated/committed render, a migration is only complete when its *source-of-truth* changes too. Identify the generator (here `compile-harness.sh`) and edit upstream, or the render's change is transient. Sibling of the GEMINI→AGY incomplete-migration lesson: a repoint that leaves a caller — or a generator's source — stale is a half-migration.

### [2026-06-14] Deleting one OS twin while keeping its sibling forces asymmetric parity tests — rewrite them to the migration reality, don't fake symmetry

**Context**: CLI-012 ported the Linux diagnostics twins (`healthcheck.sh`, `doctor.sh`) to a cross-compiled `dotf doctor` and deleted them, but kept the `.ps1` siblings because `dotf` is not yet installed on Windows (no Windows `install-dotf`).

**Problem**: A pile of cross-OS bats encoded `.sh`↔`.ps1` symmetry ("parity: both healthchecks include BUG-015", "parity: both doctors check min_version"). Deleting only the `.sh` breaks the `.sh` half of every one, and the pure `.sh`-structural greps (`healthcheck.sh has 12 sections`, the BUG-023 probe shape) become dangling. Keeping them green by quietly dropping the `.sh` line leaves a test still *named* "parity: both…" that now checks only one OS — a lie in the suite.

**Solution**: Rewrite each parity test to the actual migration state — the Linux side asserts the `dotf doctor` wiring (or its intent moves to `go test`), the Windows side keeps its `.ps1` assertion, and the test name says which ("healthcheck.ps1 includes BUG-015…", "(Windows port pending)"). Pure `.sh`-structural tests are deleted outright; their behavioural intent lives in the Go table tests.

**Rule**: When a strangler-fig port deletes one OS's twin but can't yet delete the other, the cross-OS parity tests are no longer true — don't paper over them by dropping a grep line under an unchanged "both…" name. Rewrite them to the asymmetric reality so a reader sees the migration window, and migrate the deleted side's intent to the new test home. One-OS-per-PR is the cleaner unit, and the tests should announce which OS is done.

### [2026-06-14] A consolidated diagnostic that shells out to a generator is on-demand-cheap but per-event-expensive

**Context**: `dotf doctor` (CLI-012) consolidates the 12-section healthcheck. One section gates on `compile-harness.sh --check`, which re-renders every skill record offline. The retired `claude-session-start.sh` used to run a light, env-contract-only `doctor.sh` on every Claude session start.

**Problem**: Repointing the per-session hook to the full `dotf doctor` would fork a ~2.8s sweep on **every** session start (the `compile-harness --check` re-render dominates the time), and a PATH-command call would also break the hermetic isolation the session-start test relies on — that test copies only the hook into a temp dir so sibling scripts are *absent* and skipped, an assumption a PATH binary violates. The faithful "just repoint it" would have shipped a silent latency + context-noise regression.

**Solution**: Retire the per-session drift block rather than repoint it; surface env-contract drift post-setup (`setup-linux.sh` runs `dotf doctor`) and on demand instead. A focused `dotf doctor --quick` (env-contract only, no harness gate) is tracked for the hook with the SessionStart hook port. Time the tool against the hot path before wiring it in.

**Rule**: A diagnostic that's fine to run by hand can be far too heavy for a per-event hook once it shells out to a generator or probes N tools. Before wiring a "do everything" command into a hot path (session start, pre-commit, prompt-submit), measure it and split a `--quick` subset — "it's the same checks" ignores that frequency, not check count, sets the cost budget.

### [2026-06-14] A non-runnable cobra parent with no subcommands is demoted to "Additional help topics"

**Context**: Building `dotf init` (CLI-014) incrementally — Step 1 wired the `init` parent into `root.go` before its `agents`/`github` subcommands or an orchestrator `RunE` existed.

**Problem**: A cobra command that is neither `Runnable()` (no `Run`/`RunE`) nor `HasSubCommands()` renders under **"Additional help topics"** in `dotf --help`, not **"Available Commands"** — and the default help template suppresses the whole `Usage:`/`Flags:` block (`{{if or .Runnable .HasSubCommands}}`). A user scanning `dotf --help` wouldn't see `init` as a real command. A test that asserted `Usage:` was present caught the demotion.

**Solution**: Give the parent `RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }` — the same idiom `root.go` already uses for the root. That makes it `Runnable()`, so cobra lists it under "Available Commands" and renders the Usage block, while bare `dotf init` just prints help until the real orchestrator action lands. (Adding a subcommand also flips `HasSubCommands()` true, but the parent should be first-class from its first commit.)

**Rule**: When scaffolding a cobra subcommand tree incrementally, an under-construction parent namespace must be `Runnable()` or already have a subcommand to be a first-class "Available Command". Default it to `RunE: cmd.Help` (the repo idiom) rather than leaving it bare — a bare parent silently degrades to a help topic with no Usage block.

### [2026-06-15] A gitignored `//go:embed` asset builds green everywhere and only fails at runtime in a fresh checkout

**Context**: `dotf init` (CLI-014) vendors its scaffold templates under `cli/internal/initrepo/templates/` via `//go:embed`. The CLAUDE.md pointer template was committed as `templates/CLAUDE.md`.

**Problem**: The repo's root `.gitignore` has a `CLAUDE.md` rule, which silently swallowed `templates/CLAUDE.md` — it was never tracked. `//go:embed` reads the **working tree**, not the git index, so it embedded fine on the dev machine and every local `go test`/`go build` passed. Embedding a whole directory never requires a specific file, so even a fresh CI checkout **built a green binary**. The gap surfaced only at *runtime* on the first CI integration run: `dotf init` panicked with `open templates/CLAUDE.md: file does not exist`, because the file the code expected was absent from the clean checkout. No build, unit, or lint step caught it — the failure window was build-green / run-red.

**Solution**: Commit the template under a name no `.gitignore` rule matches (`claude-md`) and map it to its output filename in `scaffold.go` (`{"claude-md", "CLAUDE.md"}`). Add `tests/cli-embed-templates.bats` (incident -> guard) asserting (1) every file under `templates/` is git-tracked via `git ls-files --error-unmatch`, and (2) the `claude-md` mapping exists while `templates/CLAUDE.md` does not — catching the whole bug-class locally before CI.

**Rule**: `//go:embed` (and any embed-from-working-tree mechanism) trusts the working tree, not the index — a gitignored embedded asset is invisible to the compiler's tracking and fails only at runtime in a clean checkout. Never let an embedded asset's filename collide with a `.gitignore` rule, and assert every embed asset is git-tracked. Generalizes: any "works on my machine, missing in CI" symptom is a tracking gap — probe it with `git ls-files`, not another rebuild.

### [2026-06-15] Minting a new script to delete four others fights a reduce-the-surface goal — remove and ticket-restore, don't extract

**Context**: CLI-014 folds `init-project.sh` + the `init-repo-*.sh` helpers into `dotf init`, executing ADR-021's north star: shrink the per-OS shell-script surface. `init-project.sh` carried a vault-only sub-mode (`--work-sdk <family> <component>` -> writes a `50_work/45-development/…` vault entry, scaffolds no repo). The prior session's plan was to **extract** that mode into a standalone transitional `init-work-sdk.sh` so the capability survived the deletion.

**Problem**: `--work-sdk` is vault work, not repo scaffolding, so it has no place in the `dotf init` orchestrator — that part was right. But "extract it to a new `.sh`" mints a brand-new per-OS script at the exact moment the whole change exists to *delete* four of them. Spending a new artifact to preserve an infrequently-used capability (onboarding a work-SDK component) is a poor trade: a transitional shim that adds surface in order to remove surface, fighting the goal it claims to serve.

**Solution**: **Remove** the mode on both OSes (the Linux `.sh` dies with `init-project.sh`; the Windows `init-project.ps1 -WorkSdk` block plus its `-Family`/`-Component` params are stripped) and **ticket its restoration in the right home** — #388 restores it inside `dotf vault` (ADR-021 step 3: cross-platform Go, no `.ps1` twin). Rejected: "extract" (a transitional artifact fighting the north star) and "port into CLI-014" (scope-bleeds the init flagship into vault concerns).

**Rule**: When a consolidation's whole point is to reduce surface, preserving a misplaced capability by extracting it into a *new* unit is usually net-negative — it adds surface to remove surface. Prefer remove-now + ticket-restore-in-the-correct-home over extract-to-a-transitional-shim. Re-weigh inherited "extract" plans against the north star mid-flight: a strangler's job is fewer units, not relocated ones.

### [2026-06-16] A migration that *broadens* a system's scope leaves single-repo assumptions hardcoded in the tools built against the old shape

**Context**: The bitácora began life as a per-repo idea and became, under ADR-018, one cross-repo GitHub Project spanning many repos (kubelab, knowledge, dotfiles, …). `dotf spec init --issue N` was written when "the bitácora" still effectively meant "the dotfiles repo": it ran the work-gate (`gh issue view N`) against the *current* repo's default, and it hardcoded the scaffolded frontmatter prefix to `issue: "dotfiles#N"`.

**Problem**: The scope-broadening migration (one repo -> a multi-repo Project) silently invalidated two assumptions baked into a tool built against the old shape. Scaffolding a kubelab spec gated by `knowledge#104` broke twice: the gate checked issue 104 in the *wrong* repo (kubelab's, where it is a different / closed issue), and the frontmatter recorded the wrong `dotfiles#104`. Neither was visible at build or test time — they manifest only when the gated issue actually lives in a different repo than the one you are scaffolding in, a case the original single-repo world made impossible. The narrow assumption stayed *correct for the common same-repo gate* and failed only at the edges the broadening newly allowed.

**Solution**: Resolve the host repo explicitly — `--bitacora-repo owner/repo` -> `$DOTF_BITACORA_REPO` -> the current repo's `git remote origin` slug — and thread that one resolved slug through BOTH the gate (`gh issue view --repo <slug>`) and the frontmatter (full `owner/repo#N`, never a bare `dotfiles#`). An unresolvable repo errors (pointing at `--bitacora-repo`) rather than fabricating a `#N`; the `[INFO] Work-gate OK` line names `owner/repo#N` so a wrong-repo gate is visible, not silent. Rejected: defaulting to a fixed `mlorentedev/knowledge` — that swaps one hardcode for another and breaks the common same-repo gate. Regression guards in `spec_test.go` + `cmd/spec_test.go` pin all three precedence paths (HARNESS-023, PR #393).

**Rule**: A migration that *broadens* a domain object's scope (single -> multi, local -> distributed, per-repo -> cross-repo) is more dangerous than one that merely moves it, precisely because the old narrow assumption stays valid for the common case and fails only at the edges the broadening newly permits — so tests written in the old world cannot see the gap. When you generalize a system, grep the tools built against its old shape for the now-too-narrow assumption (a hardcoded repo, a single-tenant ID, an implicit "there is only one") and re-derive it from the broadened source. Sibling of the incomplete-migration class (a rename that leaves callers stale): both are "the migration moved the world but not everything that still references it."

### [2026-06-16] Extracting a hook function into a sibling script can flip its exit status and silently kill the hook under `set -e`

**Context**: MEMORY-002 extracted the vault→memory symlink resolver out of `claude-session-start.sh`'s `ensure_memory_symlink` into a standalone agnostic `ensure-memory-symlink.sh`. The hook's function was rewritten to delegate: compute Claude's encoded target, call the script, append any message it printed.

**Problem**: The refactored function ended with `[ -n "$msg" ] && CONTEXT_LINES="…"`. When `$msg` is empty (the common steady state — no new symlink to create), that compound returns 1, so the *function* returns 1. It is invoked bare (`ensure_memory_symlink`) under the hook's `set -euo pipefail`, so a non-zero return **aborts the whole hook before it prints its JSON** — every Claude SessionStart would emit zero `additionalContext`. The original always returned 0 (it ended in an `if … fi`). Two suites disagreed: `session-start-false-positives.bats` copies only the hook, so the sibling is absent and an early `[ -x "$helper" ] || return 0` guard returns 0 — it passed and masked the bug; the `byte-equivalence` test runs from the real `scripts/` with the sibling present and caught zero-output-vs-full-output across 3 CWDs.

**Solution**: End the delegating function with an explicit `return 0` (plus `|| true` on the command substitution and a sibling-presence guard so the hook degrades cleanly when the script is not deployed). The byte-equivalence regression test — diffing the refactored script's stdout against `origin/main`'s across representative CWDs — is the oracle that a behaviour-preserving extraction stays byte-identical.

**Rule**: When extracting logic out of a `set -e` script into a function or sibling, audit the function's *last command's* exit status — a trailing `[ cond ] && action` returns 1 when the condition is false, and a bare call to that function then aborts the parent. Make best-effort helpers end in explicit `return 0`. And trust output-parity (byte-equivalence) tests over fixture tests that copy only a subset of the deploy tree: the subset can hide a newly-introduced sibling dependency that only bites when the full tree is present.

### [2026-06-16] Extract the shared resolution logic, not the whole caller — keep agent-specific detail in the hook

**Context**: MEMORY-002 pulled the vault→memory symlink target resolution out of `claude-session-start.sh` into a standalone `ensure-memory-symlink.sh`, so other agents (or `dotf init`) could reuse the linking mechanics without reimplementing it.

**Problem**: The naive extraction boundary is "move the whole function" — but the original function mixed two concerns: computing Claude's agent-specific encoded project key (`encode_project_path`) and the agnostic vault-source-to-target linking mechanics (resolve, link, idempotent no-op). Extracting the whole thing either drags Claude-specific encoding into a script every other agent must also call, or forces each agent to reimplement the encoding step redundantly.

**Solution**: Split at the seam: `encode_project_path` (agent-specific) stays in the Claude hook; the shared script receives the already-computed target and only does the generic resolve+link+safety-check. A future agent (or `dotf init`) supplies its own encoding scheme and calls the same shared script.

**Rule**: When extracting shared plumbing out of an agent-specific caller, extract only the pure "given X, do Y" mechanics — leave every agent-specific naming/encoding decision in the caller. The caller computes what makes it unique; the shared script does what would be identical no matter which caller invoked it.

### [2026-06-17] A byte-identical parity contract is the tripwire that exposes a template divergence masquerading as a rename

**Context**: CLI-015 PR2 (#395/#403) extracted `dotf init`'s inlined vault-entry renderer into `cli/internal/vault` as `WriteProjectEntry`, moving three `vault-*` templates with it. The inherited plan (from the prior session's handoff) was "Full SSOT + drift": vendor the templates into the vault SSOT and drift-test them, mirroring PR1's work-SDK precedent.

**Problem**: Going to reconcile the embedded `vault-context.md` against the vault SSOT's `project-context.md` — to drift-test one against the other — revealed they were **never the same artifact**. Different token schemes (`{{repo}}`/`{{stack}}` vs `{{project_name}}`/`{{git_url}}`), different structure (the SSOT version carries the HARNESS-006 orientation contract + a `/context-refresh` patchable frontmatter block; the embedded one is an older generation). Same for `vault-memory.md` vs `agent-memory.md`; `vault-roadmap.md` had no SSOT counterpart at all. "Drift-test them" silently assumed a 1:1 source existed. The `#395` byte-identical acceptance criterion was the tripwire: making the embed equal the SSOT would *change* `dotf init`'s output (a behavior change), while vendoring the embed into the SSOT as-is would duplicate `project-context` (an SSOT violation). The divergence was invisible at build/test time — only the parity contract forced the question "are these actually the same file?".

**Solution**: Re-scope PR2 to move the templates **embed-only** (no drift guard), preserving byte-identical output, and ticket the reconciliation (#400) as a separate behavior-changing PR that owns the design decision (which generation wins). Confirmed parity empirically: the `git mv` rename shows 100% similarity, and `dotf vault project` vs `dotf init` emit an identical `00-context.md` modulo the repo name. Rejected: (a) vendor-as-is into the SSOT (duplicates `project-context` → SSOT violation); (b) adopt the SSOT generation now (breaks the byte-identical contract + busts the atomic-PR cap).

**Rule**: A "vendor + drift-test against the SSOT" plan silently assumes the embedded copy and the SSOT are one artifact — a shared *name* is not a shared *lineage*. Diff them before adopting the plan. A byte-identical / behavior-preserving acceptance criterion is the cheapest tripwire: if making the two equal would change output, they were never one source that drifted but two generations that diverged — and reconciling them is a design decision (which generation wins), not a mechanical re-vendor. Defer it to its own PR; never smuggle a behavior change into a move. (Generalizes "verify-before-act on agent audits": re-weigh an inherited plan against the evidence the moment the evidence contradicts its premise.)

### [2026-06-17] Per-repo git hooks can't enforce a machine-wide invariant — core.hooksPath + a chaining dispatcher is the keystone (GUARD-001)

**Context**: Building GUARD-001 so agent memory (`MEMORY.md`, `memory/`, session handoffs) can only ever be committed to the vault, never leak into a code repo — the gap that let `MEMORY.md` reach the ts-bridge repo.

**Problem**: A per-repo `.git/hooks/pre-commit` only protects repos where it was installed; it cannot enforce an invariant across *every* repo on the machine, and a freshly-cloned or newly-`init`ed repo is unprotected by default. `pre-commit install` is per-repo too.

**Solution**: One global `git config --global core.hooksPath <dir>` points every repo at a single tracked dispatcher dir. Because a global `hooksPath` makes git ignore each repo's `.git/hooks/`, the dispatcher runs its global concern (the memory-sink guard) and then `exec`s the literal `.git/hooks/<type>` — the chaining is what keeps per-repo hooks (gitleaks, the pre-commit framework) alive. `dotf init` also bakes a `MEMORY.md`/`memory/` block into a new repo's `.gitignore` so it is born convention-correct. Wiring is **safety-first**: a global setting has machine-wide blast radius, so an unrelated pre-existing `core.hooksPath` is never clobbered — `dotf doctor` WARNs and preserves it, wiring only when unset and only under `--fix`.

**Rule**: To enforce an invariant across *all* repos, reach for global `core.hooksPath` + a chaining dispatcher, never a per-repo hook. When a tool writes a global/shared setting, treat any pre-existing value as sacred: detect-and-warn, wire only when absent, and gate the write behind `--fix` — never silently overwrite something with machine-wide reach.

### [2026-06-17] A Windows CI tool added only to $GITHUB_PATH vanishes when setup-windows rebuilds PATH from the registry

**Context**: BUG-025 (#425) replaced `choco install age.portable` in the `test-windows` job with a deterministic download of the pinned `age` release, wired onto PATH via `$GITHUB_PATH`, to kill the chocolatey-shim PATH-propagation flake (sibling of the `eza`/`zoxide` flake BUG-024).

**Problem**: `test-windows` went red on one deterministic failure — `pi`'s deployed `models.json` kept an unresolved `{env:NAN_API_KEY}` placeholder. The secrets sandbox encrypts a throwaway `nan.api-key.secret.age` and `setup-windows.ps1` decrypts it through `Substitute-EnvPlaceholders` → `& age --decrypt`. With choco, `age` lived on the **Machine registry PATH** and was always resolvable; the release-zip approach put it *only* on `$GITHUB_PATH` (a process-level PATH injection the runner applies to subsequent steps). But `setup-windows.ps1` rebuilds `$env:PATH` from the Machine+User registry mid-run (so freshly-installed tools are usable immediately), and that rebuild **discards** the process-only `$GITHUB_PATH` entry. So `& age` silently failed — its stderr was swallowed with `2>$null` — the secret never decrypted, and the placeholder survived into the deployed config. Two things hid the cause: the swallowed stderr masked the real error, and `test-windows` runs only on `pull_request` events, so every `push` to main shows it `skipped` — "main is green" was a false signal (the prior green came from PRs #413/#419/#421, which still used choco).

**Solution**: Persist `age` on the **User registry PATH** as well (`[Environment]::SetEnvironmentVariable('PATH', "$userPath;$ageDir", 'User')`) so it survives `setup-windows`'s registry-PATH rebuild — restoring the exact property the old choco Machine-PATH install had. The `eza`/`zoxide` failures in the same job were a *genuine* winget→PATH flake (BUG-024), proven by re-running the job (they passed on retry); the NAN failure reproduced across every re-run, which is what separated the real regression from the flake.

**Rule**: On Windows CI, any tool the *deploy script itself* must resolve at runtime has to be on the **registry PATH (Machine/User scope)**, not just `$GITHUB_PATH` — a script that rebuilds `$env:PATH` from the registry (a common pattern after installing tools) discards process-only additions. Never swallow a binary's stderr (`2>$null`) on a code path that can fail silently: a hidden `age --decrypt` error turned a one-line PATH bug into a cryptic downstream placeholder. And re-run a red job before declaring a regression — a failure that reproduces is a bug, one that clears is a flake; here a single job carried both at once.

### [2026-06-18] A WARN that doesn't move the exit code is invisible to CI — give the CI surface its own probe, don't shell out to the tool

**Context**: OPS-009 added a PAT-expiry preflight (`dotf doctor`'s "PAT expiry" section) after a classic PAT expired silently and broke release-please's first run with `Bad credentials`. The instinct for the second surface — a scheduled Action that warns *before* CI goes red — was to reuse the binary: run `dotf doctor` in the workflow and act on its exit code.

**Problem**: `Report.ExitCode()` is non-zero only on `StatusFail`; `WARN`/`SKIP`/`INFO` are advisory and never move it (the healthcheck/doctor exit contract). The whole point of the Action is to fire on "expiring **soon**" — which the check classifies as **WARN**. So shelling out to `dotf doctor` and reading `$?` would see exit 0 for exactly the state the Action exists to catch; it could only ever detect an already-dead token (FAIL), i.e. the outage it was meant to prevent. Coupling the CI alert to the binary's exit status would have silently defeated the feature.

**Solution**: The Action does its **own** focused probe (curl `GET /user`, read the `github-authentication-token-expiration` header, compute days-left in ~15 lines of shell) rather than invoking the Go binary. The duplication is deliberate and documented (proposal R5/R7): the two surfaces run in different contexts (local shell env vs Actions secrets) and answer different questions (full diagnostic exit code vs a single "is this one token expiring?" boolean). The local `dotf doctor` surface still shows the WARN to a human who runs it; the Action owns the machine-actionable alert.

**Rule**: Before reusing a diagnostic tool's exit code as a machine signal, check *which* severities actually move that exit code. If the state you need to act on is advisory (WARN/INFO) rather than failing, the exit code can't see it — give the consuming surface its own narrow probe instead of shelling out. A tool whose exit code means "something is hard-broken" cannot also mean "something will break soon"; don't conflate the two axes.

### [2026-06-18] An npm-global CLI under nvm is invisible to GUI/ADE processes and to any shell on a different node version — install agent CLIs into ~/.local

**Context**: Using Orca (the parallel-agent ADE), `pi` launched fine from the default terminal but `command not found` from inside Orca — only `claude` would start. `setup-linux.sh` installs `pi` with a bare `npm install -g`.

**Problem**: A bare `npm i -g` under nvm lands the binary in nvm's **per-node-version** tree (`~/.nvm/versions/node/<v>/bin/pi`). The terminal resolves node via `nvm use default` (LTS, where pi was installed); Orca — a desktop AppImage that never sources the interactive shell — propagates a PATH carrying a *different* node version (v26 vs the v24 LTS), so `pi` is absent there. Same machine, same user, different node on PATH → the CLI exists but is unreachable. `claude` worked because it lives in `~/.local/bin` as a standalone binary, independent of any node version. The doctor compounded it: when pi was configured (`~/.pi/` present) but off PATH, `checkOpenCode` reported the misleading SKIP "pi not installed" instead of failing on the real cause.

**Solution**: Install pi into the manager-independent `~/.local` prefix — `npm install -g --ignore-scripts --prefix "$HOME/.local"` puts the launcher at `~/.local/bin/pi`, the same dir `claude`/`dotf` use, which is on PATH for login shells AND GUI/ADE processes. Its `#!/usr/bin/env node` shebang then runs under whatever node each environment provides. Guard the install on `~/.local/bin/pi` (the stable location), not bare `command -v pi`, so a stale nvm-version copy can't mask a missing launcher. Add the incident→guard branch to `dotf doctor`: configured-but-off-PATH → FAIL with the root cause, not a SKIP.

**Rule**: Any CLI a GUI app / ADE (or a cron job, or a different-node shell) must spawn has to live on a version-manager-independent PATH dir like `~/.local/bin` — never only in nvm/asdf/volta's per-version global, which is invisible outside the shell that activated that version. When a tool is "found in my terminal but not in <other launcher>", first compare the *active node/runtime version* in each environment before assuming it's uninstalled. And a health check must distinguish "absent" from "present-but-unreachable" — the second is the actionable failure, not a skip.

### [2026-06-18] An env-var seam is inert until something sets it — a hardcoded fallback that matches reality hides the broken seam

**Context**: The vault and dotfiles repo were relocated from `~/Projects/` to `~/Projects/Workspace/`. Vault MCP, hive, selfupdate and the session hooks all broke silently.

**Problem**: The code already had the seams (`VAULT_PATH`, `DOTFILES_REPO_DIR`, `HIVE_VAULT_PATH`) and a Go resolver that honored them — but nothing on the deploy path ever *set* them: the shell profiles hardcoded the value, `VAULT_PATH` was never exported, and the session hooks read a literal `~/Projects/knowledge` instead of the seam. It "worked" only because each hardcoded fallback coincided with the real path. The seam was decorative. The day the assumption (path == default) shifted, every consumer broke at once, and silently — the resolver degraded to "not found" instead of erroring. The same concept even carried three names across the code (`VAULT_PATH` / `VAULT_DIR` / the literal).

**Solution**: ADR-025 — a real cascade (`env → ~/.config/dotfiles/machine.json → env-contract default[OS]`) rendered into `paths.{sh,ps1}` that shells source and that `dotf env generate` writes for every consumer; `dotf doctor` asserts drift. Collapse the three names onto `VAULT_PATH`.

**Rule**: A seam (env var, config key, DI point) is only real if something on the deploy path *sets* it AND every consumer reads it through one resolver. A fallback equal to today's reality is a latent bug, not a safety net — it postpones the failure to the first time the assumption moves and makes it silent. Audit seams by grepping who *sets* them, not just who reads them; collapse synonyms to one name.

### [2026-06-18] "Wire all consumers" must enumerate the non-shell ones — services and daemons never source a shell profile

**Context**: After ADR-025 wired the vault-path cascade into shells, hooks and the Go CLI, the Claude Code Hive still pointed at the old path.

**Problem**: `paths.{sh,ps1}` only reach interactive shells. The hive `serve` daemon (systemd `--user` on Linux, Scheduled Task on Windows) and the agy MCP registration never source a shell profile, so a shell-only mechanism silently missed them. `hive client` is a stateless proxy; the daemon reads `HIVE_VAULT_PATH` from its OWN process env at start — and `claude mcp add` has no `--env`, so the path can't flow through the MCP registration either.

**Solution**: Provision the *service* environment directly — `~/.config/environment.d/10-hive-vault.conf` (read by the systemd `--user` manager for all user services) on Linux, and a User-scope env var (Scheduled Tasks inherit it) on Windows — both from the same cascade via a new `dotf env path <KEY>`.

**Rule**: When a change claims to "wire all consumers" of a value, list them by *execution context*, not convenience: interactive shells, login shells, services (systemd/launchd/Scheduled Task), cron, MCP daemons, GUI/ADE processes. Each has its own env-provisioning mechanism; a shell-profile mechanism reaches none of the non-shell ones. A proxy daemon (client→server) holds its state in the *server* — set the server's env, not the client's.

### [2026-06-18] Number an ADR off the latest origin/main, not your branch base — a stale base collides with ADRs shipped in parallel

**Context**: I wrote ADR-023 for cross-machine path resolution, taking the next number from my branch base (highest was ADR-022). While I worked, `main` advanced: ADR-023 (agnostic session-start) and ADR-024 (PAT-expiry) shipped. On merge, my ADR-023 collided.

**Problem**: ADR numbers are a shared append-only namespace, but I picked the next one from a *stale local base* instead of current `main`. A parallel session had already claimed 023 and 024. The collision surfaced only at integration; renumbering to 025 then meant chasing ~20 files + 3 GitHub issue/PR bodies — and surgically *not* touching the other ADR-023's references in files that carried both (the session hooks).

**Solution**: Renumber to ADR-025 (next free on merged main), case-sensitive + slug-aware; in files referencing both ADR-023s, edit only the cross-machine line.

**Rule**: Before assigning any number in a shared append-only namespace (ADRs, migrations, ticket IDs), resolve it against the *latest* `origin/main`, not your branch base — `git fetch && git ls-tree origin/main docs/adr/`. When parallel work is likely, reserve the number up front rather than guessing.

### [2026-06-18] `gh issue/pr create` use GraphQL — when that bucket is rate-limited, `gh api -X POST` (REST) still works

**Context**: Mid-session, `gh repo view --json` and `gh label list` failed with "GraphQL: API rate limit already exceeded", blocking issue/PR creation.

**Problem**: GitHub keeps *separate* rate-limit buckets for REST (5000/h) and GraphQL (5000/h). `gh issue create`, `gh pr create`, `gh label list` and `gh project` go through GraphQL; when that bucket is exhausted they all fail — while the REST bucket can be completely fresh.

**Solution**: Create the issue/PR via REST — `gh api -X POST /repos/<owner>/<repo>/issues --input -` (and `/pulls`), feeding a JSON body; check both buckets with `gh api /rate_limit --jq '.resources'`. PowerShell gotchas: capture a fetched body as a single string (`-join "\n"`, since the tool splits multiline output into an array) and use case-sensitive `-creplace` — plain `-replace` is case-insensitive and corrupts e.g. `adr-023` → `ADR-025`; build the payload with a single-quoted here-string `@'...'@` so `$`-vars stay literal.

**Rule**: On a `gh` "GraphQL rate limit" error, don't wait — fall back to `gh api` REST endpoints (separate bucket), verified via `gh api /rate_limit`. For scripted body edits: single-string + case-sensitive replace + literal here-string.

### [2026-06-18] A release binary goreleaser already builds is worthless until each OS's setup script actually downloads it

**Context**: WIN-006 wired Windows setup to fetch a prebuilt `dotf` release binary instead of requiring a local Go toolchain.

**Problem**: `dotf`'s cross-platform binaries were already produced by `goreleaser` on every tagged release — but `setup-windows.ps1` had no step that downloaded them. That made "dotf doesn't work on a fresh Windows box" read as a CLI-porting problem ("needs Go"), when the actual gap was one missing fetch-and-verify step in setup.

**Solution**: Add `install-dotf.ps1` (a PowerShell mirror of the existing `install-dotf.sh`), dot-sourced by `setup-windows.ps1` non-fatally before anything needs `dotf`.

**Rule**: Before treating "doesn't work on OS X" as a build/porting problem, check whether the artifact already exists and the gap is purely a missing fetch step in that OS's setup path. Producing a release artifact and consuming it in every deploy path are two separable deliverables — verify both exist before assuming a bigger rewrite is needed.

### [2026-06-19] Orca regenerates its Copilot hooks — re-apply the fix idempotently and guard the drift (DX-006)

**Context**: On Windows, Copilot CLI tool calls were intermittently denied with "Denied by preToolUse hook ... (hook errored)". The hook is registered in `~/.copilot/hooks/orca.json` and runs `~/.orca/agent-hooks/copilot-hook.ps1`.

**Problem**: Two compounding causes. (1) `orca.json` ships `"timeoutSec": 5`. (2) `copilot-hook.ps1` POSTs the event with `Invoke-WebRequest`, whose cold init on Windows PowerShell 5.1 is ~4.5s; added to the cold `powershell.exe` spawn (~1.5s) the hook exceeds 5s under load, the CLI kills it, and the tool call is denied. Worse: both files are **generated by Orca**, so any manual fix is silently reverted on the next Orca install/upgrade (or a dotfiles redeploy that re-triggers Orca generation). A prior hand-fix had already been clobbered exactly this way.

**Solution**: `scripts/orca-hook-tune.ps1` re-applies the fix idempotently and reversibly — bump every `hooks.*.timeoutSec` below 30 up to 30, and swap the `Invoke-WebRequest` POST for a `System.Net.HttpWebRequest` call (~1.5s, no IE-engine init), with a timestamped `.bak` and atomic write. `setup-windows.ps1` deploys + runs it (best-effort; skips when Orca is absent); `healthcheck.ps1` section 13/13 is a narrow drift guard (FAIL if `timeoutSec` < 30 or `Invoke-WebRequest` is back). Measured: hook latency dropped from ~6s worst-case to ~1.5s.

**Rule**: When a tool you don't own regenerates its config/hooks on every install, treat the fix as a re-appliable **idempotent patch** (not a one-shot edit), wire it into setup, and add a health-check guard so the inevitable drift is **loud, not silent**. Fix BOTH the failure cause (the timeout) and the latency cause (the transport): a generous timeout stops the errors, the fast transport stops the per-call tax. On Windows PowerShell 5.1, avoid `Invoke-WebRequest` in hot/spawned paths — `HttpWebRequest` avoids its multi-second cold init.

### [2026-06-21] Strangler-fig deletion: the parity gate must cover OS-specific side effects, and a "different-by-design" Go path can still be parity (CLI-020)

**Context**: First real `.ps1` deletion of the ADR-020/021 CLI convergence — repoint Windows `project-init` to `dotf init` and delete the 3 init `.ps1`. The spec gated the deletion on proving `dotf init` is at parity on Windows.

**Problem**: A naive check ("does it scaffold a repo?") would have missed two things. (1) The `.ps1` resolved `VAULT_PATH` via a hardcoded `~/Projects/knowledge` fallback; `dotf init` resolves through the ADR-025 seam — *different code, must verify it resolves on Windows*. (2) The `.ps1` eagerly created the Windows memory **junction** at init time; `dotf init`'s Go `linkMemory` is **non-Windows by design** and creates nothing on Windows — which looks like a regression.

**Solution**: Verified empirically with an isolated `VAULT_PATH=$tmp dotf init <tmp> --skip-github` (throwaway vault, zero pollution): full scaffold + vault entry produced, `VAULT_PATH` honored. The junction "gap" is not one — `claude-session-start.ps1` `Ensure-MemoryJunction` recreates it every session, and a junction's only consumer *is* a Claude session, so the transient window is harmless. Outcome: pure repoint+delete, no Go change. Left `agents-spec-section.md`'s stale ref untouched (vault-SSOT, drift-tested → #461).

**Rule**: For every strangler-fig deletion, enumerate **all** behaviors of the dying twin — not just the happy path: input resolution, seeded artifacts, and **OS-specific side effects** (symlinks/junctions, registry, PATH). A Go replacement that *deliberately* omits an effect is still at parity **iff** a downstream consumer reconstructs it — find and name that consumer, don't assume. Verify in an isolated sandbox (throwaway `VAULT_PATH`/dirs), never against the live vault.

### [2026-06-20] Windows winget jq emits CRLF — breaks `< <(jq)` + read

**Context**: `compile-harness.sh --refresh` (the harness deploy engine) aborted on Windows with `section "6-attribution-policy" not found`, even though the slug matched a real heading. The deployed agent skills had silently drifted from the vault SSOT for ~16 skills.

**Problem**: The winget build of jq (1.8.1) emits CRLF on every output line. MSYS/Git-Bash command-substitution `$(jq …)` strips the trailing CRLF, so single-value reads look clean — but `read`/`mapfile`/`for` fed via `< <(jq …)` keep the bare `\r` in the last field, so `slug == want` comparisons (and any path built from jq output) silently fail. Because Windows `setup` only runs the deploy half (refresh is Linux-only), nothing caught the resulting vault→records drift.

**Solution**: Shadow `jq` with a CR-stripping wrapper that preserves the real exit status (`return ${PIPESTATUS[0]}`, so `jq -e` truthiness still works); verify the binary with `type -P jq`, not the function. (PR #511; to be superseded by the Go port CLI-026.)

**Rule**: Shell engines that read tool output into loops via process substitution must strip `\r` — the `$(...)` CRLF-strip is a Git-Bash illusion that does NOT extend to `< <(...)`+read. And: a `vault → records → deploy` pipeline has two drift axes — cover BOTH (records↔deploy = CLI-019/#488; vault↔records = CLI-026 `check --against-vault`), or one half drifts silently.

### [2026-06-21] Catalog installer: release naming is per-repo data, not a convention (CLI-029)

**Context**: `dotf tools install` (the declarative `packages.json` catalog's installer) reuses the `install-dotf` download→checksum→place pattern, generalised from one CLI to any github-release tool. First tool: sops.

**Problem**: Two assumptions baked into `install-dotf` do NOT generalise. (1) **Archive shape**: `install-dotf` extracts `dotf` from a `.tar.gz`/`.zip`; sops ships **raw binaries** (`sops-v3.13.1.linux.amd64` is the executable itself), so an extraction step would fail. (2) **Checksum filename**: `dotf` ships `checksums.txt`; sops ships `sops-v3.13.1.checksums.txt`. A hardcoded name (or a single asset template) silently 404s or mis-resolves. Both only surface against the *live* release — a unit test over a fixture happily passes a wrong assumption.

**Solution**: Treat the irregularities as **catalog data**: per-OS `asset` map (already in PR-A) plus a `Source.Checksums` template, both expanded from `packages.json`. Drop the extraction step (place the raw binary, rename to the command name, chmod). Reconcile is **pin-as-minimum** (`decideAction`: install/upgrade/skip, never downgrade — REFACTOR-011/013). Verified the real chain with one live `gh release view` + an end-to-end smoke (`dotf tools install` → `sops --version` → idempotent skip), not just the hermetic `Fetcher`-seam tests.

**Rule**: Before wiring a downloader for a new release source, verify the **exact** asset names, archive-vs-raw shape, and checksum-manifest filename against the **live** release (`gh release view <tag> --repo <r>`) — release naming is per-project data, never a safe convention. Keep those facts in the catalog (templates), not in installer code, so the next tool (CLI-028) is a data edit, not a code change. Hermetic seam tests prove the *logic*; only a live smoke proves the *facts*.

### [2026-06-21] release-please can close a multi-PR issue from a build-only sub-PR's `Refs` — keep the parent issue out of sub-PR footers

**Context**: CLI-018 was split into PR-B0 (§4 coverage port, build-only, #522) and PR-B (the deletion, tracked by #509). #522's footer deliberately said `Refs #509`, *not* `Closes`, because the deletion was not done.

**Problem**: When #522 merged, release-please rolled it into the 0.13.0 release PR (#523), whose generated changelog rendered the reference as `closes #509`. Merging the release PR auto-closed #509 — the work-gate for the still-unstarted deletion vanished while `healthcheck.ps1`/`doctor.ps1` were verifiably still in `main`. Same premature-close class as #488 earlier in the convergence.

**Solution**: Verified the deletion was genuinely undone (both `.ps1` present), reopened #509 with the remaining scope and a note on the cause.

**Rule**: A build-only sub-PR of a multi-PR issue must not reference the parent issue in its footer *at all* — release-please aggregates any issue mention into the release's `closes` list regardless of the `Refs` vs `Closes` keyword. Reference the sub-task or nothing; reserve the issue reference for the final PR that completes it. After any release that swept a multi-PR issue, re-check that issue is still open.

### [2026-06-21] `git branch --merged` answers "is the tip an ancestor?", not "is the content backed up" — verify before deleting

**Context**: Housekeeping a 5-week-old orphan branch (`fix/win-sessionstart-hook-path`). `git branch --merged origin/main` would have green-lit deleting it — its single commit does not conflict with `main`.

**Problem**: "Doesn't conflict" ≠ "redundant". The commit only *adds* files — 7 spec scaffolds — and 3 of them (AI-012, AI-013, WIN-003) exist nowhere in `main` (neither active `specs/` nor `specs/archive/`). `--merged` merely tests whether the branch tip is reachable from `main`, which is trivially true for a branch whose unique commit adds new files. Deleting on that signal silently loses unmerged work.

**Solution**: Per-artifact verification — for each path the branch adds, confirm it is implemented (in `main`), ticketed (an issue), or specified (a spec dir, active or archived). Three scaffolds failed all three → kept the branch.

**Rule**: "Safe to delete" means "all its content is backed up elsewhere", not "git says merged". Diff the branch's added content against `main` (`git log origin/main..branch`, then confirm each added path exists in main/archive/an issue) — never trust `--merged` for a delete decision.

### [2026-06-21] Resolving Windows `$PROFILE` from Go must include the OneDrive-redirected Documents root

**Context**: Porting healthcheck §4 (`$PROFILE` existence) into `dotf doctor`. Go has no `$PROFILE` intrinsic, so the check enumerates candidate paths.

**Problem**: A naive `~\Documents\{PowerShell,WindowsPowerShell}\Microsoft.PowerShell_profile.ps1` check false-FAILs on corporate Windows, where Documents is frequently redirected to `~\OneDrive\Documents` by Known Folder Move. The profile is present; the check reports "missing".

**Solution**: Enumerate `{Documents, OneDrive\Documents} × {PowerShell (pwsh 7), WindowsPowerShell (5.1)}` and PASS on any hit. Chosen over shelling out to `pwsh -Command '$PROFILE'` — pure-Go + deterministic fits the doctor's temp-tree test model and avoids a SKIP-when-pwsh-absent branch.

**Rule**: Any Go (or non-PowerShell) check that reconstructs a Windows user-profile path must account for OneDrive Known Folder redirection of Documents — `%USERPROFILE%\Documents` alone is wrong on managed/corporate boxes. Enumerate both roots, or ask PowerShell for the real path.

### [2026-06-21] In bats, a `! grep -q` guard is exempt from errexit — it won't fail the test when the pattern is found

**Context**: A guard test asserting a retired token (`diff-check`) no longer appears in the production caller files.

**Problem**: Written as `! grep -qF 'diff-check' "$file"`, the line does NOT fail the test when the token IS present. POSIX `set -e` (which bats applies inside a test body) explicitly ignores the failure of a command/pipeline negated with `!`, so the negated grep's status never trips errexit and the test passes regardless — a guard that silently never guards.

**Solution**: Write the negative assertion explicitly: `if grep -qF 'diff-check' "$file"; then echo "still referenced in $file" >&2; return 1; fi`.

**Rule**: Never rely on a bare `! cmd` line as a failing assertion under errexit (bats, or any `set -e` script) — `!`-negated commands are exempt from errexit. Use an explicit `if cmd; then return 1; fi`, or `run cmd` + an `$status` check.

### [2026-06-21] A strict cross-OS `dotf doctor` is not a drop-in CI gate for a lenient platform-specific healthcheck

**Context**: CLI-018 retired Windows `healthcheck.ps1`. The `test-windows` CI job's "Run healthcheck.ps1" step was repointed to `dotf doctor` followed by `exit $LASTEXITCODE`.

**Problem**: The step false-red'd. `dotf doctor` (the cross-OS Go diagnostic) FAILs on a partial CI runner — `HOME` unset, `wget`/`terraform`/`java` are env-contract *required* binaries, and opencode/pi/git-hooks-dispatcher are absent — so it exit-codes 1. The retired `healthcheck.ps1` had been Windows-aware and lenient about exactly those checks, so it exited 0 in the same environment. Renaming the step silently swapped a lenient checker for a strict one against an environment that never satisfied the strict one — same intent, different exit semantics.

**Solution**: Removed the live gate from `test-windows` rather than tuning the Go diagnostic's Windows-awareness (a behaviour change, out of scope for a delete+repoint PR). This mirrors Linux, which runs **no** live diagnostic gate — `dotf doctor` is covered by `go test` + structural bats, and `setup-windows.ps1` still runs end-to-end (its own *non-fatal* post-setup `dotf doctor` prints health without gating).

**Rule**: When a CI gate's underlying tool changes from a lenient platform-specific script to a strict cross-OS one, re-derive what the gate can actually assert in the CI environment — a partial runner will not satisfy a full-install diagnostic, so gating on its exit code false-reds every PR. Check what the *other* OS's CI does (parity) before inventing a gate; a diagnostic built for humans post-setup is normally validated in CI by unit + structural tests, not by gating on its live exit code.

### [2026-06-21] A delete ripples past the direct caller — token guard-greps miss transitive refs, and "orphaned" fixtures can have hidden consumers

**Context**: Deleting `healthcheck.ps1`/`doctor.ps1` + `tests/healthcheck-ps1.bats` (CLI-018 PR-B). A guard test greps the production files for the `(healthcheck|doctor)\.ps1` token.

**Problem**: Two blind spots the guard did not cover. (1) The grep matches the token `healthcheck.ps1` but NOT `tests/healthcheck-ps1.bats` (hyphen, different extension) — so a stale invocation of the deleted bats file survived in `ci.yml`'s bats subset and would have errored the step once the gating step ahead of it was removed. (2) A CI fixture commented "minimal vault tree for healthcheck section 7" looked orphaned by the deletion, but `setup-windows.ps1` itself consumes it: the auto-memory junction deploy is gated on `Test-Path $VaultRoot`, and a stub `obsidian.cmd` on PATH makes setup take its skip-install branch. Deleting on the comment's word would have silently cut end-to-end coverage.

**Solution**: Read the whole CI job, not just the grep hits. Removed the stale `healthcheck-ps1.bats` entry and the genuinely-orphaned eza/zoxide flake-guard step (it only fed the removed diagnostic), but KEPT the vault/obsidian fixtures and corrected their stale comments to name the real consumer (`setup-windows.ps1`).

**Rule**: A token guard-grep covers layer 1 (direct references in the exact filename form). It does not catch transitive references (CI test lists, glob runners) or setup steps that only fed the deleted thing. Before deleting "orphaned" setup, grep its consumers by *capability* (the dir/binary/PATH entry it provides), not just by the comment — a comment documents one original reason, not every later consumer.

### [2026-06-23] A thin per-OS shim is still a twin — converge to direct CLI invocation

**Context**: CLI-025 PR1 ported the SessionEnd hook (`session-handoff.{sh,ps1}`) to the Go `dotf mem session-end` noun. The spec's wording said the hook should become a "thin shim that `exec dotf mem …`".

**Problem**: A thin shim still ships a per-OS `.sh`/`.ps1` pair. It *miniaturizes* the cross-OS twin-drift the CLI convergence exists to eliminate rather than removing it — the disease is not the script's size, it's that it exists in plural per OS. Keeping a shim re-introduces, in the last mile, the exact maintenance burden the `dotf` binary was built to kill.

**Solution**: Wire the hook to invoke the binary directly — the Claude Code hook `command` is the single string `<abs dotf path> mem session-end`, identical on Windows and Unix because `dotf`/`dotf.exe` resolves the same subcommand. Delete both twins outright (no replacement). The one residual OS-variance — "is `dotf` on PATH / where the binary lives" — moves to the single layer that already owns it (env-contract + `dotf doctor`, ADR-025). Use the **absolute** binary path (not bare `dotf`) so the hook survives a broken profile PATH (#531).

**Rule**: When converging a shell-twin cluster to a `dotf` noun, delete the twins outright via direct binary invocation; never replace them with thin per-OS shims. Move the only residual OS-variance to the env-contract layer, never into a fallback inside the scripts you are deleting. If a spec says "thin shim", treat that wording as refinable — it predates the convergence clarity.

### [2026-06-23] A ~120-LOC change is over the SDD bar even when it "obviously" mirrors an existing check

**Context**: OPS-016 added a `checkVaultHooks` diagnostic to `dotf doctor` (#553) — a near-copy of the existing `checkGuardHooks`. Because the GitHub issue read like a complete mini-spec and the change mirrored an established pattern, it was built directly with no `specs/<id>/` folder.

**Problem**: `spec-gate` CI failed — 297 LOC of production diff (≥ the 50-LOC threshold) with no active spec folder touched. "The issue is the spec" is not a shortcut this repo honors: SDD Tier 4 is an automated gate, not a judgment call. Blind spot inside the blind spot: the gate counts `_test.go` files (they are not under `tests/`), so the test file inflated the diff — but even the 119 non-test LOC were over threshold.

**Solution**: Authored the spec retroactively via `dotf spec init OPS-016-… --issue 549` (work-gated on the open issue, ADR-018), filled proposal/tasks/verification + features.json with the real acceptance criteria and evidence, and archived it via `dotf spec archive` once merged. Gate green on the next push.

**Rule**: For any change ≳50 LOC of production diff, scaffold `specs/<id>/` FIRST — never lean on "the issue already explains it" or "this just mirrors an existing check". The gate enforces it mechanically and counts `_test.go` toward the threshold. Corollary on placement: when the change is *pure behaviour* (no repo asset to deploy), its home is a `dotf doctor` check, not a new bootstrap script (ADR-020 C7) — provisioning that just runs `pre-commit install` is behaviour, so it converges into the CLI checker, not shell.

---

### [2026-06-24] A Go-vs-shell byte-equivalence gate is POSIX-only, and it retires at cutover

**Context**: CLI-025 ported the session-start hooks (`session-brief.sh`, `claude-session-start.sh`) to `dotf mem session-start`. Each port shipped a "golden" test that diffs the Go output against the live shell script across representative CWDs.

**Problem**: Two Windows-only divergences make a Go-vs-shell diff impossible to pass there even when the logic is byte-identical. (1) `jq` — the shell hook's JSON encoder — emits **CRLF** on a Windows build, while Go's `encoding/json` emits LF, so every line "differs". (2) The shell runs under Git Bash with MSYS `/tmp/...` paths, but the native Go binary on Windows cannot resolve `/tmp`, so any emitted absolute path (a vault headline, a "not found" line) renders differently (`/c/...` vs `C:\...`). The diff is therefore meaningful **only on Linux**, where both sides use LF and native `/tmp`.

**Solution**: Guard such gates with `if runtime.GOOS == "windows" { t.Skip(...) }` and let them run on the Linux CI job — the POSIX shell is the only equivalence target anyway. And **delete the gate at cutover**: once the shell script is `git rm`'d the gate has no referent to diff against. A byte-equivalence gate proves *fidelity during migration*; the Go unit tests are the *ongoing* regression net. A forever-skipped gate is dead weight — retire it with the shell it compared to.

---

### [2026-06-24] CI golangci-lint enforces staticcheck QF* quickfixes a stale local version skips — heed the gopls hints

**Context**: Two PRs in the CLI-025 chain passed locally (`golangci-lint run` exit 0) but failed the CI `lint` job: an `errcheck` on an unchecked `fmt.Fprint`, and `QF1002` ("could use tagged switch") on a `switch { case x == "": … }`.

**Problem**: The CI action pins golangci-lint **v2.12.2**, which runs the staticcheck `QF*` (quickfix) category; the older binary on the dev machine did not flag them. The editor's `gopls` analyzer DID surface `QF1002` as a hint — but a hint reads as style noise, so it shipped and only CI caught it.

**Solution**: Treat gopls `QF*`/style hints as CI-enforced, not advisory — clear them before pushing. When practical, match CI's golangci-lint version locally; otherwise the cheap habit is: when the editor underlines a staticcheck quickfix, apply it rather than dismiss it.

---

### [2026-06-25] Three Windows path gotchas behind a "broken" auto-memory junction (Go 1.26)

**Context**: HARNESS-040 (#551) wired `dotf doctor --fix` to the merged `memlink` primitive to detect+repair the Claude auto-memory↔vault junction. Implementing it on Windows surfaced three non-obvious cross-OS facts the POSIX-first shell code had silently papered over.

**Problem**: (1) **Encoding** — Claude's per-project key under `~/.claude/projects/<key>` maps *every* path separator AND the drive colon to `-` (`C:\Users\me\proj` → `C--Users-me-proj`), but the ported `encodeProjectPath` only replaced `/`. On Windows it computed the wrong key, so the junction was created at (or looked for at) the wrong path — the latent root cause of the "junction never created here". (2) **Link detection** — a `mklink /J` junction surfaces via `os.Lstat` as `ModeIrregular`, **not** `ModeSymlink`, on Go 1.26 (verified empirically). The old `isLink` checked only `ModeSymlink`, so it never recognized a junction; `Ensure`'s "already linked" no-op only worked by accidentally falling through to its `dirNotEmpty` branch. (3) **cmd quoting** — `exec.Command("cmd","/c","mklink","/J",target,src)` relies on Go's `EscapeArg`, which quotes args with **spaces** (so `C:\Users\First Last\...` works) but not a bare **comma**; cmd then splits the path on the comma and mklink fails silently. A comma slipped in via a `t.TempDir()` path derived from a subtest name containing `(PASS, no dup)`.

**Solution**: Put the encoding in the shared `memlink` primitive (`ClaudeProjectKey` maps `/ \ :` → `-`; `ClaudeMemoryTarget` joins the full path) so the session-start adapter and doctor compute an identical target on every OS, and deleted the local `mem.encodeProjectPath`. Widened `isLink` to `ModeSymlink|ModeIrregular`. Named test subtests without `,`/`()` so they don't poison `t.TempDir()` paths; ticketed the real cmd-quoting robustness fix (#575) rather than rabbit-holing on `cmd /s /c` quoting in this PR.

**Rule**: When code that creates/inspects filesystem links is "ported from shell", re-derive the Windows facts from scratch — separators, the drive colon, junction-vs-symlink mode bits, and cmd argument quoting are all places POSIX intuition is wrong. Keep one OS-aware encoding as SSOT shared by every caller; never let two callers re-encode a path independently. And keep test names free of shell/cmd metacharacters — `t.TempDir()` embeds the test name, so a comma or paren in a subtest name becomes a real path component.

### [2026-06-25] PR title is the release contract under squash + release-please

**Context**: Shipping ADR-028 secrets work as squash-merged PRs; release-please (release-type: simple) cuts releases from conventional-commit subjects.

**Problem**: #584 (registry + `dotf secrets ls/show`) squash-merged with a non-conventional PR title ("Secrets registry: ..."). Squash promotes the PR title to the merge-commit subject, which release-please parses. It logged `unexpected token ' ' at 1:8` and counted 0 releasable commits -> no 0.19.0 release ever opened, even though a user-facing feat had landed. The feature sat on main, unreleased and undeployed, silently.

**Solution**: Landed the next `feat(secrets):` PR to re-trigger release-please (it swept #584 into the 0.19.0 tag). Opened #589 to add a conventional-commit PR-title gate.

**Rule**: With squash-merge the PR TITLE is the release-parsed subject -- it must be a valid Conventional Commit. A non-conventional title doesn't error; it silently drops the change from versioning. When a feature merged but no release PR appears, check the merge-commit subject first.

### [2026-06-25] agy bakes secrets into JSON; opencode/pi self-decrypt (they ignore ambient env)

**Context**: Migrating setup off the load-secrets eager-source (which populated $NAN_API_KEY/$OPENROUTER_API_KEY in the setup process env for deploy-time config materialization).

**Problem**: Assumed both opencode and agy consumed the eager-loaded ambient env. They don't. `substitute_env_placeholders` (utils.sh) / `Substitute-EnvPlaceholders` (utils.ps1) resolve {env:VAR} by reading env-mapping.conf and age-decrypting the .secret.age file DIRECTLY -- they ignore the ambient env. Only the agy MCP block reads $env:OPENROUTER_API_KEY, because agy does NOT expand env vars inside JSON, so the key must be baked into mcp_config.json at deploy. So the eager NAN_API_KEY fetch was dead code, and env-mapping.conf can't be deleted while the substitute functions still read it.

**Solution**: B3 fetches only OPENROUTER_API_KEY via `dotf secrets show` for agy; dropped the dead NAN fetch. Left env-mapping.conf; tracked the substitute-functions -> registry migration (a future `dotf secrets render`) as the last step before deleting env-mapping.conf (#587).

**Rule**: Trace each secret consumer to its ACTUAL resolution path before migrating it. Two configs using {env:VAR} can resolve via completely different mechanisms (self-decrypt vs deploy-time bake vs runtime). Read the substitution function; don't assume ambient env.

### [2026-06-25] A new top-level dir backing a dotf runtime read must be deployed by setup

**Context**: #584 added secrets/registry.yaml and made deployed `dotf secrets {ls,show,run}` read it from $DOTFILES_DIR/secrets/registry.yaml. 0.19.0 shipped it.

**Problem**: setup-{linux,windows} only deployed sensitive/ and scripts/ into ~/.dotfiles -- never the new secrets/ dir. So on a 0.19.0 machine, `dotf secrets run` (and the opencode/pi/agy wrappers that call it) failed `read registry: ... cannot find the path`. A post-deploy smoke caught it; otherwise it would have broken the AI-CLI wrappers silently.

**Solution**: B2 (#591) deploys secrets/registry.yaml to $DOTFILES_DIR/secrets/, mirroring sensitive/. Stopgap-copied on the current machine to unbreak it immediately.

**Rule**: When a `dotf` subcommand reads a file from $DOTFILES_DIR at runtime, setup MUST deploy that file/dir -- adding it to the repo is not enough. Always smoke a deployed binary after a release; the smoke is what catches deployed-vs-source drift (redeploy is part of "done").

### [2026-06-25] CI gotchas: Set-Content CRLF on .sh, and repointing tests creates duplicate names

**Context**: PR-C deleted files and edited tests via PowerShell Set-Content and bats edits.

**Problem**: (1) PowerShell `Set-Content` writes CRLF on Windows. Applied to scripts/test.sh (.gitattributes eol=lf) it CRLF'd the whole file -> shellcheck SC1017 on every line locally (git normalizes on commit, but the working copy + local checks break). (2) Repointing a removed test to an existing assertion ("dotfiles-sync.sh is executable") created a duplicate @test NAME within verify-setup.bats -> bats refuses to parse the file, failing the `test` AND `integration` jobs.

**Solution**: Stripped CR from test.sh (`sed -i 's/\r$//'`). Deleted the repointed tests outright.

**Rule**: Don't rewrite .sh files with PowerShell Set-Content (it injects CRLF); use an LF-preserving edit or strip \r after. When removing a feature, DELETE its tests -- don't repoint them to another assertion, or you risk a duplicate @test name (a per-file bats parse error).

### [2026-06-25] Determinism "presence" is cheapest as instructions-file injection, not a provider hook

**Context**: ADR-027 cross-harness agent pipeline. The first cut of the curator dogfood (HARNESS-043) emitted a claude-only `SessionStart` hook into ~/.claude/settings.json to force an agent's skills into context.

**Problem**: The hook was claude-only (opencode/pi/copilot have no equivalent) and emitted a POSIX shell command, so it carried a Windows-vs-POSIX command-form axis -- it would not run on Windows without a separate native-command port. A presence mechanism that needs both a per-provider plugin AND a per-OS command form does not generalize; it is the silent-drift failure the pattern exists to prevent.

**Solution**: Every daily harness already loads a harness-managed instructions file (~/.claude/CLAUDE.md, ~/.config/opencode/AGENTS.md, ~/.pi/agent/AGENTS.md, ~/.copilot/copilot-instructions.md). "Presence" therefore equals injecting the forced-skills directive into that file via a marked region -- one uniform mechanism across all four harnesses, in a distinct AGENT-PRESENCE marker namespace that coexists with the patterns region. Text injection is cross-OS by nature, so the OS axis disappears.

**Rule**: For determinism, separate the LEVEL from the MECHANISM. Presence (skill in context every turn) is the cheapest level and needs no plugin -- a system-prompt hook that only ADDS TEXT is equivalent to injecting that text into an always-loaded file. Reserve the provider plugins (SessionStart / chat.system.transform / session_start / PreToolUse) for the Action level, where gating actually requires code. Default to the agnostic injection primitive; reach for a provider hook only when it buys something injection cannot.

### [2026-06-25] bats silently drops @test names with non-ASCII chars or duplicates — lint them

**Context**: HARNESS-043 (#607) had a `@test` name with an em-dash; bats 1.13.0 reported "executed 36 instead of 37" and exited 0. A prior lesson noted duplicate `@test` names break parsing. The auto-curation analyzer (CURATOR-001, #135) flagged the recurrence; implementing the proposed lint surfaced 6 more non-ASCII `@test` names already in the suite (em-dash, `<=`), 3 of them silently skipped in opencode.bats (44 declared, only 41 run).

**Problem**: bats translates `@test "<name>"` into a shell function name. Non-ASCII bytes make bats fail to register the function ("unknown test name"); duplicate names make it refuse to parse the whole file. In both cases the SUITE still exits 0 — a green test that never ran. CI's `bats tests/*.bats` was passing while 3 opencode tests were dead.

**Solution**: `scripts/check-bats-names.sh` scans `tests/*.bats` for (a) non-ASCII characters in `@test` names and (b) duplicate names within a file, failing with `file:line`. Wired into the CI `lint` job. Fixed the 6 existing violations (em-dash to `-`, `<=` glyph to ASCII `<=`), recovering the silently-skipped tests (opencode.bats 41 -> 44, all green). **Promoted to check**: `scripts/check-bats-names.sh`.

**Rule**: Keep `@test` names ASCII and unique within a file. A passing bats suite does not prove every test ran — only an executed-vs-declared count or a name lint proves that. If a test name carries an em-dash or a `<=` glyph, it is probably not running.

---

### [2026-06-26] Never read a locked secret store as "absent" — discriminate before create, or you spawn duplicates

**Context**: `dotf secrets set`/`migrate` (#612) write a value into a Bitwarden item, creating the item when it does not exist. `BWPut.SetField` deliberately refuses to create; the create path lives in the command (C3, #621).

**Problem**: `bw get item <x>` fails the SAME way (exit 1) whether the item is genuinely missing OR the vault is locked / not-logged-in. A create-if-absent path that treats *any* read failure as "absent" will, against a locked vault, CREATE a duplicate of an item that already exists but was merely unreadable — silently, and worse under a non-interactive `--yes`.

**Solution**: Discriminate by the store's specific signal, not a generic error. `BWGet` wraps `ErrBWItemNotFound` ONLY when bw's message is "Not found."; a locked/unauthenticated vault yields a different message and falls through to fail-loud. `applySet` switches on the sentinel, so the create branch is reached only on a genuine not-found. Belt-and-braces: create still needs an interactive confirm or `--yes`. (`ErrBWFieldNotFound` similarly separates "item present, field missing" = append, from "item missing" = create.)

**Rule**: A create-if-absent against an auth-gated store must distinguish "genuinely absent" from "unreachable/locked" by a *specific* signal (a typed not-found, a status probe) before creating — never on an ambiguous error. When the only signal is a CLI message, match it narrowly, gate the create behind a confirm/`--yes`, and document the fragility (a structured API like `bw serve`, #622, replaces the string-match later).

### [2026-06-26] On Windows, `bash` from PATH is the System32 WSL launcher, not Git Bash — resolve the real interpreter before shelling out

**Context**: `dotf mem session-start` ports `session-brief.sh` (HARNESS-026) and shells out to `vault-health.sh` via Go. The first cut used a bare `exec.Command("bash", script)` (the faithful port of the shell's `bash "$vault_health"`). It worked on Linux/macOS but on Windows the vault-health step failed with `execvpe(/bin/bash) failed: No such file or directory` (#629).

**Problem**: Windows ships `C:\Windows\System32\bash.exe` — the **WSL launcher**, not a real bash. It precedes Git Bash on `PATH` for most installs, so Go's `exec.LookPath("bash")` (and a bare `"bash"` argv, which the OS resolves the same way) picks it. Two distinct failure modes hide behind it: (1) with **no WSL distro installed** it is a broken stub that aborts with `execvpe(/bin/bash) failed`; (2) even with a working distro, WSL runs in the Linux namespace and **cannot read a Windows-path script argument** — it sees `/mnt/c/...`, not `C:\...`, so `bash C:\...\vault-health.sh` fails to find the script. A bare `"bash"` is therefore wrong on Windows in *both* WSL states.

**Solution**: Added `resolveBash()` (`cli/internal/mem/session_start.go`) and route every bash shell-out through it instead of a literal `"bash"`. Resolution order: `$DOTF_BASH` override → the first `bash.exe` on `PATH` that is **not** `%SystemRoot%\System32\bash.exe` (compared case-insensitively, the WSL launcher is skipped) → a bare `"bash"` fallback (Linux/macOS have no System32 ambiguity, so behaviour there is unchanged). dotfiles installs Git Bash; that is the interpreter that can run a Windows-path script. The same resolver is reused by the deploy doctor check (`cli/internal/doctor/checks_deploy.go`).

**Rule**: On Windows, never shell out to a bare `"bash"` (nor trust `exec.LookPath("bash")`) — it resolves to the System32 WSL launcher, which is a broken stub without a distro and cannot read Windows-path arguments even with one. Resolve a real interpreter first: prefer an explicit override env var, then skip the `%SystemRoot%\System32\bash.exe` candidate when scanning `PATH`, then fall back to `bash` only where there is no such ambiguity (POSIX). Cross-platform Go that ports a shell script must treat the interpreter itself as a resolution problem, not a constant.

---

### [2026-06-26] `secrets sync ci` refreshed `updated_at` on a dead PAT — a successful write is not a live credential

**Context**: After rotating-by-redeploy, `dotf secrets sync ci` uploaded `BITACORA_PAT` to the repo's Actions secrets and reported success. The board automation (`add-to-project`/`bitacora-status`) then failed every run with HTTP 401 — the uploaded token was expired at source. `sync` had verified the *write* (`gh secret set` succeeded, `updated_at` refreshed) but never that the *value still authenticates*.

**Problem**: A secret-sync tool's success criterion is "the write landed", not "the payload is live" — those are different claims. Uploading a 401 token looks identical to uploading a good one. Worse, the latent monitor that should have caught it (`pat-expiry.yml`) had been dead-on-arrival: the job has no `actions/checkout`, so its very first `gh` call died with `fatal: not a git repository` before probing anything (fixed by setting `GH_REPO`). So nothing — not the sync, not the monitor — actually checked liveness.

**Solution**: Three tiers. (0) Make `pat-expiry.yml` **fail the job** (red `::error` + `exit 1`) on an invalid/expired token, not just file an ignorable issue; fix the `GH_REPO` checkout bug so it runs at all. (1) Opt-in pre-upload liveness in `sync ci`: a registry entry marked `validate: github-token` is probed with `gh api user` (authenticating *as* the token under test) **before any upload**; a dead token aborts the whole sync. (2) Structural — migrate the board automation to a GitHub App installation token so the long-lived PAT stops existing.

**Rule**: Validating a secret's *liveness* is a separate concern from writing it, and from monitoring its expiry — do all three deliberately. Liveness validation does **not** generalize across providers (each is a bespoke probe, or none), so make it opt-in per credential, scoped to what you can cheaply probe (GitHub tokens via `gh api user`), and fail loud *before* the upload — never push a credential that does not authenticate. And the durable fix for "this PAT keeps expiring" is to delete the PAT (OIDC / GitHub App short-lived tokens), not to monitor it better.

### [2026-06-26] Name-match at the consumer boundary, decouple at the storage boundary

**Context**: `dotf secrets sync ci` (CLI-024-secrets-sync) uploads registry secrets to a repo's GitHub Actions secrets.

**Problem**: A secret crosses two boundaries with different naming pressures — the consumer boundary (`gh secret set` / a workflow's `${{ secrets.X }}` need an exact name match) and the storage boundary (Bitwarden's own item/field organization, grouped by service or account for a human browsing the vault). Forcing one naming scheme across both either breaks the consumer's exact-match requirement or fights the storage layer's own organizing logic.

**Solution**: The Actions secret name is always identical to the exposed env var — a flat 1:1 convention at the consumer boundary. Bitwarden storage (`bw: {item, field}`) stays decoupled and is free to group related secrets by service/account; the registry (`secrets/registry.yaml`) is the only place that maps between the two.

**Rule**: At a consumer boundary — anywhere a caller's literal expectation must match exactly (env var names, API param names) — name-match precisely. At a storage boundary — anywhere only the system itself reads the layout — organize for the storage's own convenience. Never let storage-layer naming leak into a consumer contract, and keep exactly one seam (here, the registry) that translates between the two.

### [2026-06-27] A CLI that reads its config from the *deployed* copy, not the checkout, silently reverts its own writes

**Context**: `dotf secrets` resolves `secrets/registry.yaml` (the mapping SSOT) to drive `show`/`run`/`migrate`/`set`. The first cut read and wrote the **deployed** copy at `~/.dotfiles/secrets/registry.yaml` — the same path setup rsyncs from the checkout on every redeploy. The first real C8 migrate (#635) never ran, so the footgun stayed latent until the `sync ci` smoke surfaced it.

**Problem**: Setup deploys by copying checkout → `~/.dotfiles`, so the deployed copy is a *derived artifact*, not a source. A `migrate` that flipped `backend: age → bw` in the deployed copy produced a write that (1) the next redeploy silently reverts and (2) never reaches git at all — a durability black hole that looks successful in the moment. Worse, it is a two-tier split-brain: `#636` fixed the registry to resolve checkout-first, but left the secret *files* (`sensitive/*.age`, `SecretsDir`) deployed-only, so the identical bug recurred for values — a token re-encrypted in the checkout was invisible to `dotf` until a redeploy, hit live during the RELEASE_TOKEN rotation (`#642`). (The same PR also caught an ADR-number collision — the new model was renumbered 029→030.)

**Solution**: Split the seam by intent. Reads prefer the checkout and fall back to the deployed copy (`env.ResolveRegistryPath` / `ResolveSensitiveDir`, "live-on-pull" with a graceful degrade); **writes** resolve the checkout only and **fail loud** when no checkout is found (`env.RepoRegistryPath`), so a mutation can never land in a derived artifact. `env.RepoDir` is the shared resolver (`DOTFILES_REPO_DIR` or a `.git` walk-up). Registry and values now share one source, so a checkout-side rotation or migrate is authoritative immediately and is committable.

**Rule**: When a tool both *reads* and *writes* a file that a deploy step copies from a source-of-truth, the deployed copy is a cache, not a store — never write to it. Resolve writes to the checkout/SSOT and fail loud if it is absent; reads may fall back to the deployed copy for ergonomics, but the read and write seams must be **separate** so a write never silently targets a derived artifact. And when you fix this for one file (the registry), audit every sibling the same deploy touches (the secret values) — a split-brain rarely has exactly one half.

### [2026-06-27] "Same set as the script it replaces" is the wrong parity gate when the old tool was itself wrong

**Context**: `dotf secrets sync ci` replaces `github-secrets-manager.sh` as the path that uploads secrets to a repo's GitHub Actions. The instinct when retiring the script was to gate the cutover on "the new command uploads the same set of secrets the script did" — set-equality as the parity proof.

**Problem**: The old script had **no consumer filter** — it pushed every pair it knew about to the repo, over-uploading secrets no workflow in that repo consumes. Gating on set-equality would have forced the new command to faithfully reproduce that over-upload, i.e. to inherit the bug as a spec. The script's behaviour was the *defect under repair*, not the reference to match.

**Solution**: Reframe parity as **functional coverage**, not set-equality. `sync ci` selects by intent: `reg.SelectCI(repo)` returns exactly the registry entries whose `consumers:` contains `ci:<repo>` (`secrets_sync.go`), so it uploads what the repo's workflows actually consume and nothing more. The cutover gate became "every secret the target repo's workflows reference resolves and uploads" (a grep of `.github/workflows/` for consumed names), verified per-repo — not "the byte-set equals the script's output". Migration of `ci:*` entries was scoped to the same evidence: only the names the workflows actually consume.

**Rule**: When replacing a tool, do not adopt its output as the correctness oracle — first ask whether the old behaviour was right. If the legacy tool was over-broad, permissive, or buggy, set-equality parity *encodes the bug* into the replacement. Define the gate from first principles (what does the consumer actually need?) and let the new tool's narrower, correct output differ from the old — then verify functional coverage, not byte-for-byte sameness.

### [2026-06-27] A successful operation is not evidence of the property you depend on — assert the property, not the success

**Context**: Four incidents in the secrets/CI surface within two cycles: `sync ci` refreshed a PAT's `updated_at` (#639); `pat-expiry` probed `GET /user` (#647); `setup` `curl`ed a release tarball (#648); `AgeDecrypt` reported a decrypt failure (#644).

**Problem**: Each conflated "the operation reported success/failure" with "the property I actually depend on is true". A **write** landed (`gh secret set` ok, `updated_at` bumped) but the value was an **expired** token (#639 — write ≠ live). An **auth** succeeded (`GET /user` 200) but the token lacked `Pull requests: write`, so release-please's PR-create 403'd and every release silently stalled (#647 — authenticates ≠ authorized). A **download** completed (curl followed redirects, wrote a file) but, lacking `-f`, it saved a 404 "Not Found" body *as* the `.tar.xz`, detonating later at `xz` (#648 — bytes arrived ≠ bytes are the artifact). And a decrypt **failed** loudly but as opaque `exit status 1`, hiding age's actual cause (#644 — failure reported ≠ cause surfaced). In every case the misleading signal looked identical to the healthy one, and the truth surfaced far downstream (a 401 in production, a red release run, a corrupt extract) instead of at the operation.

**Solution**: Assert the downstream property explicitly, right after the operation, failing loud. #639 → opt-in pre-upload liveness (`gh api user` authenticating *as* the token under test). #647 → a capability probe (`POST /pulls` with a non-existent head; 403 = missing scope, 422 = permitted — non-destructive). #648 → `curl -fsS` so an HTTP error is the curl's failure, not the next step's. #644 → capture the child's stderr and surface its message, not the bare exit code.

**Rule**: When a step's success stands in for something you actually rely on — the value is *live*, the credential is *authorized*, the bytes are the *artifact*, the error names its *cause* — verify that property directly; it does not come for free with the operation returning 0. The cheapest place to assert it is immediately after, fail-loud; the most expensive place to discover its absence is three steps (or three days) downstream, as a silent corruption that looks exactly like success.

### [2026-06-27] A "latest/stable" download URL rots silently, and `curl` without `-f` turns a 404 into a corrupt artifact

**Context**: `setup-linux.sh` installed shellcheck from `…/releases/latest/download/shellcheck-stable.linux.x86_64.tar.xz`. A from-scratch container shakeout (running setup in a clean Ubuntu image) found shellcheck **never installs on a fresh machine** — a bug invisible on any box that already had it.

**Problem**: Two chained silent failures. (1) ShellCheck **retired the `-stable` asset alias** after v0.10, so `latest/download/shellcheck-stable…` 302s to the new tag then **404s** — an unversioned URL that rotted the moment upstream renamed its assets. (2) `curl -Lo … 2>/dev/null` with **no `-f`** wrote the 404 body (`Not Found`, 9 bytes) *as* `shellcheck.tar.xz`; the failure only surfaced later at `tar/xz` as a misleading "File format not recognized". A developer's machine had shellcheck only because it was installed back when `-stable` existed; fresh setups silently lost it. CI never caught it either: the integration container lacked `xz-utils`, so the path died even earlier with yet another error, masking the real one.

**Solution**: Pin `SHELLCHECK_VERSION` in `versions.conf` (the version SSOT) and build both the URL and the tarball's internal dir (`shellcheck-v<ver>/`) from it; switch to `curl -fsSL` so an HTTP error fails the step instead of poisoning the extract. Make the test representative: add `xz-utils` to `Dockerfile.integration` (without it `tar xJf` could never run, so CI never exercised the install) and assert `~/.local/bin/shellcheck` exists in `verify-setup.bats`.

**Rule**: A `latest`/`stable` download URL is an **unversioned dependency that rots without warning** — pin the version (in the version SSOT) and derive the URL from it. Any `curl` fetching a file MUST use `-f`: without it, a 404 or redirect-to-error-page is written as the file and the corruption detonates far from its cause. And a test environment that omits a tool the real install path needs (here `xz`) doesn't test that path — it hides it behind a different error; make the test environment representative or the path is effectively untested.

### [2026-06-29] A uv tool's Windows launcher is a trampoline that orphans silently — and a running daemon blocks its own repair

**Context**: A session opened with the start-of-session `[hive]` banner printed, but `ToolSearch` for `vault_query`/`vault_search` returned nothing — the Hive MCP server's tools never registered. The config in `~/.claude.json` looked correct (`hive.exe`, right `HIVE_VAULT_PATH`), so it was not a config problem.

**Problem**: Running the configured binary directly exposed the cause: `hive.exe --version` → `error: uv trampoline failed to canonicalize script path`. On Windows `uv tool install` cannot symlink, so it writes a tiny launcher `.exe` (a **trampoline**) into `~/.local/bin/` that embeds the absolute path to the real venv under `%APPDATA%\uv\tools\hive-vault\`. That venv had been pruned/half-written out from under the trampoline (the same malformed state as #574 — venv missing `rich`), so the launcher survived but pointed at nothing and the MCP process never started. Two Windows-specific forces produced and then *protected* the broken state: (1) Windows cannot replace a running `.exe`, so an out-of-band `uv` upgrade against the live daemon left the env partially written; (2) the stale `hive serve` daemon (PID + its `python.exe` child) held an open handle to `hive.exe`, so even the repair — `uv tool install --force hive-vault` — rebuilt the venv but failed the final entrypoint copy with `os error 32` (file in use). The banner had masked all of this: it is emitted by a startup hook, **not** by the live MCP process, so "banner printed" was never evidence the server connected.

**Solution**: Stop the holder before repairing. Kill the Startup-folder supervisor FIRST (else it respawns the daemon mid-copy — and exclude `$PID` from the match, since the kill script's own text contains the supervisor name), then the daemon + its python child, then `uv tool install --force hive-vault` (entrypoint copy now lands), then verify two things, not one: `hive.exe --version` resolves AND a real MCP `initialize` returns a JSON-RPC result with `serverInfo`. MCP connects at session start, so the running session can't see the fix — restart Claude Code. Captured the full recipe in `docs/troubleshooting/hive-mcp-orphaned-trampoline.md`; the durable cross-machine guard is #574 (`dotf doctor --fix`), left open because this fix was manual.

**Rule**: On Windows, a uv-installed CLI in `~/.local/bin/*.exe` is a **trampoline, not the program** — "the launcher exists" ≠ "the tool runs"; diagnose by invoking it (`--version`) and reading the error, never by checking the file is present. When repairing a tool whose **running process holds its own executable**, stop every holder (and its supervisor, so it can't respawn) *before* the reinstall, or the lock turns `--force` into a half-install. And when testing a **stdio MCP server**, keep stdout pure JSON-RPC: the FastMCP startup banner goes to stderr, so a `2>&1` merge makes a healthy server look broken — separate the streams (`2>/dev/null`) and assert on the `initialize` result.

### [2026-07-01] A CLI's `--help`/`Long` strings are untested literals — a dangling doc ref ships green

**Context**: The DR-escrow slice (#661) shipped `dotf secrets backup` whose `Long` help referenced a `guide-secrets-recover.md` that was never created — the recover protocol was (correctly) hardened into `guide-secrets-governance.md` instead, so the referenced file never existed. Every behaviour test was green and the command worked; the dangling reference was caught only because a human read the real `--help` output during review.

**Problem**: A Cobra command's `Short`/`Long`/`Example` are plain string literals. Behaviour tests invoke `RunE` and assert on the command's *effects*; they never render `--help`, so the help text is a user-facing surface that **no test exercises**. A broken flag description, a stale path, or a reference to a file that does not exist sails through `go test` untouched and only surfaces when a user (or a lucky reviewer) reads `--help`. The escrow's dangling `guide-secrets-recover.md` was exactly this class: invisible to the whole suite, visible only to a human.

**Solution**: Treat help text as testable output (`cli/internal/cmd/help_smoke_test.go`). (1) `TestEveryCommandHelpRenders` walks root + every subcommand and runs the real `--help`, asserting a `Usage:` block renders with no error — catching panics and broken templates. (2) `TestHelpDocReferencesExist` scans each command's `Short`/`Long`/`Example` for `docs/….md` references and asserts each file exists in the repo — the precise guard for the dangling-ref class. Red-teaming the guard itself (injecting a bogus `docs/…` ref and confirming it goes red) exposed a gap in the first cut: it scanned only *subcommands*, so a dangling ref in the **root** command slipped through — scanning root + subcommands closed it.

**Rule**: Help and usage literals are user-facing output, so test them like output — render `--help` for every command in CI, and assert any concrete repo file a help string names actually exists. A guard you have never watched fail may be blind: inject the exact fault it targets, confirm it goes red, then revert — here that step is what revealed the root command was unscanned. Scope reference checks to unambiguous repo paths (`docs/….md`) so the guard stays precise and false-positive-free; vault paths (`MEMORY.md`) and template patterns (`specs/<id>/…`) are deliberately excluded.

### [2026-07-07] `bash` on PATH via scoop is not GNU Bash — it silently mis-executes bashisms

**Context**: Adding two new skills required running `scripts/compile-harness.sh --refresh` from a Windows machine to render them into the committed `harness/skills/` record. `bash` resolved on PATH to `C:\Users\mlorente\scoop\shims\bash.exe`.

**Problem**: Invoking the script through that shim failed immediately with `syntax error: bad substitution`, then a cascading `[ERROR] manifest not found: C://harness/manifest.json` — a path that could not have come from the script's own real repo-root resolution logic. `bash --version` returned `bad option '--version'`, and `$HOME` resolved to the malformed `C:Usersmlorente` (no separators). All three symptoms point the same way: scoop's `bash` shim is not GNU Bash — it is a minimal shell (BusyBox `ash` or equivalent) that accepts the name `bash` but rejects real bash syntax (parameter expansion, `--version`) and does not populate the environment the way real Bash does. The script never had a chance to run; it failed inside its own `#!/usr/bin/env bash` shebang-equivalent invocation.

**Solution**: Use Git for Windows' real Bash instead — `C:\Program Files\Git\bin\bash.exe` (already present on any machine with Git installed) — and invoke the script explicitly through its full path rather than relying on whatever `bash` resolves to on PATH. The script then ran correctly end-to-end (skill records rendered, `[refresh] OK`).

**Rule**: On Windows, never trust a bare `bash` on PATH to be GNU Bash — multiple tools (scoop, WSL stubs, Git for Windows, MSYS2) can all provide something answering to that name with materially different capabilities. Before running any bash script that isn't trivially POSIX, verify with a cheap probe (`bash -c 'echo ${x//a/b}'` or just `bash --version`) or invoke Git for Windows' bash by its full path (`C:\Program Files\Git\bin\bash.exe`) directly — it is the one on this machine confirmed to be real Bash.

### [2026-07-08] A guard test that names its own trigger string can match itself once tracked

**Context**: GUARD-002 (#669) added `tests/sensitive-hygiene.bats`, asserting `git grep -l "docs/SECRETS.md"` returns no matches, to catch a future resurrection of a dead doc reference.

**Problem**: The test file's own source contains the literal string `"docs/SECRETS.md"` as the grep pattern argument. `git grep` only searches tracked files, so the assertion passed locally against the untracked new test file (written but not yet `git add`-ed) — then failed in CI once the file was committed and became a match for its own search pattern.

**Solution**: Exclude the guard file itself from its own search scope via a git pathspec exclusion (`git grep -l "pattern" -- . ':!tests/sensitive-hygiene.bats'`).

**Rule**: Before trusting a "no references to X" guard test locally, make sure your local run sees the same tracked state CI will — stage the new test file first, or otherwise verify against the committed tree, not the working tree. An untracked new file is invisible to plain `git grep` and can produce a false-clean local pass that only breaks in CI. Separately: any guard whose search pattern literally appears in its own source needs an explicit self-exclusion, or it will eventually fail on itself.

### [2026-07-08] Keeping a secret off curl's argv: `-K -` (stdin config) is portable; process-substitution and `mktemp` are not

**Context**: #687 (audit C26) required moving a bearer token out of `curl -H "Authorization: Bearer $KEY"` — argv is world-readable via `/proc/<pid>/cmdline` for the call's duration — in the `nan-*` benchmark scripts. The issue suggested curl's `-H @file` form. The scripts run on Linux but were being edited and tested from a Windows box, so the mechanism had to survive both.

**Problem**: The obvious "no temp file" idiom — process substitution `-H @<(printf 'Authorization: Bearer %s' "$KEY")` — failed on Windows curl with `curl: Failed to open /proc/<pid>/fd/63`: a native Windows curl (Schannel/mingw32) cannot resolve the MSYS `/dev/fd` pseudo-path that Git Bash hands it. The temp-file idiom (`mktemp` + `-H @file`) worked, but `mktemp` creates the file `0644` on MSYS (not `0600` as on Linux), so the header file holding the secret is briefly world-readable unless an explicit `chmod` follows. Both facts were found empirically against a local listener — not from docs, which describe none of these platform quirks.

**Solution**: Feed the whole auth header to curl through a config read from stdin: `curl -K - <<CFG` / `header = "Authorization: Bearer $KEY"` / `CFG`. The secret lives only in stdin (never argv, never disk), it is portable (no `/dev/fd`, no temp-file perms), and needs no cleanup. Verified against a local HTTP listener that the header arrives and the token never reaches any process's argv. For `gh secret set`, the equivalent is piping the value on stdin — `printf '%s' "$PAT" | gh secret set NAME` — using `printf` (not `echo`) so no trailing newline corrupts the token.

**Rule**: To keep a secret off a child process's argv, prefer feeding it via **stdin** (`curl -K -`, `gh secret set` piped) over a temp file — and never use process substitution for it in code that may run under Windows curl, where the `/dev/fd` path is unresolvable. When a temp file is unavoidable, remember `mktemp` is not `0600` on every platform (MSYS makes it `0644`); `chmod 600` before the secret is written. And prove the mechanism empirically (a local listener + an argv check) rather than trusting that a documented flag behaves identically across curl builds.

### [2026-07-09] A three-dot `origin/BASE...HEAD` diff needs the merge-base — `--depth=1` starves it, and a fail-closed gate makes that loud

**Context**: The C3 fail-closed hardening (#686/#716) made `check-spec-gate.sh` exit 2 whenever `git diff "origin/BASE...HEAD"` cannot resolve its refs, instead of silently passing with `TOTAL_LOC=0`. The next PR to run the gate — #728, whose branch lagged `main` by 6 commits — failed `spec-gate` with `base/head ref could not be resolved. The Discipline Gate fails closed (exit 2).`

**Problem**: `spec-gate.yml` fetched the base ref with `git fetch --no-tags --prune --depth=1 origin "$BASE_REF"`. The gate diffs `origin/BASE...HEAD` — the **three-dot** form, which git resolves as `git diff $(git merge-base origin/BASE HEAD) HEAD`. A depth-1 base tip carries no history, so for any PR branch that lags `main` by more than one commit the merge-base is not reachable from the shallow base ref → `git diff` errors → the gate (correctly) fails closed. The C3 change did **not** introduce the bug: the shallow fetch was always inadequate for a three-dot diff; C3 only stopped it from passing silently as a zero-LOC no-op. This is exactly the "shallow/fresh clone" case #686/C3 warned about, now made loud instead of latent.

**Solution**: Drop `--depth=1` from the base-ref fetch — one line: `git fetch --no-tags --prune origin "$BASE_REF"`. `actions/checkout` already fetches full head history (`fetch-depth: 0`); the full base fetch makes the merge-base always reachable. Verified end-to-end: #728 (rebased onto current `main`) and #729's own `spec-gate` run both pass under the fixed workflow. Validated on #730, a branch deliberately behind `main`, which now passes.

**Rule**: A three-dot diff (`A...B`) is defined relative to the **merge-base** of A and B, so both sides need enough history to *find* that merge-base — a `--depth=1` fetch of either ref starves it the moment the branch is more than one commit behind its base. In CI, whenever a gate uses a three-dot diff or `git merge-base`, pair `fetch-depth: 0` on `actions/checkout` with a **full (non-shallow) fetch of the base ref** — never `--depth=1`. And remember the interaction that surfaced this: a fail-closed gate does not create shallow-fetch bugs, it *exposes* pre-existing ones as hard failures — so when hardening a check to fail closed, audit every ref-resolution the check depends on in the same change, because faults that used to pass silently will now block every PR.

### [2026-07-10] A clean local `golangci-lint` does not certify CI — v1 default-excludes errcheck Close/Remove, v2 does not

**Context**: BUG-029 (#696) added an atomic `machine.json` writer with `defer os.Remove(tmpName)` and a `tmp.Close()` in an error branch. `golangci-lint run` was clean locally (the machine's binary was v1.62.2), so the change was pushed as done.

**Problem**: The `cli.yml` `lint` job (`golangci-lint-action@v8`) failed with exactly two errcheck findings — `os.Remove` and `tmp.Close` return values not checked — on those two lines. The action pins golangci-lint **v2**, and v2 dropped v1's default-on exclusion set (`issues.exclude-use-default`), whose built-in list suppressed the stock "Error return value of `(*os.File).Close`/`os.Remove` is not checked" reports. So identical code passes v1 locally and fails v2 in CI: a linter **version** mismatch, not a config or code difference. `go vet` never flags these either, so the local gates were all falsely green.

**Solution**: Write the discards explicitly, matching the pattern the repo already uses (`internal/doctor/checks_secrets_tooling.go`): `_ = tmp.Close()` and `defer func() { _ = os.Remove(tmpName) }()`. The explicit-discard form satisfies errcheck on every version and documents that ignoring the error is deliberate. Confirmed by reading the failed job log (`gh run view <run> --log-failed` → "errcheck: 2") rather than re-guessing.

**Rule**: A clean local `golangci-lint run` certifies nothing when the version differs from CI — `golangci-lint-action@v8` runs golangci-lint v2, which enables errcheck for `Close`/`Remove` that v1 excluded by default. Either pin the CI version locally (check the action's major version → linter major) or, better, write `_ = x.Close()` / `defer func() { _ = os.Remove(...) }()` explicitly so the code is linter-version-agnostic. When a Go PR's `lint` job is red but local is green, read the failed job log for the exact `(linter)` tag before touching code — the fix is usually mechanical once the specific linter is known.

### [2026-07-14] A characterization test can pin the bug you are removing — grep every test extension, not just the source

**Context**: BUG-031 (#689) fixed the Windows Claude project-key encoding by deleting a local `Get-EncodedPath` (which mapped the drive `:` to `''`, the bug) and routing through a shared `dotf`-backed helper. Before pushing I grepped the repo for `Get-EncodedPath` — but only across `*.ps1`. Local Go tests and the Pester guard were green, so the PR looked done.

**Problem**: CI's `test` and `test-windows` jobs failed on two bats cases in `tests/knowledge-crystallize-ps1.bats`: one asserted `grep -q 'function Get-EncodedPath'` (the function I had just deleted), the other asserted `grep -q "Replace.*':'.*''"` (the exact colon-deleting pattern that WAS the bug). These were characterization tests written against the original behavior, so a correct fix flipped them red. My `*.ps1`-only grep never saw them because the stale assertions lived in a `.bats` file, and the local Go + Pester suites did not include that bats file.

**Solution**: Re-point both bats cases at the corrected reality — assert the script sources `utils.ps1` and uses `Get-ClaudeProjectKey` with no local encoder, and invert the colon test to assert the buggy `Replace ':' ''` is **absent** and the decoder expects the double-dash key. Confirmed by running `bats tests/knowledge-crystallize-ps1.bats` locally (16/16) before pushing the follow-up commit; CI then green.

**Rule**: When a fix removes or renames a symbol, or changes an observable string, grep the **whole test tree across every extension** (`*.bats`, `*.Tests.ps1`, `*_test.go`) for the old name/pattern before pushing — a green local run only covers the suites you actually ran, and a characterization test that encoded the old behavior will fail precisely *because* the fix is correct. Treat a test that asserts a bug's fingerprint (a specific buggy pattern) as a liability: when you kill the bug, invert the test to guard against its return in the same change.

### [2026-08-04] zsh expands aliases at parse time, and the resulting parse error still exits 0

**Context**: `.zsh/functions.sh` held the Gemini saved-prompt helper. It had already been renamed once — `gp` → `gpr` — after colliding with oh-my-zsh's `alias gp='git push'`. The rename picked `gpr`, which oh-my-zsh's git plugin also owns (`alias gpr='git pull --rebase'`, `git.plugin.zsh:269`). `.zshrc` loads oh-my-zsh at line 13 and sources `functions.sh` at line 135, so the alias was always live first.

**Problem**: zsh expands aliases during **parsing**, not at invocation, so `gpr() {` parsed as `git pull --rebase () {` → `defining function based on alias` + `parse error near '()'`. The damage was not the message: a parse error aborts the rest of the sourced file, so the `utils.sh` load in the file's **last** block never ran, and every zsh session silently lost the shared library (`log_info`, `version_gte`, `deploy_file`). bash was unaffected — the git plugin is zsh-only. The 147-test suite missed it twice over: `tests/shell-wrapper-dedup.bats` sourced `functions.sh` in a bare `zsh -c` where no oh-my-zsh alias is live, and — decisively — **zsh returns exit status 0 after this parse error**, so the suite's `[ "$status" -eq 0 ]` assertions passed on a truncated file.

**Solution**: Renamed the helper to `agyp`, out of the `g*` namespace that the git plugin owns (~150 aliases) instead of picking a third name inside it. Added `tests/shell-alias-collision.bats`, whose tests assert **reach** — that `version_gte`, defined via the file's last block, still resolves after sourcing with `alias gp`/`alias gpr` pre-defined — rather than exit status. Verified the guard by reverting `functions.sh` to the pre-fix version: 4 of 5 tests went red, including the reach test in both bash and zsh.

**Rule**: Never name a shell function inside a namespace an installed plugin owns; when a name collides, leave the namespace rather than pick another name inside it — this repo hit the same rake twice (`gp`, then `gpr`). When testing that a sourced rc file loads correctly, assert that a symbol defined **after** the suspect line resolves; never assert exit status alone, because zsh reports a parse error on stderr and still exits 0. And source the file in an environment that has the aliases the real shell will have, or the collision cannot occur during the test.

### [2026-08-04] Validating config files in isolation cannot catch a broken reference between them

**Context**: Adding the `deepseek-v4-flash-0731` model to pi touches two files that must agree: `ai/pi/models.json` declares the model `id`, and `ai/pi/settings.json` enables it as `nan/<id>`. `tests/pi-config.bats` already had seven assertions over these files.

**Problem**: The settings entry read `nan/deepseek-v4-flash 0731` — a space where the id has a hyphen — so the model would never have resolved. The new model also carried the display name of the model beside it (`"deepseek (NaN)"`), which would have shown two indistinguishable entries in the picker. All seven tests passed: both files were valid JSON, neither held a literal API key, the `{env:NAN_API_KEY}` placeholder was present, no banned OpenRouter provider was named. Every assertion validated one file **in isolation**, and both defects lived in the relationship *between* the files, which nothing checked.

**Solution**: Four guards in `tests/pi-config.bats` (#749): every `nan/*` entry in `enabledModels` resolves to an id in `models.json`; `defaultModel` resolves; model ids are unique; model display names are unique. The id-uniqueness and name-uniqueness checks are deliberately separate — the two ids differed (`deepseek-v4-flash` vs `…-0731`) and only the names collided, so one check could not have caught the other's defect. The reference check reads its input line by line rather than word-splitting: an earlier draft split on whitespace and reported only the fragment `0731`, hiding the shape of the bug; it now reports `'nan/deepseek-v4-flash 0731'` whole.

**Rule**: When two config files must agree, per-file validity checks (valid JSON, no secret, no banned value) prove nothing about the pairing — add an explicit assertion that every cross-file reference **resolves**, plus uniqueness on every field a human or UI selects by. Derive the referenced set from the source of truth at test time rather than restating it, so the guard cannot drift. And when a guard reports a bad value, print it quoted and whole: a message mangled by word splitting sends the reader after the wrong defect.

### [2026-08-05] A config file the tool itself rewrites must be seeded, not synced

**Context**: `setup-{linux,windows}` deploy three pi files side by side. `models.json` and `tui.json` are dotfiles-owned: the deploy copies them whenever source and destination differ, which is correct — the repo is the source of truth and any local edit is drift to be corrected. `ai/pi/settings.json` was given the same shape, and `ai/pi/README.md` plus `tests/pi-config.bats` have described it as seed-if-missing since AI-025.

**Problem**: pi rewrites `settings.json` at runtime — `lastChangelogVersion`, `theme`, and the model picked in the TUI — and `tests/pi-config.bats` *forbids* the committed copy from carrying `lastChangelogVersion` at all. The two files therefore can never be byte-identical once pi has run: `cmp -s` could only ever fail, the `already in sync` branch was unreachable, and every setup run silently reset the user's theme and default model. The bug had shipped on both platforms since AI-025 and surfaced only because a doc edit prompted someone to read what the code beside it actually did. Nothing was red: the integration container seeds a fresh `HOME`, so it exercises the first run and never the second, which is where the whole bug lives.

**Solution**: Guard the copy on the destination being absent, on both platforms (#756). Add source-level assertions that the shape cannot drift back, since the container cannot observe run two, and state that reason in the test so the next reader does not "improve" it into a behavioral test that silently covers nothing.

**Rule**: Before choosing a deploy policy, ask who writes the file at runtime. A file only dotfiles writes is *synced* (copy when different); a file the installed tool also writes is *seeded* (copy when absent) and every later run must leave it alone. A byte-comparison against a self-mutating destination is not a weak check, it is dead code — if the file is guaranteed to differ, the comparison has no true branch. And when a test cannot cover a code path (here, the second run of a bootstrap script), say why in the test body: an unexplained source-level assertion is the first thing a future reader deletes.

### [2026-08-05] A guard can be green because its assertion never ran

**Context**: The fix above shipped with two source-level assertions, one per platform, each verifying that the pi settings deploy is guarded on the destination being absent and carries neither `Compare-Object` nor `-Force`. Both were green on the first run, and the Linux one was genuinely correct.

**Problem**: The Windows assertion was green for two independent wrong reasons, neither visible from a passing run. First, `setup-windows.ps1` is CRLF (`.gitattributes`), so the `sed` range ending at `/^}$/` never closed — a CR sits before every newline — and the extracted "block" was the remaining 842 lines of the file rather than the intended 10. Second, `grep -qF '-Force'` parses a leading-dash pattern as **options** (`-F -o -r -c -e`), leaving `-e` without its argument; grep exits 2, so the `&&` that reports the failure never fired. The assertion could not have failed even with `-Force` present on the exact line it was written to catch — and it was, 27 times, in the unrelated code the runaway range had swallowed.

**Solution**: Strip CR before matching, pass the pattern as `grep -qF -e '<pattern>'`, and assert the extracted block is small — that size check is what converts a range that fails to close into a red test instead of a silently vacuous one. Confirmed by planting each defect in turn: re-adding `-Force` is now red, and was **green** before the repair.

**Rule**: A new guard is not done when it passes; it is done when you have watched it fail. Plant the exact defect it exists to catch and confirm red — a guard verified only in the green direction is indistinguishable from one that asserts nothing. Two mechanical traps make this cheap to get wrong: any pattern starting with `-` needs `grep -e` (or `--`), and any line-anchored `sed`/`grep` range over a CRLF file must strip CR first or the anchor silently never matches. When a test extracts a region of a file, assert the region's size too, so a range that runs to EOF fails loudly instead of quietly widening what the test claims to cover.

### [2026-08-06] An enforcement gate fails in two directions, and the cheap one is the refusal

**Context**: Adding the archive-on-merge half of the SDD Discipline Gate (#670) to `check-spec-gate.sh`, which runs under `set -euo pipefail`. The suite was written first and deliberately covered both directions: ten tests asserting the gate *fires* on a violation, ten asserting it stays silent on `Refs #N`, on a prose mention, on a cross-repo reference, on an empty PR body, on a spec with no `issue:` frontmatter.

**Problem**: The first implementation passed every "must fire" test and failed four of the "must not fire" ones — and both causes were `set -e` interactions on the *nothing-matched* path, which for a gate is the normal path. `grep` exits 1 when it finds no closing keyword, so under `pipefail` the capture aborted the entire script: every ordinary PR whose body said `Refs #N` would have been blocked by an exit code that had nothing to do with the rule. Separately, `[[ -n "$num" ]] && printf …` as the last command of a `while` body made the enclosing loop exit 1 whenever a spec carried no `issue:` field — 28 of 44 active specs — and the caller, correctly written to fail closed on an unreadable tree, read that as an unreadable tree. A gate that refuses valid work is not "safely strict"; it is broken in the direction that costs the most, because every author hits it and none of them can tell why.

**Solution**: Guard the no-match path explicitly (`grep … || true`, with the reason in a comment) and replace the `&&` idiom with an `if` block so the loop body cannot leak a falsy status. Then dogfood the finished gate against the repository itself rather than only fixtures: run it on the real branch with `Closes #670` (must fail, naming the spec) and with `Refs #670` (must pass). That surfaced a further gap no fixture had — reading active specs only at the base ref misses a PR that creates a spec and closes its issue in the same change, which is precisely the "created, shipped, never archived" pattern the gate exists to stop.

**Rule**: For anything that can block work, write the negative tests first and treat them as the primary suite — "does not fire when it shouldn't" is harder to get right than "fires when it should", and its failures are invisible until they are blocking someone. Under `set -euo pipefail`, audit every command whose *success* case is "found nothing": `grep`, a `[[ … ]] &&` as a loop body's last statement, an empty `read` loop. They are the ones that turn a clean pass into an abort. And a gate is only verified once it has been dogfooded on real data — fixtures encode the cases you already thought of, the repository encodes the ones you did not.

### [2026-08-06] A guard installed machine-wide can silently disable every other guard

**Context**: GUARD-001 enforces its memory-sink check everywhere by setting a **global** `core.hooksPath`. Its dispatcher was written knowing that this makes git ignore `.git/hooks/` entirely, so it deliberately chains onward — the comment in `git-hooks/pre-push` says it exists so "per-repo guards (gitleaks) survive". The intent was right and the code did exactly what it said.

**Problem**: `pre-commit install` refuses to run while `core.hooksPath` is set (`Cowardly refusing to install hooks with core.hooksPath set`), so `.git/hooks/pre-push` was never created in the first place. The dispatcher chained to a file that did not exist and returned its clean no-op, exit 0. The knowledge vault's `gitleaks` secret scan had therefore not run on a single push — one guard had disabled another, and both were reporting success. Worse, `dotf doctor --fix` offered `pre-commit install` as the remedy for that exact FAIL: the repair was the very command the first guard blocks, so the diagnosis was correct, the fix was impossible, and nothing in the output said so. The same theme surfaced twice more the same day (#761): the dispatchers are extensionless, so `.gitattributes`' `*.sh eol=lf` misses them and a Windows checkout gives them a `\r` shebang that dies with exit 127 — a guard whose liveness depends on which `bash` happens to run it.

**Solution**: Make the dispatcher hand the stage to `pre-commit hook-impl` when there is no local hook but a `.pre-commit-config.yaml` exists, which restores every repo's gates without touching `core.hooksPath`. Notably **not** `pre-commit run --hook-stage`, which the issue had proposed: `run` accepts no stdin, and git delivers a pre-push hook's ref list *on stdin*, so it would have scanned the staged file set instead of the commits being pushed — green for the wrong reason.

**Rule**: A security control must assert it is **effective**, not that it is installed. "The hook file exists", "the setting is set", "the dispatcher ran" are all satisfiable while nothing is being checked; the only honest test fires a real violation and watches it get blocked. Two corollaries. When one enforcement mechanism claims a shared resource machine-wide — `core.hooksPath`, a PATH shim, a global config — enumerate what else uses that resource, because it is now yours to keep alive. And a repair action that cannot succeed is worse than no repair action: if a fix is structurally blocked by another component, say so in the output instead of proposing it every run.

### [2026-08-06] GraphQL's primary rate limit is billed to the account, not the token

**Context**: `dotfiles#530` already documented an agent-side fallback (runbook §8a) for when the bitácora board's GraphQL pool runs dry: fall back to REST for issue data, degrade to waiting for the reset for board fields. `add-to-project.yml` and `bitacora-status.yml` kept failing anyway whenever an interactive Claude Code session ran a few full `gh project item-list` sweeps — three separate incidents across two days.

**Problem**: The obvious fix — give the interactive session and the CI automation separate tokens, so one exhausting its pool can't touch the other — does not work, and OPS-007 (per-purpose PAT convention) would not have prevented this. `BITACORA_PAT` and a personal OAuth token are different token strings, but both authenticate as the same GitHub account, and GitHub's primary GraphQL limit (5,000 points/hour) is billed to the *account*, not the token: the error is literally `API rate limit exceeded for user ID <id>`, never "for this token". Confirmed live, not just inferred: filing `dotfiles#774` from an interactive session that had just drained the pool, `add-to-project.yml` fired on the `issues: opened` webhook 3 seconds later and failed with the identical error — a different credential, same account, same exhausted bucket.

**Solution**: The one mechanism that would truly isolate the pools — a GitHub App installation token, which gets its own budget — is a dead end here per ADR-031: Apps can't write to a user-owned Projects v2 board, only an org-owned one, and moving the board to an org is out of scope. With isolation closed off, the fix is tolerance instead: soft-fail the board mutations on a rate-limit-specific 403 rather than hard-failing the job, and run the already-idempotent `scripts/bitacora-rollout.sh backfill` on a schedule so a soft-failed event gets reconciled later instead of silently lost. Filed as `dotfiles#774` (OPS-022).

**Rule**: Before reaching for "split the credential" as a fix for a shared rate limit, check who the tokens authenticate *as*, not just what string they are. Two PATs owned by the same account still share that account's primary limit — separating tokens only isolates blast-radius (OPS-007's concern), never quota. When a shared-account collision can't be engineered away (no App-token path available for the resource in question), degrade gracefully — soft-fail plus scheduled reconciliation — rather than treating every transient exhaustion as a hard failure.
