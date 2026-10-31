#!/usr/bin/env python3
"""Assert every `dotf secrets run --only` token resolves in secrets/registry.yaml.

`--only` fails LOUD on an unknown token -- it refuses the whole launch rather than
dropping the one name it cannot resolve. Right for a secrets facade, and it makes
a stale token in a wrapper a total outage of whatever that wrapper launches, not a
degradation.

Measured 2026-08-25: the opencode wrapper named OLLAMA_API_KEY for a provider slot
whose registry entry was never created, so `opencode` did not start on either
shell while `dotf doctor` reported 152 passed / 0 failed.

Lives here rather than inline in the bats file so the scan is independently
runnable and the test stays a short assertion. Usage:

    python3 tests/lib/only-scope-scan.py <repo-root>

Exit 0 with a count, or 1 naming every offender.
"""

import glob
import os
import re
import sys

# Missing PyYAML must FAIL, never skip. The first draft exited 0 on ImportError,
# which made this guard pass without reading a single token -- the same false
# all-clear the NO TOKENS check below exists to prevent, introduced ten lines
# above it. A guard that cannot run has not passed.
try:
    import yaml
except ImportError:  # pragma: no cover - environment defect, not a code path
    sys.exit(
        "FAIL: PyYAML is unavailable, so the registry could not be read.\n"
        "      This check cannot run, which is not the same as passing.\n"
        "      Install it (pip install --user pyyaml) or fix the CI image."
    )

# A token may be a shell expansion ("$VAR", "${VAR}") that resolves at runtime;
# this static scan cannot judge those and skips them explicitly.
EXPANSION = "$"

# The one `--only` argument that is not a list: the profile's helper forwarding
# the names its callers pass with -Names, which are read at the call site.
FORWARD = "($Names"

# Prefix for an `--only` scope the scan cannot read; never a registry name.
UNREADABLE = "unreadable --only scope: "

# The surfaces a human launch actually goes through. Specs and archived docs are
# prose about past states and are deliberately not scanned.
SCAN_GLOBS = (".zsh/*.zsh", "scripts/*.sh", "systemd/**/*.conf")
SCAN_FILES = (".bashrc", ".zshrc", "powershell/profile.ps1")


def registry_names(root):
    """Every id and exposed variable name the registry declares."""
    doc = yaml.safe_load(open(os.path.join(root, "secrets", "registry.yaml")))
    names = set()
    for secret in doc.get("secrets", []) or []:
        names.add(secret.get("id"))
        exposed = secret.get("expose") or {}
        env = exposed.get("env")
        if isinstance(env, str):
            names.add(env)
        elif isinstance(env, list):
            names.update(v for v in env if isinstance(v, str))
        elif isinstance(env, dict):
            names.update(env.keys())
        as_file = exposed.get("file")
        if isinstance(as_file, dict) and as_file.get("var"):
            names.add(as_file["var"])
    names.discard(None)
    return names


def candidate_files(root):
    for rel in SCAN_FILES:
        path = os.path.join(root, rel)
        if os.path.isfile(path):
            yield path
    for pattern in SCAN_GLOBS:
        yield from (
            p
            for p in glob.glob(os.path.join(root, pattern), recursive=True)
            if os.path.isfile(p)
        )


def tokens_in(path):
    """(lineno, token) for real invocations only.

    `secrets run` must be on the same line and the line must not be a comment. A
    bare `--only` search matched PROSE: the hive drop-in's doc comment explains
    "a --only token matching an id", and the scan reported `token` as an unknown
    secret. Asking for the text instead of the invocation is the mistake this
    guard exists to catch.
    """
    forward_line, call_site_tokens = 0, 0
    with open(path, encoding="utf-8", errors="replace") as handle:
        for lineno, line in enumerate(handle, 1):
            if re.match(r"\s*(#|//)", line):
                continue
            # The PowerShell wrappers name their scope at the call site and the
            # helper forwards it (#2909); the list read here is that call site's.
            # A call whose list is not on its own line (a continuation, a splat)
            # is reported, not skipped.
            if re.search(r"Invoke-WithSessionSecrets\b", line) and not re.search(r"\bfunction\s+Invoke-WithSessionSecrets\b", line):
                names = re.search(r"-Names\s+([A-Za-z0-9_,]+)(\s|$)", line)
                if not names:
                    yield lineno, UNREADABLE + "an Invoke-WithSessionSecrets call without a -Names list on its line"
                    continue
                for token in names.group(1).split(","):
                    if token.strip():
                        call_site_tokens += 1
                        yield lineno, token.strip()
            if not re.search(r"secrets\s+run\b", line):
                continue
            for match in re.finditer(r"--only[=\s]+(\S+)", line):
                if match.group(1) == FORWARD:
                    forward_line = forward_line or lineno
                    continue
                if not re.fullmatch(r"[A-Za-z0-9_,${}]+", match.group(1)):
                    # An argument this scan cannot read must not pass as checked:
                    # the forward above was invisible to it until #2909.
                    yield lineno, UNREADABLE + match.group(1)
                    continue
                for token in match.group(1).split(","):
                    token = token.strip()
                    if token and EXPANSION not in token:
                        yield lineno, token
    # A forward is skipped because its call sites are read instead; with none
    # read, the file would contribute nothing and the scan would still say OK.
    if forward_line and not call_site_tokens:
        yield forward_line, UNREADABLE + "a forwarded --only list with no call site read in this file"


def main():
    root = sys.argv[1] if len(sys.argv) > 1 else "."
    known = registry_names(root)
    offenders, checked = [], 0

    for path in candidate_files(root):
        for lineno, token in tokens_in(path):
            checked += 1
            if token not in known:
                offenders.append(f"{os.path.relpath(path, root)}:{lineno}: {token}")

    if offenders:
        print("FAIL: --only tokens absent from secrets/registry.yaml:")
        for offender in offenders:
            print("  " + offender)
        return 1

    # A check that verified nothing must say so. A silent zero would read as
    # health while proving nothing -- if the wrappers were restructured, this
    # scan should be rewritten, not quietly pass.
    if checked == 0:
        print("FAIL: no --only tokens matched; has the wrapper shape changed?")
        return 1

    print(f"OK: {checked} --only token(s) all resolve in the registry")
    return 0


if __name__ == "__main__":
    sys.exit(main())
