#!/usr/bin/env python3
"""HARNESS-046 f4 — the roster and the definitions must not drift apart.

This exists because they already had, in two directions at once, and neither was
visible from reading either file alone:

  * `architect` bundled two skills, breaking the roster's OWN stated rule
    ("every role bundles >=3 skills (no single-skill wrapper)") -- the design
    document contradicting itself, and the definition faithfully copying it.
  * the `curator` row omitted `dispose-proposals`, which its definition had
    carried and documented all along. The definition is what compiles, so the
    catalog was quietly wrong about what the deployed persona does.

Both were found by hand. This is the check that finds the next one.

Reads the roster and the definitions from the vault (the SSOT), never from the
generated records under harness/agents/ -- checking the generated copy against
the catalog would pass whenever the generator faithfully rendered a wrong
definition, which is the failure mode, not the guard.

Exit 0 when consistent, 1 with every divergence named.
"""

import os
import re
import subprocess
import sys

SKILLS_MIN = 3


def vault_path() -> str:
    """Resolve the vault the same way everything else does, and fail closed."""
    for candidate in (
        os.environ.get("VAULT_PATH"),
        _try(["dotf", "env", "path", "VAULT_PATH"]),
    ):
        if candidate and os.path.isdir(candidate):
            return candidate
    sys.exit("VAULT_PATH unresolved — set it or ensure `dotf env path VAULT_PATH` works")


def _try(cmd: list) -> str:
    try:
        return subprocess.run(cmd, capture_output=True, text=True, timeout=20).stdout.strip()
    except Exception:
        return ""


def roster_rows(path: str) -> dict:
    """Parse the invocable table. Autonomous roles are cataloged, not phase-bound."""
    rows = {}
    for line in open(path, encoding="utf-8"):
        m = re.match(r"\|\s*\*\*(\w[\w-]*)\*\*\s*\|\s*(\w+)\s*\|\s*([^|]+)\|", line)
        if m and m.group(2) != "Phase":
            rows[m.group(1)] = [s.strip() for s in m.group(3).split(",") if s.strip()]
    return rows


class UnreadableSkills(Exception):
    """A `skills:` key this guard can see but cannot parse."""


def definition_skills(path: str):
    """`kind:` by regex (a single scalar); the skill list by asking the CLI.

    THIS IS THE COMING-BACK THIS GUARD ASKED FOR. It used to match `skills:` with
    `^skills:\\s*\\[(.*?)\\]` and raise on anything else, because HARNESS-045 was
    going to introduce a mapping form carrying per-skill severity:

        skills:
          - id: audit
            enforce: warn

    and the ORIGINAL `... if skills else []` read that as "declares no skills",
    kept reporting exit 0, and would have compared an empty list against the
    roster -- the drift check disarmed by a schema change it never noticed
    (measured 2026-08-27). Raising was the placeholder; its own docstring said
    the real fix was to resolve the list "in `dotf harness resolve-skills`, where
    the parser already lives", rather than make this the third hand-rolled
    frontmatter reader in the repository. The reviewer migration landed, so it is
    now done: `LoadPersona` reads both forms on a real YAML parser, and this
    guard consumes its output.

    Delegation does not weaken the loud failure, it relocates it. The CLI exits
    non-zero on a `skills:` key that is present but unparseable and writes
    nothing to stdout, and every failure below -- non-zero exit, missing binary,
    timeout -- raises UnreadableSkills. Nothing here can degrade to []: an
    unreadable skill list and an empty one produce the same downstream behaviour
    and mean opposite things, which is the whole reason this class exists.
    """
    head = open(path, encoding="utf-8").read().split("---")[1]
    kind = re.search(r"^kind:\s*(\S+)", head, re.M)
    return kind.group(1) if kind else "", resolved_skills(path)


