#!/bin/bash

# bitacora-rollout.sh: Roll the bitácora board plumbing out to every repo (OPS-002, #258).
#
# Decision (2026-06-09): per-repo Action, NOT the built-in project Auto-add — git-native
# (IaC: the workflow lives in each repo, no UI snowflake config), uniform across N repos,
# portable via dotf init, and the add-to-project trigger covers PRs (OPS-003, #266).
#
# Per repo (idempotent — mutates only on diff, 2nd run reports 0 changes):
#   1. link the repo to GitHub Project #1 (the bitácora)
#   2. upload the BITACORA_PAT secret (decrypted from sensitive/, or $BITACORA_PAT)
#   3. deploy .github/workflows/add-to-project.yml + bitacora-status.yml
#      (canonical copies = THIS repo's .github/workflows/; deployed via the contents API)
#   4. backfill: put every open issue and PR on the board (item-add is idempotent)
#
# Usage: ./scripts/bitacora-rollout.sh [--check] [repo ...]
#        (no repo args = every non-archived, non-fork repo of $OWNER)
# Exit:  0 ok, 1 on any failed step

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

OWNER="${BITACORA_OWNER:-mlorentedev}"
PROJECT_NUMBER="${BITACORA_PROJECT:-1}"
WORKFLOWS=(add-to-project.yml bitacora-status.yml)
SECRET_FILE="${BITACORA_SECRET_FILE:-$REPO_ROOT/sensitive/github.bitacora.secret.age}"
AGE_KEY="${AGE_KEY_FILE:-$HOME/.config/age/key.txt}"

CHECK=0
REPOS=()
for arg in "$@"; do
    case "$arg" in
        --check) CHECK=1 ;;
        --help|-h) sed -n '3,20p' "$0"; exit 0 ;;
        *) REPOS+=("$arg") ;;
    esac
done

CHANGED=0
FAILED=0
chg() { echo "📝 $1"; CHANGED=$((CHANGED + 1)); }
ok()  { echo "✅ $1"; }
err() { echo "❌ $1"; FAILED=1; }
run() { if [ "$CHECK" = 1 ]; then echo "   (dry-run) $*"; else "$@"; fi; }

# ── Repo discovery: every non-archived, non-fork repo (forks never run our CI) ──
if [ "${#REPOS[@]}" -eq 0 ]; then
    mapfile -t REPOS < <(gh repo list "$OWNER" --limit 100 \
        --json name,isArchived,isFork \
        -q '.[] | select(.isArchived == false and .isFork == false) | .name' | sort)
fi
echo "=== bitácora rollout$( [ "$CHECK" = 1 ] && echo ' (--check, read-only)') — ${#REPOS[@]} repo(s) ==="

# ── Secret: explicit env wins; otherwise decrypt the age-encrypted PAT ──
if [ -z "${BITACORA_PAT:-}" ]; then
    BITACORA_PAT="$(age --decrypt -i "$AGE_KEY" "$SECRET_FILE" 2>/dev/null | tr -d '[:space:]')" \
        || { err "cannot decrypt $SECRET_FILE (set BITACORA_PAT or fix the age key)"; exit 1; }
fi

