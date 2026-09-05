#!/usr/bin/env python3
"""AC8 — prove the CI-003 assertions fail against a mutated tree.

A test that passes on first write is not evidence, so each assertion added by
CI-003 is mutation-tested: break the guarantee, require the test to go red.

WHAT MAKES THIS DIFFERENT FROM THE HARNESS THAT LIED (lesson 267)
-----------------------------------------------------------------
"The mutation was not caught" and "the mutation was not applied" are
indistinguishable from outside a harness -- both yield a green suite and the
words NOT CAUGHT -- and they demand opposite responses. The previous harness
did `source.replace(entry, '', 1)`, deleted an identically-spelled line from a
DIFFERENT block, and reported four sound assertions as vacuous.

So this one refuses to reach a verdict it cannot support:

  * every mutation is located by line position at or after the RECONCILE
    BLOCK's anchor, never by a repo-wide string search;
  * the mutated line is printed, before and after, so the diff is a fact rather
    than a hypothesis;
  * a mutation that produces an identical file, or a `-f` filter that selects no
    test, is reported as a HARNESS FAULT and never as a result.

Run from anywhere in the working tree:  python3 specs/CI-003/mutate-assertions.py
Exit 0 iff every mutation was caught.
"""
import pathlib
import re
import subprocess
import sys

REPO = pathlib.Path(subprocess.run(
    ["git", "rev-parse", "--show-toplevel"],
    capture_output=True, text=True, check=True).stdout.strip())
SH = REPO / "setup-linux.sh"
PS = REPO / "setup-windows.ps1"
SUITE = REPO / "tests" / "pi-packages.bats"
BATS = pathlib.Path.home() / ".local" / "bin" / "bats"

# The same anchors tests/pi-packages.bats slices on. If these drift, the tests
# and this harness stop talking about the same region -- which is the failure
# being defended against, so they are asserted rather than assumed.
BLOCK_START = {
    SH: re.compile(r"^PI_PACKAGES_SRC="),
    PS: re.compile(r"^\$piPackagesSrc = Join-Path"),
}


def block_offset(path):
    lines = path.read_text().splitlines()
    for i, line in enumerate(lines):
        if BLOCK_START[path].match(line):
            return i, lines
    raise SystemExit(f"HARNESS FAULT: block anchor not found in {path.name}")


def mutate(path, find, repl, drop=False):
    """First match AT OR AFTER the block anchor. Never the whole file."""
    off, lines = block_offset(path)
    for i in range(off, len(lines)):
        if find in lines[i]:
            before = lines[i]
            if drop:
                return lines[:i] + lines[i + 1:], i + 1, before, "<line deleted>"
            after = lines[i].replace(find, repl)
            return lines[:i] + [after] + lines[i + 1:], i + 1, before, after
    raise SystemExit(
        f"HARNESS FAULT: target not found in {path.name} at/after line {off + 1}: {find!r}")


CASES = [
    ("AC1", "neither twin discards", SH,
     dict(find='install "$pi_pkg" 2>&1)', repl='install "$pi_pkg" >/dev/null 2>&1)')),
    ("AC1", "neither twin discards", PS,
     dict(find='2>&1 | Out-String)', repl='2>$null | Out-String)')),
    ("AC2", "output is CAPTURED", SH,
     dict(find='if pi_pkg_out=$("$PI_BIN"', repl='if $("$PI_BIN"')),
    ("AC3", "elapsed time, on EVERY outcome", SH,
     dict(find='installed in ${pi_pkg_elapsed}s', repl=None, drop=True)),
    ("AC3", "elapsed time, on EVERY outcome", PS,
     dict(find='installed in ${piElapsed}s', repl=None, drop=True)),
    ("AC4", "emits what happened", SH,
     dict(find='(exit $pi_pkg_rc) — output follows',
          repl='— run \\"$PI_BIN install $pi_pkg\\" to see why')),
    ("AC5", "FENCED", SH,
     dict(find='"--- end pi install $pi_pkg ---"', repl=None, drop=True)),
    ("AC5", "FENCED", PS,
     dict(find='"--- end pi install $piPkgName ---"', repl=None, drop=True)),
    ("AC7", "VERBOSITY only", SH,
     dict(find="printf '%s\\n' \"--- pi install $pi_pkg",
          repl="continue  # mutated\n                    printf '%s\\n' \"--- pi install $pi_pkg")),
    ("AC6", "terminated by a noisy install", PS,
     dict(find='} finally {', repl='}; if ($false) {')),
    ("AC9", "outcome is unknown counts as FAILED", PS,
     dict(find='                $piRc = 1', repl=None, drop=True)),
    # The default's VALUE is the guarantee, not its presence: defaulted to 0, an
    # install whose outcome is unknown is counted as a success, which is the exact
    # outcome the line exists to prevent.
    ("AC9", "outcome is unknown counts as FAILED", PS,
     dict(find='                $piRc = 1', repl='                $piRc = 0')),
]


def run_case(name, path, spec):
    original = path.read_text()
    new, lineno, before, after = mutate(
        path, spec["find"], spec["repl"], drop=spec.get("drop", False))
    body = "\n".join(new) + "\n"
    if body == original:
        print("    HARNESS FAULT: the mutation produced an identical file")
        return None
    path.write_text(body)
    try:
        print(f"    @ {path.name}:{lineno}")
        print(f"      -{before.strip()[:96]}")
        print(f"      +{after.strip()[:96]}")
        r = subprocess.run([str(BATS), "-f", name, str(SUITE)],
                           capture_output=True, text=True, cwd=REPO)
        selected = sum(1 for line in r.stdout.splitlines()
                       if line.startswith(("ok ", "not ok ")))
        if selected == 0:
            print("    HARNESS FAULT: -f matched no test; nothing was exercised")
            return None
        caught = r.returncode != 0
        print(f"      {selected} test(s) selected -> "
              f"{'CAUGHT' if caught else '!!! NOT CAUGHT !!!'}")
        return caught
    finally:
        path.write_text(original)


def main():
    results = []
    for ac, name, path, spec in CASES:
        print(f"\n== {ac}  {name}  [{path.name}]")
        results.append((ac, name, path.name, run_case(name, path, spec)))
    print("\n" + "=" * 68)
    for ac, name, fname, caught in results:
        verdict = "CAUGHT " if caught is True else (
            "MISSED " if caught is False else "FAULT  ")
        print(f"{verdict} {ac}  {name}  [{fname}]")
    print("=" * 68)
    sys.exit(0 if all(r[3] is True for r in results) else 1)


main()
