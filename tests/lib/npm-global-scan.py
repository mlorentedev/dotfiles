#!/usr/bin/env python3
"""Assert no setup twin globally npm-installs a tool packages.json owns (ADR-036).

A tool whose packages.json source is `npm` is converged by `dotf tools install`;
a setup twin that also runs `npm install -g` on it installs it twice, from two
pins. OPS-042's guard grepped for the two spellings it had deleted, so a plain
`npm install -g yarn@1.22.22` passed it (#1864).

This matches the property instead. For each global npm install command in a twin:

- continuation lines are joined first (a trailing backslash in sh, a trailing
  backtick in PowerShell), so a package on the next line is still seen;
- the verb (`install`, `i`, `add`) may sit anywhere after `npm`, so
  `npm -g install yarn` counts, and the command ends at a shell operator;
- the value of a flag that takes one (`--prefix "$HOME/.local"`) is not a
  package;
- a `$var` argument is resolved through every assignment to it in the same file,
  `export`/`local`/`readonly` forms and PowerShell's `$x = if (..) { "a" } else
  { "b" }` included: every string literal on the right-hand side is a candidate.
  A variable with no assignment in the file FAILS the check rather than passing
  it, since the guard cannot say what it installs.

Comment lines are skipped. Usage:

    python3 tests/lib/npm-global-scan.py <packages.json> <twin>...

Exit 0 with a count of the global installs checked, or 1 naming every offender.
Exit 1 also when no global install with a package argument was found, so the
scan cannot pass by matching nothing.
"""

import json
import re
import sys

VERBS = {"install", "i", "add", "in"}
GLOBAL_FLAGS = {"-g", "--global", "--global=true", "--location=global"}
# Flags whose next token is their value, not a package.
VALUE_FLAGS = {"--prefix", "--registry", "--cache", "--userconfig",
               "--globalconfig", "--tag", "--location", "--workspace", "-w"}
OPERATORS = {"||", "&&", "|", ";", "&"}

ASSIGN_RE = re.compile(
    r"(?:^|[\s;{(])(?:export\s+|local\s+|readonly\s+|declare\s+(?:-\w+\s+)?)?"
    r"\$?([A-Za-z_]\w*)\s*=(?!=)\s*(.*)$")
LITERAL_RE = re.compile(r"\"([^\"]*)\"|'([^']*)'")
VAR_RE = re.compile(r"^\$\{?([A-Za-z_]\w*)\}?(.*)$")


def logical_lines(text):
    """Yield (first line number, joined line), joining continuations."""
    buf, start = [], None
    for n, line in enumerate(text.splitlines(), 1):
        stripped = line.rstrip()
        if start is None:
            start = n
        if stripped.endswith("\\") or stripped.endswith("`"):
            buf.append(stripped[:-1])
            continue
        buf.append(line)
        yield start, " ".join(buf)
        buf, start = [], None
    if buf:
        yield start, " ".join(buf)


def assignments(lines):
    """Map each lower-cased variable name to every value assigned to it."""
    values = {}
    for _, line in lines:
        if line.lstrip().startswith("#"):
            continue
        m = ASSIGN_RE.search(line)
        if not m:
            continue
        rhs = m.group(2)
        cands = [a or b for a, b in LITERAL_RE.findall(rhs)]
        bare = rhs.split()[0] if rhs.split() else ""
        if bare and bare[0] not in "\"'":
            cands.append(bare.rstrip(";"))
        values.setdefault(m.group(1).lower(), []).extend(cands)
    return values


def package(arg):
    """The package name of an npm argument, without its version."""
    if arg.startswith("@"):
        scope, _, rest = arg[1:].partition("/")
        return "@" + scope + "/" + rest.split("@", 1)[0]
    return arg.split("@", 1)[0]


def command_tokens(line):
    """The tokens of the npm command on line, from `npm` to its end, or None."""
    tokens = line.split()
    for i, tok in enumerate(tokens):
        if tok.strip("\"'&") in ("npm", "npm.cmd") or tok.endswith("/npm"):
            out = []
            for t in tokens[i + 1:]:
                if t in OPERATORS or re.match(r"^\d?>", t):
                    break
                ends = t.endswith(";")
                out.append(t.rstrip(";").strip("\"'()"))
                if ends:
                    break
            return out
    return None


def package_args(tokens):
    """The package arguments of a global install, or None if it is not one."""
    verb = next((i for i, t in enumerate(tokens) if t in VERBS), None)
    if verb is None or not any(t in GLOBAL_FLAGS for t in tokens):
        return None
    args, skip = [], False
    for t in tokens[verb + 1:]:
        if skip:
            skip = False
            continue
        if t in VALUE_FLAGS:
            skip = True
            continue
        if t.startswith("-") or not t:
            continue
        args.append(t)
    return args


def candidates(arg, assigned):
    """What arg may install; None when it is a variable with no assignment."""
    m = VAR_RE.match(arg)
    if not m:
        return [arg]
    values = assigned.get(m.group(1).lower())
    if not values:
        return None
    return [v + m.group(2) for v in values]


def scan(owned, path):
    lines = list(logical_lines(open(path, encoding="utf-8").read()))
    assigned = assignments(lines)
    bad, seen, name = [], 0, path.rsplit("/", 1)[-1]
    for n, line in lines:
        if line.lstrip().startswith("#"):
            continue
        tokens = command_tokens(line)
        args = package_args(tokens) if tokens is not None else None
        if not args:
            continue
        seen += 1
        for arg in args:
            cands = candidates(arg, assigned)
            if cands is None:
                bad.append(f"{name}:{n}: {arg} has no assignment in the file, so what it installs is unknown")
                continue
            for c in cands:
                pkg = package(c.strip("\"'"))
                if pkg in owned:
                    bad.append(f"{name}:{n}: {pkg}")
    return bad, seen


def main(argv):
    owned = {t["source"]["package"] for t in json.load(open(argv[1]))["tools"]
             if t["source"]["type"] == "npm"}
    if not owned:
        print("packages.json owns no npm tool, so the check would be vacuous")
        return 1
    bad, seen = [], 0
    for path in argv[2:]:
        b, s = scan(owned, path)
        bad += b
        seen += s
    if not seen:
        print("no global npm install with a package argument in any twin, so the scan matched nothing real")
        return 1
    if bad:
        print("\n".join(bad))
        return 1
    print(f"{seen} global npm install(s) checked, none of a catalog-owned tool")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
