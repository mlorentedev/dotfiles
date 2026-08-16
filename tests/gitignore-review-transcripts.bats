#!/usr/bin/env bats
# #998: the ignore rule for adversarial-review transcripts was written
# `specs/*/review-transcript.jsonl`. A `*` does not cross a `/`, so the rule
# stopped matching the moment `dotf spec archive` renamed a spec from
# `specs/<id>/` into `specs/archive/<id>/` -- which is precisely when the file
# would be committed. HARNESS-072's transcript was 552 MB, five times GitHub's
# 100 MB hard limit, so the archive commit could not have been pushed at all.
#
# The rule is now `specs/**/`. This is the guard that keeps it there: the
# expiry-at-rename is invisible in review (both globs look right), so it needs
# an assertion at the DESTINATION path, not only the origin.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

# `git check-ignore` matches on the pathname, so these paths need not exist --
# which is the point: the archived path is the one nobody creates until the
# archive runs, and by then the commit is already being built.
_is_ignored() {
    git -C "$REPO" check-ignore -q "$1"
}

@test "review transcripts are ignored in an ACTIVE spec folder" {
    _is_ignored "specs/AI-999-probe/review-transcript.jsonl"
    _is_ignored "specs/AI-999-probe/review-transcript.jsonl.stderr"
}

@test "review transcripts are ignored in an ARCHIVED spec folder (#998)" {
    # The regression this file exists for: `specs/*/` matched the first case
    # above and not this one, and the difference only shows up at archive time.
    _is_ignored "specs/archive/AI-999-probe/review-transcript.jsonl"
    _is_ignored "specs/archive/AI-999-probe/review-transcript.jsonl.stderr"
}

@test "review transcripts are ignored in an ABANDONED spec folder (#998)" {
    # `spec archive --abandoned` nests one level deeper still.
    _is_ignored "specs/archive/_abandoned/AI-999-probe/review-transcript.jsonl"
}

@test "the ignore rule does not swallow authored spec artifacts" {
    # Keeps the fix from being "widened" into `specs/**` outright. Every file a
    # human or an agent writes into a spec folder must stay tracked -- an
    # over-broad rule here would silently stop versioning the spec contract,
    # which is a far worse failure than the one being fixed.
    local name
    for name in proposal.md tasks.md verification.md features.json review.md; do
        if _is_ignored "specs/AI-999-probe/$name"; then
            printf 'specs/AI-999-probe/%s is ignored, but it is an authored artifact\n' "$name" >&2
            return 1
        fi
        if _is_ignored "specs/archive/AI-999-probe/$name"; then
            printf 'specs/archive/AI-999-probe/%s is ignored, but it is an authored artifact\n' "$name" >&2
            return 1
        fi
    done
}

@test "the transcript filename this guard asserts is the one the CLI writes" {
    # A drift guard on the guard: if TranscriptFile is ever renamed, every
    # assertion above keeps passing while protecting a filename nothing
    # produces. Pin the literal to its single source of truth in the Go code.
    local src="$REPO/cli/internal/spec/review_launch.go"
    [ -f "$src" ] || {
        printf '%s is gone -- update this test to the new SSOT for the transcript name\n' "$src" >&2
        return 1
    }
    grep -q 'TranscriptFile = "review-transcript.jsonl"' "$src" || {
        printf 'TranscriptFile in %s no longer matches the name .gitignore ignores.\n' "$src" >&2
        printf 'Update .gitignore and this test together.\n' >&2
        return 1
    }
}
