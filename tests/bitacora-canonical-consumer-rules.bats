#!/usr/bin/env bats
# The canonical bitácora workflows meet the strictest consumer's own checks (#3157).
#
# bitacora-rollout.sh copies these files verbatim into every product repository,
# and each one runs its own workflow checks on the PR. The 2026-11-04 rollout
# went red in two of seven: ts-bridge's check-workflow-permissions.sh wants a
# top-level `permissions:` key, and kubelab's yamllint caps lines at 130
# characters. Patching a consumer is undone by the next rollout, so the rules
# are held here, against the files the script actually ships.

setup() {
    ROOT="$BATS_TEST_DIRNAME/.."
    # The shipped set, read from the script so a new canonical file is covered.
    WORKFLOWS=$(sed -n 's/^WORKFLOWS=(\(.*\))$/\1/p' "$ROOT/scripts/bitacora-rollout.sh")
}

@test "the rollout's workflow list is read" {
    [ -n "$WORKFLOWS" ]
}

@test "every canonical workflow declares top-level permissions (ts-bridge hygiene)" {
    for wf in $WORKFLOWS; do
        run python3 -c '
import sys, yaml
doc = yaml.safe_load(open(sys.argv[1], encoding="utf-8"))
sys.exit(0 if "permissions" in doc else 1)' "$ROOT/.github/workflows/$wf"
        [ "$status" -eq 0 ] || { echo "$wf: no top-level permissions"; return 1; }
    done
}

@test "no canonical workflow line exceeds 130 characters (kubelab yamllint)" {
    for wf in $WORKFLOWS; do
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
