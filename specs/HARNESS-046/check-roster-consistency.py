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


def definition_skills(path: str):
    """Frontmatter only; a regex on the skills line avoids a yaml dependency."""
    head = open(path, encoding="utf-8").read().split("---")[1]
    kind = re.search(r"^kind:\s*(\S+)", head, re.M)
    skills = re.search(r"^skills:\s*\[(.*?)\]", head, re.M | re.S)
    return (
        kind.group(1) if kind else "",
        [s.strip() for s in skills.group(1).split(",") if s.strip()] if skills else [],
    )


def main() -> int:
    vault = vault_path()
    roster = roster_rows(os.path.join(vault, "00_meta", "agents", "ROSTER.md"))
    defs_dir = os.path.join(vault, "00_meta", "agents", "definitions")
    errors = []

    for name in sorted(os.listdir(defs_dir)):
        agent_md = os.path.join(defs_dir, name, "AGENT.md")
        if not os.path.isfile(agent_md):
            continue
        kind, skills = definition_skills(agent_md)
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
