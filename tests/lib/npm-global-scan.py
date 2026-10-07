#!/usr/bin/env python3
"""Assert no setup twin globally npm-installs a tool packages.json owns (ADR-036).

A tool whose packages.json source is `npm` is converged by `dotf tools install`;
a setup twin that also runs `npm install -g` on it installs it twice, from two
pins. OPS-042's guard grepped for the two spellings it had deleted, so a plain
`npm install -g yarn@1.22.22` passed it (#1864).

This matches the property instead. For each global npm install command in a twin:

- continuation lines are joined first (a trailing backslash in sh, a trailing
  backtick in PowerShell), so a package on the next line is still seen;
- every `npm` on a line starts a command, including one after `&&` and one
  opening a command substitution (`X=$(npm install -g yarn)`);
- the verb (`install`, `i`, `add`) may sit anywhere after `npm`, so
  `npm -g install yarn` counts, and the command ends at a shell operator;
- the value of a flag that takes one (`--prefix "$HOME/.local"`) is not a
  package, and `--location global` is as global as `-g`;
- a `$var` argument is resolved through every assignment to it in the same file,
  `export`/`local`/`readonly` forms and PowerShell's `$x = if (..) { "a" } else
  { "b" }` included: every string literal on the right-hand side is a candidate,
  and a value that is itself a variable is followed in turn. An argument that
  cannot be followed (no assignment in the file, `$1`, `$env:X`, a command
  substitution, a `${VAR:-x}`-style expansion) FAILS the check rather than
  passing it, since the guard cannot say what it installs;
- an expansion inside an argument (`yarn${SUFFIX}`) may be empty, so the
  argument is also compared with every expansion removed;
- variable names are case-sensitive in sh and case-insensitive in PowerShell,
  so only a `.ps1` file's names are folded.

Comment lines and trailing comments are skipped. Usage:

    python3 tests/lib/npm-global-scan.py <packages.json> <twin>...

Exit 0 with a count of the global installs checked, or 1 naming every offender.
A twin with no global npm install at all passes: that is ADR-036's end state.
That the scan sees the spellings it claims to is pinned by the fixture test in
tests/setup-linux.bats, not by requiring the twins to carry an install.
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
# `$VAR` or `${VAR}`, plus whatever follows it. Any other `${...}` form is an
# expansion the scan does not evaluate, so it does not match and fails closed.
VAR_RE = re.compile(r"^\$(?:\{([A-Za-z_]\w*)\}|([A-Za-z_]\w*))(.*)$")


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


def assignments(lines, key):
    """Map each variable name, through key, to every value assigned to it."""
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
        values.setdefault(key(m.group(1)), []).extend(cands)
    return values


# An expansion inside an argument, `${...}` (one level of nesting) or `$VAR`.
EMBEDDED_RE = re.compile(r"\$\{[^{}]*\}|\$[A-Za-z_]\w*")


def package(arg):
    """The package name of an npm argument, without its version."""
    if arg.startswith("@"):
        scope, _, rest = arg[1:].partition("/")
        return "@" + scope + "/" + rest.split("@", 1)[0]
    return arg.split("@", 1)[0]


NPM_RE = re.compile(r"(?:^|[=(`&])npm(?:\.cmd)?$")


def npm_commands(line):
    """The tokens of every npm command on line, each from `npm` to its end.

    A line can hold several (`npm dedupe && npm install -g x`), and npm can
    open a command substitution (`X=$(npm install -g x)`), so every token that
    ends in `npm` starts one.
    """
    tokens, out, i = line.split(), [], 0
    while i < len(tokens):
        if not (NPM_RE.search(tokens[i].strip("\"'")) or tokens[i].endswith("/npm")):
            i += 1
            continue
        cmd, i = [], i + 1
        while i < len(tokens):
            t = tokens[i]
            if t in OPERATORS or t.startswith("#") or re.match(r"^\d?>", t):
                break
            i += 1
            ends = t.endswith(";")
            cmd.append(t.rstrip(";").strip("\"'()"))
            if ends:
                break
        out.append(cmd)
    return out


def package_args(tokens):
    """The package arguments of a global install, or None if it is not one."""
    verb = next((i for i, t in enumerate(tokens) if t in VERBS), None)
    is_global = any(t in GLOBAL_FLAGS or (t == "--location" and nxt == "global")
                    for t, nxt in zip(tokens, tokens[1:] + [""]))
    if verb is None or not is_global:
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


def candidates(arg, assigned, key, depth=0):
    """What arg may install, or None when that cannot be known.

    A variable resolves through its assignments, and a value that is itself a
    variable resolves in turn. Unknown is anything that cannot be followed: a
    variable with no assignment in the file, a positional or environment
    parameter (`$1`, `$env:X`), a command substitution, or a chain too deep.
    """
    if arg.startswith("$(") or "`" in arg:
        return None
    m = VAR_RE.match(arg)
    if not m:
        return None if arg.startswith("$") else [arg]
    values = assigned.get(key(m.group(1) or m.group(2)))
    if not values or depth >= 8:
        return None
    out = []
    for v in values:
        resolved = candidates(v.strip("\"'") + m.group(3), assigned, key, depth + 1)
        if resolved is None:
            return None
        out += resolved
    return out


def scan(owned, path):
    lines = list(logical_lines(open(path, encoding="utf-8").read()))
    # PowerShell names are case-insensitive; sh names are not.
    key = str.lower if path.lower().endswith(".ps1") else str
    assigned = assignments(lines, key)
    bad, seen, name = [], 0, path.rsplit("/", 1)[-1]
    for n, line in lines:
        if line.lstrip().startswith("#"):
            continue
        for tokens in npm_commands(line):
            args = package_args(tokens)
            if not args:
                continue
            seen += 1
            for arg in args:
                cands = candidates(arg, assigned, key)
                if cands is None:
                    bad.append(f"{name}:{n}: what {arg} installs cannot be resolved from the file")
                    continue
                for c in cands:
                    c = c.strip("\"'")
                    for pkg in {package(c), package(EMBEDDED_RE.sub("", c))}:
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
    if bad:
        print("\n".join(bad))
        return 1
    print(f"{seen} global npm install(s) checked, none of a catalog-owned tool")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
