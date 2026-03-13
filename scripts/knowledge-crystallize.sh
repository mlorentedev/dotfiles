#!/usr/bin/env bash
# knowledge-crystallize.sh: AI knowledge maintenance script
#
# Updates MEMORY.md dates, checks knowledge health, and prints
# a checklist of manual actions needed to keep the AI workflow healthy.
#
# Usage:
#   ./scripts/knowledge-crystallize.sh [--all | PROJECT_DIR]
#
# Idempotent: safe to run multiple times.
# Compatible: bash and zsh, POSIX-safe patterns.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

# Source shared utilities if available
if [ -f "$SCRIPT_DIR/utils.sh" ]; then
    # shellcheck source=utils.sh disable=SC1091
    . "$SCRIPT_DIR/utils.sh"
else
    # Minimal fallbacks when run standalone
    log_info()    { printf '[INFO]    %s\n' "$1"; }
    log_success() { printf '[SUCCESS] %s\n' "$1"; }
    log_warning() { printf '[WARNING] %s\n' "$1"; }
fi

usage() {
    cat <<'EOF'
knowledge-crystallize.sh — AI knowledge maintenance tool

USAGE
    ./scripts/knowledge-crystallize.sh [OPTIONS] [PROJECT_DIR]

DESCRIPTION
    Locates the MEMORY.md for PROJECT_DIR (defaults to CWD), then:
      1. Removes duplicate # currentDate entries (keeps latest)
      2. Updates # currentDate to today
      3. Stamps ## Last Crystallized: YYYY-MM-DD
      4. Warns if MEMORY.md exceeds 150 lines
      5. Prints a checklist of manual AI workflow actions

OPTIONS
    --all         Auto-discover and process ALL projects from ~/.claude/projects/
    -h, --help    Show this help message

EXAMPLES
    ./scripts/knowledge-crystallize.sh
    ./scripts/knowledge-crystallize.sh ~/Projects/kubelab
    ./scripts/knowledge-crystallize.sh --all
EOF
}

# Encode a path the same way Claude Code does:
# Replace all '/' with '-' (leading '/' becomes leading '-', kept as-is)
encode_path() {
    printf '%s' "$1" | tr '/' '-'
}

# Decode a Claude Code encoded path back to a filesystem path.
#
# Strategy:
#   1. Simple decode: replace all '-' with '/' and check if directory exists.
#      Works for paths where no directory component contains a literal '-'.
#   2. Filesystem scan: walk $HOME up to depth 5, encode each directory, and
#      match against the target. Handles paths with dashes in dir names.
#
# Returns the decoded path on stdout; returns 1 if no match found.
decode_path() {
    local encoded="$1"

    # Step 1: simple decode
    local simple
    simple=$(printf '%s' "$encoded" | tr '-' '/')
    if [ -d "$simple" ]; then
        printf '%s' "$simple"
        return 0
    fi

    # Step 2: filesystem scan under $HOME (handles dashes in directory names)
    local tmp_file result
    tmp_file=$(mktemp)
    find "$HOME/Projects" -maxdepth 5 -type d 2>/dev/null > "$tmp_file" || true

    result=""
    while IFS= read -r dir; do
        if [ "$(encode_path "$dir")" = "$encoded" ]; then
            result="$dir"
            break
        fi
    done < "$tmp_file"
    rm -f "$tmp_file"

    if [ -n "$result" ]; then
        printf '%s' "$result"
        return 0
    fi
    return 1
}

# Find the MEMORY.md for a given project directory
find_memory_file() {
    local project_dir="$1"
    local encoded memory_file
    encoded=$(encode_path "$project_dir")
    memory_file="$HOME/.claude/projects/$encoded/memory/MEMORY.md"
    [ -f "$memory_file" ] && printf '%s' "$memory_file" && return 0
    return 1
}

# Remove duplicate # currentDate lines, keep only the last one
dedup_current_date() {
    local file="$1"
    local count
    # Note: grep -c exits 1 on 0 matches (still prints "0"), so use || count=0
    # to overwrite, NOT "|| echo 0" which would append a second "0" to stdout.
    count=$(grep -c '^# currentDate' "$file" 2>/dev/null) || count=0
    [ "$count" -le 1 ] && return 0

    log_info "Removing $((count - 1)) duplicate # currentDate entries..."
    local tmp_file="${file}.tmp.$$"
    awk '
        /^# currentDate/ { skip=1; next }
        skip && /^Today.s date is / { skip=0; next }
        skip { skip=0 }
        { print }
    ' "$file" > "$tmp_file"
    mv "$tmp_file" "$file"
}

# Update the # currentDate section to today's date
update_current_date() {
    local file="$1" today="$2"

    if grep -q '^# currentDate' "$file" 2>/dev/null; then
        local tmp_file="${file}.tmp.$$"
        awk -v today="$today" '
            /^# currentDate/ { print; in_date=1; next }
            in_date && /^Today.s date is / { print "Today'\''s date is " today "."; in_date=0; next }
            { print }
        ' "$file" > "$tmp_file"
        mv "$tmp_file" "$file"
        log_info "Updated currentDate to $today"
    else
        printf '\n# currentDate\nToday'\''s date is %s.\n' "$today" >> "$file"
        log_info "Added currentDate section ($today)"
    fi
}

