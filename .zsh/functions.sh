# shellcheck shell=bash
#
# Swiss-army shell functions (IDEAS-002). POSIX-portable: works in bash AND zsh.
# Sourced by both ~/.zshrc and ~/.bashrc. Curated from mathiasbynens/dotfiles
# (.functions); each is self-contained with graceful degradation.
#
# Convention: `.zsh/functions.zsh` holds zsh-only helpers (zsh prompt expansion,
# etc.); THIS file (`.sh`) holds portable helpers shared by every shell. Keep it
# clean under both ShellCheck and `bash -n`/`zsh -n`.
#
# Name-collision escape hatch (IDEAS-001): ~/.zshrc.local / ~/.bashrc.local are
# sourced last and can re-alias any of these (e.g. if `server`/`gz` shadow a
# binary you need). See README "Shell helpers".

# mkd <dir>: make a directory (with parents) and cd into it.
mkd() {
    if [ -z "${1:-}" ]; then
        printf 'Usage: mkd <dir>\n' >&2
        return 1
    fi
    mkdir -p -- "$1" || return 1
    cd -- "$1" || return 1
}

# gz <file>: report original vs gzipped size and the savings ratio. Read-only —
# never writes a .gz file.
gz() {
    if [ -z "${1:-}" ] || [ ! -f "$1" ]; then
        printf 'Usage: gz <file>\n' >&2
        return 1
    fi
    local origsize gzipsize ratio
    origsize=$(wc -c < "$1")
    gzipsize=$(gzip -c "$1" | wc -c)
    ratio=$(awk -v o="$origsize" -v g="$gzipsize" \
        'BEGIN { if (o > 0) printf "%.1f", g * 100 / o; else print "0" }')
    printf 'original: %d bytes\n' "$origsize"
    printf 'gzipped:  %d bytes (%s%% of original)\n' "$gzipsize" "$ratio"
}

# dataurl <file>: print a base64 data: URI for a file. Falls back to
# application/octet-stream when `file` (MIME detection) is unavailable (R2).
dataurl() {
    if [ -z "${1:-}" ] || [ ! -f "$1" ]; then
        printf 'Usage: dataurl <file>\n' >&2
        return 1
    fi
    local mime
    mime=$(file -b --mime-type "$1" 2>/dev/null) || mime=""
    [ -z "$mime" ] && mime="application/octet-stream"
    case "$mime" in
        text/*) mime="$mime;charset=utf-8" ;;
    esac
    printf 'data:%s;base64,%s\n' "$mime" "$(base64 < "$1" | tr -d '\n')"
}

# targz <file-or-dir>: create <input>.tar.gz, preferring the best available
# compressor (zopfli for inputs < 50MB, else pigz, else gzip). All three emit a
# gzip-compatible stream, so the result is always `gzip -d`-decompressible (R1).
targz() {
    if [ -z "${1:-}" ] || [ ! -e "$1" ]; then
        printf 'Usage: targz <file-or-dir>\n' >&2
        return 1
    fi
    local input="$1" archive size compressor
    archive="${1%/}.tar.gz"
    size=$(du -sk -- "$input" 2>/dev/null | awk '{print $1 * 1024}')
    [ -z "$size" ] && size=0

    if command -v zopfli >/dev/null 2>&1 && [ "$size" -lt 52428800 ]; then
        compressor="zopfli"
    elif command -v pigz >/dev/null 2>&1; then
        compressor="pigz"
    else
        compressor="gzip"
    fi

    if [ "$compressor" = "zopfli" ]; then
        # zopfli has no stdin filter mode: compress a temp tar into <input>.tar.gz.
        local tmptar="${archive%.gz}"
        tar -cf "$tmptar" -- "$input" || return 1
        zopfli "$tmptar" && rm -f "$tmptar"
    else
        tar -cf - -- "$input" | "$compressor" > "$archive" || return 1
    fi
    printf '%s (via %s)\n' "$archive" "$compressor"
}

# server [port]: serve the current directory over HTTP (default 8000) and open a
# browser if an opener exists. Blocks until Ctrl-C. Requires python3.
server() {
    local port="${1:-8000}" opener=""
    if ! command -v python3 >/dev/null 2>&1; then
        printf 'server: python3 not found\n' >&2
        return 1
    fi
    if command -v xdg-open >/dev/null 2>&1; then
        opener="xdg-open"
    elif command -v open >/dev/null 2>&1; then
        opener="open"
    fi
    # Best-effort, non-fatal browser open once the server is likely bound.
    [ -n "$opener" ] && ( sleep 1; "$opener" "http://localhost:${port}/" >/dev/null 2>&1 & )
    python3 -m http.server "$port"
}

# getcertnames <host[:port]>: print the Common Name and Subject Alternative
# Names from a host's TLS certificate. Requires openssl + network. Default 443.
getcertnames() {
    if [ -z "${1:-}" ]; then
        printf 'Usage: getcertnames <host[:port]>\n' >&2
        return 1
    fi
    if ! command -v openssl >/dev/null 2>&1; then
        printf 'getcertnames: openssl not found\n' >&2
        return 1
    fi
    local hostport="$1" host port cert
    case "$hostport" in
        *:*) host="${hostport%:*}"; port="${hostport##*:}" ;;
        *)   host="$hostport";      port="443" ;;
    esac
    cert=$(openssl s_client -connect "${host}:${port}" -servername "$host" </dev/null 2>/dev/null \
        | openssl x509 -noout -text 2>/dev/null)
    if [ -z "$cert" ]; then
        printf 'getcertnames: no certificate retrieved for %s:%s\n' "$host" "$port" >&2
        return 1
    fi
    printf 'Common Name:\n'
    printf '%s\n' "$cert" | grep 'Subject:' | grep -oE 'CN ?= ?[^,/]+' | sed 's/^/  /'
    printf 'Subject Alternative Names:\n'
    printf '%s\n' "$cert" | grep -A1 'Subject Alternative Name' | tail -n1 \
        | tr ',' '\n' | sed 's/^[[:space:]]*/  /'
}