# ── Already-linked repos (one GraphQL call, not one per repo) ──
LINKED="$(gh api graphql -f query="query { user(login: \"$OWNER\") { projectV2(number: $PROJECT_NUMBER) { repositories(first: 100) { nodes { name } } } } }" \
    -q '.data.user.projectV2.repositories.nodes[].name' 2>/dev/null || true)"

PR_FILES=()   # workflows that hit branch protection on the current repo (reset per repo)

deploy_workflow() {  # $1 = repo, $2 = workflow filename
    local repo="$1" wf="$2" path=".github/workflows/$2" src remote sha
    src="$REPO_ROOT/.github/workflows/$wf"
    [ -f "$src" ] || { err "$repo: canonical $wf missing in this checkout"; return; }
    remote="$(gh api "repos/$OWNER/$repo/contents/$path" -q '.content' 2>/dev/null | base64 -d 2>/dev/null || true)"
    if [ "$remote" = "$(cat "$src")" ]; then
        ok "$repo: $wf current"
        return
    fi
    sha="$(gh api "repos/$OWNER/$repo/contents/$path" -q '.sha' 2>/dev/null || true)"
    chg "$repo: deploy $wf$( [ -n "$sha" ] && echo ' (update)')"
    if [ "$CHECK" = 0 ]; then
        local -a put_args=(-X PUT "repos/$OWNER/$repo/contents/$path"
            -f message="ci: deploy bitácora workflow $wf (OPS-002)"
            -f content="$(base64 -w0 < "$src")")
        [ -n "$sha" ] && put_args+=(-f sha="$sha")
        if ! gh api "${put_args[@]}" >/dev/null 2>&1; then
            # HTTP 409 on protected branches ("changes must be made through a pull request")
            # — queue the file for the per-repo PR fallback below.
            PR_FILES+=("$wf")
        fi
    fi
}

deploy_via_pr() {  # $1 = repo; deploys ${PR_FILES[@]} on a branch + opens an auto-merge PR
    local repo="$1" branch="ci/bitacora-workflows" base base_sha wf path src sha
    base="$(gh api "repos/$OWNER/$repo" -q '.default_branch')"
    base_sha="$(gh api "repos/$OWNER/$repo/git/ref/heads/$base" -q '.object.sha')"
    # Create the branch at base HEAD; if it already exists, FORCE-reset it to base.
    # Blind reuse caused merge conflicts: after the previous PR was squash-merged the
    # leftover branch held stale commits that textually conflicted with the new base
    # (observed on pollex/yt-metrics-cli/pdf-modifier-mcp, 2026-06-10). Re-PUTting the
    # canonical files on a base-fresh branch is the convergent form.
    if ! gh api -X POST "repos/$OWNER/$repo/git/refs" \
        -f ref="refs/heads/$branch" -f sha="$base_sha" >/dev/null 2>&1; then
        gh api -X PATCH "repos/$OWNER/$repo/git/refs/heads/$branch" \
            -f sha="$base_sha" -F force=true >/dev/null \
            || { err "$repo: cannot reset $branch to $base HEAD"; return; }
    fi
    for wf in "${PR_FILES[@]}"; do
        path=".github/workflows/$wf"
        src="$REPO_ROOT/.github/workflows/$wf"
        sha="$(gh api "repos/$OWNER/$repo/contents/$path?ref=$branch" -q '.sha' 2>/dev/null || true)"
        local -a put_args=(-X PUT "repos/$OWNER/$repo/contents/$path"
            -f message="ci: deploy bitácora workflow $wf (OPS-002)"
            -f content="$(base64 -w0 < "$src")" -f branch="$branch")
        [ -n "$sha" ] && put_args+=(-f sha="$sha")
        gh api "${put_args[@]}" >/dev/null || { err "$repo: PUT $wf to $branch failed"; return; }
    done
    if ! gh pr list --repo "$OWNER/$repo" --head "$branch" --state open --json number -q '.[].number' | grep -q .; then
        gh pr create --repo "$OWNER/$repo" --base "$base" --head "$branch" \
            --title "ci: deploy bitácora workflows (OPS-002)" \
            --body "Canonical \`add-to-project.yml\` + \`bitacora-status.yml\` from \`mlorentedev/dotfiles\` (runbook §7). Deployed via \`scripts/bitacora-rollout.sh\` — this repo's default branch is protected, so the rollout lands through a PR." >/dev/null \
            || { err "$repo: pr create failed"; return; }
    fi
    # Merge is a supervised human action: auto-merge is forbidden repo-wide
    # (it lands a PR the instant CI is green, bypassing human review), so the
    # rollout only opens the PR and leaves the merge to a reviewer.
    ok "$repo: protected branch — PR opened for review (${PR_FILES[*]})"
}

for repo in "${REPOS[@]}"; do
    echo
    echo "── $repo ──"

    # 1. Link to the project board
    if echo "$LINKED" | grep -qx "$repo"; then
        ok "$repo: already linked to project #$PROJECT_NUMBER"
    else
        chg "$repo: link to project #$PROJECT_NUMBER"
        run gh project link "$PROJECT_NUMBER" --owner "$OWNER" --repo "$OWNER/$repo" >/dev/null \
            || err "$repo: project link failed"
    fi

    # 2. Secret (set is idempotent — encrypted server-side, no diff possible: always set)
    if [ "$CHECK" = 1 ]; then
        echo "   (dry-run) gh secret set BITACORA_PAT --repo $OWNER/$repo"
    else
        # Feed the token via stdin, not --body: an argv value is world-readable
        # in /proc/<pid>/cmdline for the call's duration. printf (no trailing
        # newline) keeps the PAT byte-exact.
        if printf '%s' "$BITACORA_PAT" | gh secret set BITACORA_PAT --repo "$OWNER/$repo" 2>/dev/null; then
            ok "$repo: BITACORA_PAT set"
        else
            err "$repo: secret set failed"
        fi
    fi

    # 3. Workflows (direct push; PR fallback when the default branch is protected)
    PR_FILES=()
    for wf in "${WORKFLOWS[@]}"; do
        deploy_workflow "$repo" "$wf"
    done
    if [ "${#PR_FILES[@]}" -gt 0 ]; then
        deploy_via_pr "$repo"
    fi

    # 4. Backfill open issues + PRs (item-add returns the existing item when already on board)
    backfilled=0
    while IFS= read -r url; do
        [ -z "$url" ] && continue
        if run gh project item-add "$PROJECT_NUMBER" --owner "$OWNER" --url "$url" >/dev/null; then
            backfilled=$((backfilled + 1))
        else
            err "$repo: item-add failed for $url"
        fi
    done < <(
        gh issue list --repo "$OWNER/$repo" --state open --limit 200 --json url -q '.[].url' 2>/dev/null
        gh pr list    --repo "$OWNER/$repo" --state open --limit 200 --json url -q '.[].url' 2>/dev/null
    )
    ok "$repo: backfill — $backfilled open item(s) ensured on board"
done

echo
echo "── summary ──"
if [ "$FAILED" = 1 ]; then
    echo "❌ rollout finished with errors — review above"
    exit 1
elif [ "$CHANGED" = 0 ]; then
    ok "all repos already rolled out (0 changes)"
else
    echo "📝 $CHANGED change(s)$( [ "$CHECK" = 1 ] && echo ' pending (dry-run)' || echo ' applied')"
fi
