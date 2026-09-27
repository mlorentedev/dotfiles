#!/usr/bin/env bats
# GUARD (#1533 follow-up): a spec number names one piece of work.
#
# WHAT THIS CAUGHT
# ----------------
# specs/HARNESS-111 (issue #1241) and specs/archive/HARNESS-111-review-base-and-
# reality (issue #1533) shared a number: two sessions took the next free
# HARNESS-NNN from the same tree, neither saw the other's unpushed folder, and
# main accepted both because nothing asserted across them. The same shape as the
# lesson-number collisions in guard-lesson-numbers-unique.bats, one directory up.
# Measured on main at 0c18d26 before this guard existed: three such pairs
# reachable from an active or bare-ID folder (HARNESS-027, HARNESS-041,
# HARNESS-111), all renumbered by the change that added this file, and six more
# pairs living entirely in specs/archive/ (AI-022, AI-023, CLI-025, DOCS-013,
# MEMORY-001, REFACTOR-012), left as reviewed history under #1563.
#
# WHY THE ACTIVE SIDE IS WHERE THE ASSERTION LIVES
# ------------------------------------------------
# A spec is born active and archived later, so every future collision enters
# through specs/<id>/. Asserting there catches it at creation; asserting over
# the whole archive would go red on main today for six historical pairs and
# stay red until someone renumbers reviewed material, which is how a guard
# becomes a baseline file.
#
# WHAT IS DELIBERATELY ALLOWED
# ----------------------------
# The same tree carries eighteen CLI-024-secrets-* folders tracking different
# issues. That is a series: one ticket family split into sub-specs that share
# the ticket ID AND the first slug word. Forbidding it would renumber months of
# archived, reviewed material to satisfy a rule nobody had written down. So the
# assertion is: folders that share a ticket ID must all carry a slug and must
# agree on the slug's first word. A bare ID (specs/HARNESS-111) claims the whole
# number; a slugged sibling beside it is the collision.
#
# Runs against the real tree, active and archived alike, for the same reason
# the lessons guard does: the defect is a property of the actual directory.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    SPECS_DIR="$DOTFILES_DIR/specs"
    # The ticket-ID part of the spec idPattern (cli/internal/spec/spec.go): an
    # AREA of hyphen-joined upper-case segments, each optionally ending in
    # digits, then the number. Narrower than that, and APP-CONFIG-015 or
    # ADR028-004 folders are never listed, so they can collide unseen (#1479).
    # The sub-id letter stays outside it: SDD-012b belongs to SDD-012's number.
    TICKET_RE='[A-Z]+[0-9]*(-[A-Z]+[0-9]*)*-[0-9]+'
}

# Every spec folder name, active and archived, one per line.
_spec_dirs() {
    ls "$SPECS_DIR" "$SPECS_DIR/archive" 2>/dev/null | grep -E "^${TICKET_RE}" || true
}

# The ticket ID a folder name carries.
_ticket_id() {
    printf '%s' "$1" | grep -oE "^${TICKET_RE}"
}

_explain() {
    printf 'Renumber the NEWER one to the next free <AREA>-NNN across specs/ and\n'
    printf 'specs/archive/, update its frontmatter id, headings and features.json\n'
    printf 'ids, and grep the tree for the old ID before renaming, not after.\n'
}

@test "guard: a bare-ID spec folder owns its number outright" {
    [ -d "$SPECS_DIR" ] || skip "specs/ not present"

    bad=""
    for id in $(_spec_dirs | grep -xE "${TICKET_RE}"); do
        siblings="$(_spec_dirs | grep -E "^${id}-" || true)"
        [ -n "$siblings" ] || continue
        bad="$bad$id plus $(printf '%s' "$siblings" | tr '\n' ' ')"$'\n'
    done

    if [ -n "$bad" ]; then
        printf 'Bare spec folders with a slugged sibling on the same number:\n%s\n' "$bad"
        _explain
    fi

    [ -z "$bad" ]
}

@test "guard: no active spec shares its ticket ID with anything but its own series" {
    [ -d "$SPECS_DIR" ] || skip "specs/ not present"

    bad=""
    for active in $(ls "$SPECS_DIR" | grep -E "^${TICKET_RE}"); do
        id="$(_ticket_id "$active")"
        members="$(_spec_dirs | grep -E "^${id}(-|\$)" | sort -u)"
        [ "$(printf '%s\n' "$members" | wc -l)" -gt 1 ] || continue
        # Series: every member slugged, all agreeing on the first slug word.
        stems="$(printf '%s\n' "$members" | sed -E "s/^${id}\$/(bare)/; s/^${id}-([^-]+).*/\1/" | sort -u)"
        if [ "$(printf '%s\n' "$stems" | wc -l)" -ne 1 ]; then
            bad="$bad$id: $(printf '%s' "$members" | tr '\n' ' ')"$'\n'
        fi
    done

    if [ -n "$bad" ]; then
        printf 'Active specs sharing a ticket ID with unrelated work:\n%s\n' "$bad"
        _explain
    fi

    [ -z "$bad" ]
}

@test "guard: the series exception really is exercised (CLI-024-secrets-*)" {
    [ -d "$SPECS_DIR/archive" ] || skip "specs/archive/ not present"
    # If this ever drops to one, the exception above is untested and the first
    # assertion is only ever running its collision branch.
    n="$(_spec_dirs | grep -cE '^CLI-024-secrets-')"
    [ "$n" -ge 2 ]
}

@test "guard: the ticket-ID grammar sees every AREA shape the spec CLI accepts" {
    # A folder the guard cannot parse is a folder it never checks.
    [ "$(_ticket_id APP-CONFIG-015-webhook-auth)" = "APP-CONFIG-015" ]
    [ "$(_ticket_id ADR028-004-slug)" = "ADR028-004" ]
    [ "$(_ticket_id CI-GATE-2)" = "CI-GATE-2" ]
    [ "$(_ticket_id SDD-012b-guard)" = "SDD-012" ]
    [ "$(_ticket_id HARNESS-111)" = "HARNESS-111" ]
}