# Stamp or update the ## Last Crystallized date
stamp_last_crystallized() {
    local file="$1" today="$2"

    if grep -q '^## Last Crystallized:' "$file" 2>/dev/null; then
        local tmp_file="${file}.tmp.$$"
        awk -v today="$today" '
            /^## Last Crystallized:/ { print "## Last Crystallized: " today; next }
            { print }
        ' "$file" > "$tmp_file"
        mv "$tmp_file" "$file"
        log_info "Updated Last Crystallized to $today"
    elif grep -q '^# currentDate' "$file" 2>/dev/null; then
        local tmp_file="${file}.tmp.$$"
        awk -v today="$today" '
            /^# currentDate/ { print "## Last Crystallized: " today "\n"; print; next }
            { print }
        ' "$file" > "$tmp_file"
        mv "$tmp_file" "$file"
        log_info "Added Last Crystallized stamp ($today)"
    else
        printf '\n## Last Crystallized: %s\n' "$today" >> "$file"
        log_info "Added Last Crystallized stamp ($today)"
    fi
}

# Check line count and warn if over limit; return 1 if warning
check_line_count() {
    local file="$1" limit=150
    local count
    count=$(wc -l < "$file")
    if [ "$count" -gt "$limit" ]; then
        log_warning "MEMORY.md has $count lines (limit: $limit) — run /crystallize to trim"
        return 1
    else
        log_success "MEMORY.md line count: $count / $limit"
        return 0
    fi
}

# Print the manual action checklist (shown once at end of run)
print_checklist() {
    printf '\n=== Knowledge Crystallization Checklist ===\n\n'
    printf 'Manual steps (AI-assisted, run in Claude Code):\n'
    printf '  [ ] /insights  — audit observation backlog\n'
    printf '  [ ] /crystallize — promote observations to vault lessons\n'
    printf '  [ ] Check ~/Projects/knowledge/00_meta/patterns/ for new patterns\n'
    printf '  [ ] Update 11-tasks.md backlog progress bar\n\n'
    printf 'Automated (done by this script):\n'
    printf '  [x] currentDate updated\n'
    printf '  [x] Last Crystallized stamped\n'
    printf '  [x] Duplicate date entries removed\n\n'
}

# Process a single project: update MEMORY.md, check health
process_project() {
    local project_dir="$1" memory_file="$2" today="$3"

    dedup_current_date "$memory_file"
    update_current_date "$memory_file" "$today"
    stamp_last_crystallized "$memory_file" "$today"
    check_line_count "$memory_file" || true
    log_success "Updated: $memory_file"
}

# --all mode: auto-discover all projects from ~/.claude/projects/
run_all() {
    local today="$1"
    local projects_dir="$HOME/.claude/projects"
    local found=0 processed=0 skipped=0

    log_info "Discovering all projects in $projects_dir..."
    printf '\n'

    for memory_file in "$projects_dir"/*/memory/MEMORY.md; do
        # glob expands to literal string if no matches
        [ -f "$memory_file" ] || continue
        found=$((found + 1))

        local encoded_name project_dir
        encoded_name=$(basename "$(dirname "$(dirname "$memory_file")")")

        project_dir=$(decode_path "$encoded_name") || true

        if [ -n "$project_dir" ]; then
            log_info "[$encoded_name] → $project_dir"
            process_project "$project_dir" "$memory_file" "$today" || \
                log_warning "Failed to process $project_dir — skipping"
        else
            log_warning "[$encoded_name] → not found on disk (different machine or deleted — skipping)"
            skipped=$((skipped + 1))
        fi
        printf '\n'
    done

    if [ "$found" -eq 0 ]; then
        log_warning "No MEMORY.md files found in $projects_dir"
        log_warning "Open any project in Claude Code first to initialize its memory directory."
        exit 0
    fi

    processed=$((found - skipped))
    log_success "Processed $processed / $found projects ($skipped skipped)"
    print_checklist
}

# --- Main ---

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
fi

TODAY=$(date +%Y-%m-%d)

if [ "${1:-}" = "--all" ]; then
    log_info "Date: $TODAY"
    run_all "$TODAY"
    exit 0
fi

PROJECT_DIR="${1:-$(pwd)}"
PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd)"

log_info "Project: $PROJECT_DIR"
log_info "Date: $TODAY"

MEMORY_FILE=""
if ! MEMORY_FILE=$(find_memory_file "$PROJECT_DIR"); then
    log_warning "No MEMORY.md found for $PROJECT_DIR"
    log_warning "Expected: $HOME/.claude/projects/$(encode_path "$PROJECT_DIR")/memory/MEMORY.md"
    log_warning "Run Claude Code in this project first to initialize the memory directory."
    exit 0
fi

log_info "MEMORY.md: $MEMORY_FILE"
process_project "$PROJECT_DIR" "$MEMORY_FILE" "$TODAY"
print_checklist

exit 0