def resolved_skills(path: str) -> list:
    """Ask `dotf harness resolve-skills` for the ids, in either frontmatter form.

    A record declaring no `skills:` key at all prints nothing and exits 0, which
    is a genuine empty list -- the >= SKILLS_MIN check below then reports it,
    which is the correct complaint. Only a PRESENT-but-unreadable key is an
    exception, and the CLI, not this script, is what tells the two apart.

    THE STDOUT CONTRACT IS PINNED HERE: a YAML flow sequence, `[a, b, c]`, which
    is what `dotf harness resolve-skills` documents and emits. Anything else
    raises. Raised in review on #1412 as "fragile output parsing", with the
    prediction that a format change would make the drift check "pass silently".
    That half is measured false -- stubbing the CLI to emit a JSON array, bare
    newline-separated ids, arbitrary text, and nothing at all each produce exit 1,
    because a mangled list is still compared against a non-empty roster row and
    diverges from it. What was true is subtler and worth fixing: every one of
    those failures BLAMED THE DEFINITION, printing "skills diverge" against a
    vault file that was correct, which sends the reader to edit the wrong thing.
    A shape check turns four misdiagnoses into one accurate message.
    """
    try:
        run = subprocess.run(
            ["dotf", "harness", "resolve-skills", path],
            capture_output=True,
            text=True,
            timeout=20,
        )
    except Exception as exc:  # missing binary, timeout, anything else
        raise UnreadableSkills(
            f"{path}: could not run `dotf harness resolve-skills` ({exc}). "
            "Refusing to report an empty skill list -- that would compare 'no "
            "skills' against the roster and pass."
        ) from exc
    if run.returncode != 0:
        raise UnreadableSkills(
            f"{path}: `dotf harness resolve-skills` refused this record "
            f"(exit {run.returncode}): {run.stderr.strip() or '(no stderr)'}. "
            "If the message is about an unknown command, the `dotf` on PATH "
            "predates the subcommand -- rebuild it rather than parsing here."
        )
    out = run.stdout.strip()
    if not out:
        return []
    if not (out.startswith("[") and out.endswith("]")):
        raise UnreadableSkills(
            f"{path}: `dotf harness resolve-skills` printed {out[:120]!r}, which is "
            "not the documented `[a, b, c]` flow sequence. Reporting this as a "
            "skill list would blame the definition for a change in the CLI."
        )
    ids = [s.strip() for s in out[1:-1].split(",") if s.strip()]
    # A JSON array clears the bracket check above and yields '"audit"' -- the one
    # alternative format the review named by hand. Skill ids are slugs, so
    # anything carrying a quote, a space or a colon came from a different
    # serialisation, not from a record.
    bad = [i for i in ids if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", i)]
    if bad:
        raise UnreadableSkills(
            f"{path}: `dotf harness resolve-skills` returned {bad!r}, which are not "
            "skill ids. The brackets matched but the contents did not -- a JSON "
            "array reads this way. Blaming the definition for that would send the "
            "reader to the wrong file."
        )
    return ids


def main() -> int:
    vault = vault_path()
    roster = roster_rows(os.path.join(vault, "00_meta", "agents", "ROSTER.md"))
    defs_dir = os.path.join(vault, "00_meta", "agents", "definitions")
    errors = []

    for name in sorted(os.listdir(defs_dir)):
        agent_md = os.path.join(defs_dir, name, "AGENT.md")
        if not os.path.isfile(agent_md):
            continue
        try:
            kind, skills = definition_skills(agent_md)
        except UnreadableSkills as exc:
            errors.append(str(exc))
            continue
        if kind != "invocable":
            continue  # autonomous entries are cataloged separately, not in the phase table

        if name not in roster:
            errors.append(f"{name}: invocable definition has no ROSTER.md row")
            continue
        if roster[name] != skills:
            errors.append(
                f"{name}: skills diverge\n"
                f"    roster:     {roster[name]}\n"
                f"    definition: {skills}"
            )
        if len(skills) < SKILLS_MIN:
            errors.append(
                f"{name}: bundles {len(skills)} skill(s); the roster requires >= {SKILLS_MIN} "
                f"(no single-skill wrapper)"
            )

    for name in sorted(roster):
        if not os.path.isfile(os.path.join(defs_dir, name, "AGENT.md")):
            errors.append(f"{name}: ROSTER.md declares this role and no definition exists")

    if errors:
        print("ROSTER.md and the definitions have drifted:\n")
        for e in errors:
            print(f"  - {e}")
        return 1
    print(f"roster and definitions consistent ({len(roster)} invocable roles, all >= {SKILLS_MIN} skills)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
