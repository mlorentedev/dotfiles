#!/usr/bin/env bats
#
# Every token in a `dotf secrets run --only <list>` must be a live registry id or
# exposed env var.
#
# `--only` fails LOUD on an unknown token -- it refuses the whole launch rather
# than dropping the one name it cannot resolve. That is the right behaviour for a
# secrets facade, and it means a stale token in a wrapper is not a degradation but
# a total outage of whatever the wrapper launches.
#
# Measured 2026-08-25: the opencode wrapper in .bashrc and .zshrc named
# OLLAMA_API_KEY for a provider slot whose registry entry was never created. Both
# shells produced:
#
#     Error: --only: unknown id or env var "OLLAMA_API_KEY"
#
# so opencode -- the primary daily agent -- did not start at all, on either shell.
# `dotf doctor` reported 152 passed / 0 failed with its [OpenCode + pi] section
# green throughout: nothing checked that the wrapper's list and the registry still
# agreed.
#
# The deeper fault is duplication: the registry already declares which secrets
# each consumer takes (`consumers: [agent:opencode]`). Restating that list inside
# a shell function creates two sources of truth, and this is what their drift
# costs. Until the wrapper derives its scope from the registry, this test is what
# keeps them honest.

load 'lib/refute'

setup() {
    REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
}

@test "every --only token names a live registry id or exposed env var" {
    run python3 - "$REPO_ROOT" <<'PY'
import os, re, sys, glob

try:
    import yaml
except ImportError:
    print("SKIP: pyyaml unavailable")
    sys.exit(0)

root = sys.argv[1]
reg = yaml.safe_load(open(os.path.join(root, "secrets", "registry.yaml")))

known = set()
for s in reg.get("secrets", []) or []:
    known.add(s.get("id"))
    exp = s.get("expose") or {}
    env = exp.get("env")
    if isinstance(env, str):
        known.add(env)
    elif isinstance(env, list):
        known.update(v for v in env if isinstance(v, str))
    elif isinstance(env, dict):
        known.update(env.keys())
    fil = exp.get("file")
    if isinstance(fil, dict) and fil.get("var"):
        known.add(fil["var"])
known.discard(None)

# The surfaces a human launch actually goes through. Specs and archived docs are
# prose about past states and are deliberately not scanned.
candidates = [
    ".bashrc", ".zshrc", "powershell/profile.ps1",
    *glob.glob(os.path.join(root, ".zsh", "*.zsh")),
    *glob.glob(os.path.join(root, "scripts", "*.sh")),
    *glob.glob(os.path.join(root, "systemd", "**", "*.conf"), recursive=True),
]

# A token may legitimately be a shell expansion rather than a literal name
# ("$VAR", "${VAR}"): those resolve at runtime and this static check cannot judge
# them. Skip them explicitly rather than failing on something unknowable.
offenders = []
checked = 0
for rel in candidates:
    path = rel if os.path.isabs(rel) else os.path.join(root, rel)
    if not os.path.isfile(path):
        continue
    for lineno, line in enumerate(open(path, encoding="utf-8", errors="replace"), 1):
        # `secrets run` must appear on the same line, and the line must not be a
        # comment. A bare `--only` search matched prose: the hive drop-in's own
        # doc comment explains "a --only token matching an id", and the scan
        # dutifully reported `token` as an unknown secret. Asking for the text
        # instead of the invocation is the same mistake this guard exists to catch.
        if re.match(r"\s*(#|//)", line):
            continue
        if not re.search(r"secrets\s+run\b", line):
            continue
        for m in re.finditer(r"--only[=\s]+([A-Za-z0-9_,\$\{\}]+)", line):
            for tok in m.group(1).split(","):
                tok = tok.strip()
                if not tok or "$" in tok:
                    continue
                checked += 1
                if tok not in known:
                    offenders.append(
                        f"{os.path.relpath(path, root)}:{lineno}: {tok}"
                    )

if offenders:
    print("UNKNOWN --only TOKENS (not in secrets/registry.yaml):")
    for o in offenders:
        print("  " + o)
    sys.exit(1)

# A check that verifies nothing must say so. If the scan matched no tokens at all
# the wrappers were probably restructured, and a silent pass would be a false
# all-clear -- the exact failure this suite has been bitten by before.
if checked == 0:
    print("NO --only TOKENS FOUND — the scan matched nothing; has the wrapper shape changed?")
    sys.exit(1)

print(f"OK: {checked} --only token(s) all resolve in the registry")
PY
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

# The specific name that caused the outage, asserted so a revert is loud.
#
# Comment lines are excluded on purpose: the removal is DOCUMENTED at each site
# it was removed from, and a naive whole-file refute fails on the explanation of
# the very thing it is checking. What must not come back is the executable
# reference, not the memory of it.
@test "no executable line names OLLAMA_API_KEY (the provider is gone)" {
    local f
    for f in .bashrc .zshrc ai/opencode/opencode.jsonc; do
        run bash -c "grep -vE '^[[:space:]]*(#|//)' '$REPO_ROOT/$f' | grep -n 'OLLAMA_API_KEY' || true"
        [ -z "$output" ] || { echo "$f still names OLLAMA_API_KEY in code: $output"; false; }
    done
}
