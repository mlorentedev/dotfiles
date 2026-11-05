#!/usr/bin/env bats
# The canonical bitácora workflows meet the strictest consumer's own checks (#3157).
#
# bitacora-rollout.sh copies these files verbatim into every product repository,
# and each one runs its own workflow checks on the PR. The 2026-11-04 rollout
# went red in two of seven: ts-bridge's check-workflow-permissions.sh wants a
# top-level `permissions:` key, and kubelab's yamllint caps lines at 130
# characters. A third followed: ts-bridge runs actionlint with no ignore list,
# so a shellcheck info note fails its job. Patching a consumer is undone by the next rollout, so the rules
# are held here, against the files the script actually ships.
#
# Only rules that fail a consumer's check are held. kubelab's yamllint also
# warns about one space before `# v9` after a SHA pin; that is the fleet's pin
# convention, which Dependabot writes, and a warning does not fail the job.

setup() {
    ROOT="$BATS_TEST_DIRNAME/.."
    # The shipped set, read from the script so a new canonical file is covered.
    read -ra WORKFLOWS <<<"$(sed -n 's/^WORKFLOWS=(\(.*\))$/\1/p' "$ROOT/scripts/bitacora-rollout.sh")"
}

@test "the rollout's workflow list is read" {
    [ "${#WORKFLOWS[@]}" -gt 0 ]
}

@test "every canonical workflow declares top-level permissions (ts-bridge hygiene)" {
    for wf in "${WORKFLOWS[@]}"; do
        run python3 -c '
import sys, yaml
doc = yaml.safe_load(open(sys.argv[1], encoding="utf-8"))
sys.exit(0 if "permissions" in doc else 3)' "$ROOT/.github/workflows/$wf"
        # 3 is the missing key; anything else is a broken file or environment.
        [ "$status" -eq 0 ] || { echo "$wf: status $status, no top-level permissions if 3: $output"; return 1; }
    done
}

@test "no canonical workflow line exceeds 130 characters (kubelab yamllint)" {
    for wf in "${WORKFLOWS[@]}"; do
        # Characters, not bytes: yamllint counts the accented letters and dashes as one.
        run python3 -c '
import sys
long = [n for n, line in enumerate(open(sys.argv[1], encoding="utf-8"), 1)
        if len(line.rstrip("\n")) > 130]
print(" ".join(map(str, long)))
sys.exit(1 if long else 0)' "$ROOT/.github/workflows/$wf"
        [ "$status" -eq 0 ] || { echo "$wf: lines over 130: $output"; return 1; }
    done
}

@test "every canonical workflow passes actionlint with no ignores (ts-bridge hygiene)" {
    # The Linux test job installs actionlint and sets DOTFILES_CI_EXPECT_ACTIONLINT,
    # so there a missing binary fails; elsewhere (the macOS job, a workstation
    # without it) the test says it skipped (#3160 item 7).
    if ! command -v actionlint >/dev/null; then
        [ -z "${DOTFILES_CI_EXPECT_ACTIONLINT:-}" ] || { echo "DOTFILES_CI_EXPECT_ACTIONLINT is set, actionlint is not on PATH"; return 1; }
        skip "actionlint not on PATH"
    fi
    for wf in "${WORKFLOWS[@]}"; do
        run actionlint -config-file /dev/null "$ROOT/.github/workflows/$wf"
        [ "$status" -eq 0 ] || { echo "$wf: $output"; return 1; }
    done
}

@test "CI installs the actionlint that dotf init's lint.yml pins" {
    # Two copies of one pin; this keeps them one version and one checksum.
    pin() { sed -n "s/^ *$1: *//p" "$2" | head -1; }
    tmpl="$ROOT/cli/internal/initrepo/templates/lint.yml"
    ci="$ROOT/.github/workflows/ci.yml"
    for key in ACTIONLINT_VERSION ACTIONLINT_SHA256; do
        [ -n "$(pin "$key" "$tmpl")" ]
        [ "$(pin "$key" "$ci")" = "$(pin "$key" "$tmpl")" ] || { echo "$key differs between ci.yml and lint.yml"; return 1; }
    done
}
