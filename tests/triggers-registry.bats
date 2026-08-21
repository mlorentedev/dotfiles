#!/usr/bin/env bats
# HARNESS: harness/triggers.json had a Go loader and zero guards (#1137) — the
# only registry in this repo with neither a test nor a CI reference.
#
# Why the duplication it guards cannot simply be deleted, measured 2026-08-21:
#
#   1. `cmd/harness.go` calls `LoadTriggers("")` — empty repoRoot, so the
#      resolution order is walk-up-from-cwd, then the embedded copy. In any repo
#      that is not this checkout the walk-up finds nothing, so the EMBEDDED copy
#      is the only source. Verified from /tmp with no checkout present:
#      `dotf harness suggest .github/workflows/ci.yml` returned real triggers.
#      That is C12 working as designed — user-level deploy, every repo inherits.
#
#   2. The copies cannot be collapsed into one. The Go module root is `cli/`, and
#      `//go:embed ../../harness/triggers.json` is rejected at compile time with
#      "invalid pattern syntax" — embed cannot reference paths outside its
#      package directory.
#
# So two copies are structurally required, and what was missing is the assertion
# that they agree. Identical today only means nobody has edited one alone yet;
# the moment they diverge, the binary ships a registry the repo does not declare
# and nothing says so.
#
# Deliberately NOT here: a JSON Schema or a `dotf doctor` check. ADR-035 gives
# harness/model-map.json both and records that this sets a precedent the other
# four registries do not meet, and that whether they follow is a separate
# decision. This closes the guard gap, not that decision.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    REPO_COPY="$REPO/harness/triggers.json"
    EMBED_COPY="$REPO/cli/internal/harness/triggers.json"
}

@test "triggers: the repo copy exists and is valid JSON" {
    [ -f "$REPO_COPY" ]
    run python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$REPO_COPY"
    [ "$status" -eq 0 ]
}

@test "triggers: the embedded copy exists and is valid JSON" {
    # It is what every repo outside this checkout actually reads.
    [ -f "$EMBED_COPY" ]
    run python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$EMBED_COPY"
    [ "$status" -eq 0 ]
}

@test "triggers: the embedded copy is byte-identical to the repo copy" {
    # The load-bearing assertion. A diverged embed means `dotf harness suggest`
    # answers from a registry this repo does not declare, in every repo except
    # this one — and silently, because both files parse and both look right.
    run diff -q "$REPO_COPY" "$EMBED_COPY"
    [ "$status" -eq 0 ]
}

@test "triggers: every rule carries the fields the loader reads" {
    # TriggerRule declares id/pattern/globs plus optional skills/keywords. A rule
    # missing id or globs loads without error and then matches nothing, which is
    # indistinguishable from a path the repo genuinely has no trigger for.
    run python3 -c '
import json, sys
d = json.load(open(sys.argv[1]))
bad = []
for i, t in enumerate(d.get("triggers", [])):
    if not t.get("id"):
        bad.append(f"triggers[{i}] has no id")
    if not t.get("globs"):
        bad.append(f'"'"'triggers[{i}] ({t.get("id","?")}) has no globs'"'"')
if bad:
    print("\n".join(bad)); sys.exit(1)
' "$REPO_COPY"
    [ "$status" -eq 0 ]
}

@test "triggers: the file declares a version, per the engine-config idiom" {
    # triggers.json and manifest.json carry `version`; reviewer-pool.json and
    # review-attestation.json carry `$comment` instead. ADR-035 names the two
    # idioms. This pins which one this file belongs to so a later edit does not
    # quietly drop the field the loader unmarshals.
    run python3 -c '
import json, sys
d = json.load(open(sys.argv[1]))
sys.exit(0 if isinstance(d.get("version"), int) else 1)
' "$REPO_COPY"
    [ "$status" -eq 0 ]
}