# ---------------------------------------------------------------------------
# opencode wrappers - shared bash/zsh core (REFACTOR-010)
# ---------------------------------------------------------------------------
# The portable pieces of the opencode quick-question + TUI wrappers live here,
# once, instead of being duplicated in .zsh/aliases.zsh and .bashrc. Each shell
# adds its own thin qq/qf/dbg on top: zsh needs `noglob` aliases (it errors on
# the unmatched `?` glob in `qq por que tardas?`); bash uses plain functions.

# _qq_call <model> <name> <prompt...>: one-shot opencode quick question. Each
# invocation is a fresh session; for follow-ups use `opencode run -c` or the TUI.
#   qq -> nan/qwen3.6           (default daily, multilingual, 262K ctx)
#   qf -> nan/deepseek-v4-flash (long-context 500K, large transforms)
_qq_call() {
    local model="$1" name="$2"; shift 2
    if [ ! -t 0 ]; then
        local stdin_data
        stdin_data=$(cat)
        if [ $# -gt 0 ]; then
            opencode run -m "$model" "$*"$'\n\n'"$stdin_data"
        else
            opencode run -m "$model" "$stdin_data"
        fi
    else
        [ $# -eq 0 ] && { printf 'usage: %s <consulta libre>\n' "$name" >&2; return 1; }
        opencode run -m "$model" "$*"
    fi
}

# oc / ocfull: opencode TUI dispatch. `--pure` bypasses MCPs + skills + plugins,
# avoiding the tool-resolution hang on complex queries (empirical 2026-05-25);
# `ocfull` is the opt-in full mode (MCP tool-use: Hive vault writes, etc.).
oc() { opencode --pure "$@"; }
ocfull() { opencode "$@"; }

# ---------------------------------------------------------------------------
# agyp: run a saved Gemini/AGY prompt. Shared bash/zsh (REFACTOR-010 style).
#
# Namespace history — this helper collided with oh-my-zsh's `git` plugin TWICE,
# both rooted in the same cause: zsh resolves aliases before execution or
# sourcing reaches function-definition syntax.
#   `gp`  -> shadowed by `alias gp='git push'` — the alias wins over the function
#            at the interactive prompt, so `gp` silently ran git push instead.
#   `gpr` -> shadowed by `alias gpr='git pull --rebase'` — worse, because when
#            oh-my-zsh loads before this file, `gpr() {` itself gets expanded to
#            `git pull --rebase () {` while functions.sh is being sourced: a
#            parse error that aborted the REST of this file, silently dropping
#            the utils.sh load below in every zsh session (bash was unaffected:
#            the git plugin is zsh-only).
#
# The `g*` namespace belongs to that plugin (~150 aliases) — do not re-enter it.
# `agyp` = the `agy` tool + "prompt", outside the minefield. Enforced by
# tests/shell-alias-collision.bats, which fails CI on any future collision.
#   agyp <prompt-name> [extra args]   -> ~/.gemini/prompts/<prompt-name>.md
agyp() {
    [ -z "${1:-}" ] && { printf 'usage: agyp <prompt-name> [args]\n' >&2; return 1; }
    local prompt_file="$HOME/.gemini/prompts/$1.md"
    shift
    if [ ! -f "$prompt_file" ]; then
        printf 'agyp: prompt not found at %s\n' "$prompt_file" >&2
        return 1
    fi
    agy -i "$(< "$prompt_file")"$'\n\n'"$*"
}

# Shared utility library (log_*, version_gte, deploy_file, …). Sourced from this
# single shared entrypoint so BOTH bash and zsh get it — replaces the old
# ~/.profile runtime mutation that appended `source utils.sh` to the user's rc
# files on every login (non-declarative; mutated deployed files).
if [ -f "$HOME/.dotfiles/scripts/utils.sh" ]; then
    # shellcheck source=/dev/null
    . "$HOME/.dotfiles/scripts/utils.sh"
fi
