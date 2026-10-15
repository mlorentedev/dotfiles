#!/usr/bin/env python3
"""Fail when two bats tests make the same static assertion.

Usage: check-duplicate-assertions.py <repo-root>

Exit 0 clean, 1 duplicates or a malformed marker, 2 usage error.

WHAT COUNTS AS ONE ASSERTION
----------------------------
A static assertion is a `grep`, `refute_grep` or `refute_grep_fixed` line in an
@test body whose target resolves to a file that exists in the repository. Its
identity is (claim, pattern, target): the claim is "present", or "absent" for a
refute_* call or a `!`-negated grep, so asserting a line and refuting it are two
claims. Flags are not part of it: `grep -q` and `grep -qE` on one pattern ask the
same question of the same file.

A grep whose output feeds a pipe selects text; the assertion is the whole
pipeline. `grep -B1 X f | grep -qx Y` asserts that Y precedes X, so its identity
is the pipeline, flags included, with the target resolved: the same pipeline in
two tests is a duplicate, `grep -A10 X f | grep -q Z` is another claim.

Two tests holding the same identity is a duplicate. A test repeating a line of
its own is not: that is one test, and only one test fails when the line goes.

What is deliberately NOT an assertion here:
- A grep whose target is a file the test writes ($BATS_TEST_TMPDIR, a $log, a
  per-test fixture). Two tests reading the same pattern from what two different
  runs produced check two runs, not one line twice.
- `run grep ...` lines: their claim is in the status checks after them.
- A grep captured in `$(...)`: it extracts a value for an assertion that
  follows, and two tests may extract the same value to check different things.
- A grep in an `if`/`elif`/`while` condition, and `grep -v`: a filter, not a claim.
- A target that does not resolve without running the test.

These are the audit's terms (2026-10-07 CI test duplication audit, class 3):
counted this way the repo held 63 repeated assertions, the audit's number.

A line that must repeat another test's assertion says why on the line:
    grep -q 'x' "$SCRIPT"  # dup-ok: <reason>
A marker without a reason is an error, so an exemption is always explained.
"""

import os
import re
import shlex
import sys
from collections import defaultdict

TEST_RE = re.compile(r'^@test\s+(["\'])(.*)\1\s*\{\s*$')
ASSIGN_RE = re.compile(r'^\s*(?:export\s+|local\s+)?([A-Za-z_][A-Za-z0-9_]*)=(.*)$')
GREP_RE = re.compile(r'(?:^|[\s;&(])(!\s+)?(grep|refute_grep_fixed|refute_grep)\s+(.*)$')
DUP_OK_RE = re.compile(r'#\s*dup-ok:(.*)$')
CD_PWD_RE = re.compile(r'^\$\(\s*cd\s+"?([^"]*?)"?\s*(?:>\s*/dev/null\s*(?:2>&1\s*)?)?&&\s*pwd(?:\s+-P)?\s*\)$')
VAR_RE = re.compile(r'\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)')
# grep options that take a value; the value is not a positional argument.
VALUED = {'-e', '-f', '-m', '-A', '-B', '-C', '--include', '--exclude'}
# Tokens that end the grep command. `|` ends it too, but starts a pipeline.
END = {'||', '&&', ';', ')', ']', ']]', '}', 'then', 'do'}
REDIRECT = {'>', '>>', '<', '>&', '&>', '<&'}


def unquote(value):
    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in '"\'':
        return value[1:-1]
    return value


def assigned_value(line):
    """(name, value) when line assigns a variable, else None."""
    m = ASSIGN_RE.match(line)
    if not m:
        return None
    return m.group(1), unquote(m.group(2).split(' #')[0])


def file_vars(lines):
    """First assignment of each variable outside @test bodies, as written.

    That is the scope every test inherits; a test's own assignments override
    it for the lines after them (scan).
    """
    found, in_test = {}, False
    for line in lines:
        if TEST_RE.match(line):
            in_test = True
            continue
        if in_test:
            in_test = not line.startswith('}')
            continue
        a = assigned_value(line)
        if a is not None and a[0] not in found:
            found[a[0]] = a[1]
    return found


def resolve(path, assigned, tests_dir, depth=0):
    """Expand a target path; None when it depends on anything but the layout."""
    if depth > 5:
        return None
    m = CD_PWD_RE.match(path)
    if m:
        path = m.group(1)
    path = path.replace('$(dirname "$BATS_TEST_FILENAME")', '$BATS_TEST_DIRNAME')

    unresolved = []

    def sub(match):
        name = match.group(1) or match.group(2)
        if name == 'BATS_TEST_DIRNAME':
            return tests_dir
        if name in assigned:
            value = resolve(assigned[name], assigned, tests_dir, depth + 1)
            if value is not None:
                return value
        unresolved.append(name)
        return ''

    path = VAR_RE.sub(sub, path)
    if unresolved or '$' in path or '`' in path:
        return None
    return os.path.normpath(path)


def tokens_of(rest):
    lexer = shlex.shlex(rest, posix=True, punctuation_chars=';&|<>()')
    lexer.whitespace_split = True
    return list(lexer)


def grep_args(rest):
    """(flags, pattern, [targets], downstream) from the text after `grep`.

    downstream is the rest of the pipeline the grep feeds, '' when it feeds
    none. None when the line is not a grep this check can read.
    """
    try:
        tokens = tokens_of(rest)
    except ValueError:
        return None
    flags, pattern, positional, downstream = [], None, [], []
    options_done = False
    i = 0
    while i < len(tokens):
        tok = tokens[i]
        if tok in END:
            break
        if tok == '|':
            for later in tokens[i + 1:]:
                if later in END:
                    break
                downstream.append(later)
            break
        if tok in REDIRECT or (tok.isdigit() and i + 1 < len(tokens) and tokens[i + 1] in REDIRECT):
            i += 2 if tok in REDIRECT else 3
            continue
        if options_done:
            positional.append(tok)
            i += 1
            continue
        if tok == '--':
            options_done = True
            i += 1
            continue
        if tok == '-e' and i + 1 < len(tokens):
            pattern = tokens[i + 1]
            i += 2
            continue
        if tok in VALUED and i + 1 < len(tokens):
            flags.append(tok + tokens[i + 1])
            i += 2
            continue
        if tok.startswith('-') and len(tok) > 1:
            flags.append(tok)
        else:
            positional.append(tok)
        i += 1
    if pattern is None:
        if len(positional) < 2:
            return None
        pattern, positional = positional[0], positional[1:]
    if not positional:
        return None
    return flags, pattern, positional, ' '.join(downstream)


def scan(root):
    tests_dir = os.path.join(root, 'tests')
    index = defaultdict(list)
    errors = []
    scanned = 0
    files = sorted(f for f in os.listdir(tests_dir) if f.endswith('.bats'))
    for name in files:
        rel_file = 'tests/' + name
        with open(os.path.join(tests_dir, name), encoding='utf-8') as fh:
            lines = fh.read().split('\n')
        inherited = file_vars(lines)
        test = None
        for lineno, line in enumerate(lines, 1):
            m = TEST_RE.match(line)
            if m:
                test = m.group(2)
                assigned = dict(inherited)
                continue
            if test is None:
                continue
            if line.startswith('}'):
                test = None
                continue
            a = assigned_value(line)
            if a is not None:
                assigned[a[0]] = a[1]
                continue
            stripped = line.strip()
            if not stripped or stripped.startswith('#') or stripped.startswith(('run ', 'if ', 'elif ', 'while ')):
                continue
            g = GREP_RE.search(line)
            if not g or line[:g.start(2)].rstrip().endswith('$('):
                continue
            args = grep_args(g.group(3))
            if args is None:
                continue
            flags, pattern, targets, downstream = args
            if any(f.startswith('-') and not f.startswith('--') and 'v' in f for f in flags):
                continue
            negated = bool(g.group(1)) or g.group(2).startswith('refute') or stripped.startswith('!')
            for target in targets:
                path = resolve(target, assigned, tests_dir)
                if path is None or not os.path.isfile(path):
                    continue
                rel = os.path.relpath(path, root)
                if rel.startswith('..'):
                    continue
                scanned += 1
                ok = DUP_OK_RE.search(line)
                if ok:
                    if not ok.group(1).strip():
                        errors.append(f'{rel_file}:{lineno}: a dup-ok marker needs a reason after "dup-ok:"')
                    continue
                claim = 'absent' if negated else 'present'
                if downstream:
                    # The pipeline is the claim: its flags select the text the
                    # next stage reads, so they are part of it here.
                    pattern = f"grep {' '.join(sorted(flags))} {pattern!r} | {downstream}"
                index[(claim, pattern, rel)].append((rel_file, lineno, test))
    return index, errors, scanned, len(files)


def main(argv):
    if len(argv) != 2:
        print(__doc__.strip().split('\n\n')[1], file=sys.stderr)
        return 2
    root = os.path.abspath(argv[1])
    if not os.path.isdir(os.path.join(root, 'tests')):
        print(f'check-duplicate-assertions: no tests/ directory under {root}', file=sys.stderr)
        return 2
    index, errors, scanned, nfiles = scan(root)
    print(f'scanned {scanned} static assertions in {nfiles} bats files')
    duplicates = []
    for (claim, pattern, target), hits in sorted(index.items()):
        if len({(f, t) for f, _, t in hits}) > 1:
            duplicates.append((claim, pattern, target, hits))
    for claim, pattern, target, hits in duplicates:
        shown = pattern if pattern.startswith('grep ') else repr(pattern)
        print(f'\n{target}: {shown} asserted {claim} by {len(hits)} tests:')
        for f, lineno, test in hits:
            print(f'  {f}:{lineno}  @test "{test}"')
    for err in errors:
        print(err)
    if duplicates:
        print(f'\n{len(duplicates)} assertion(s) made by more than one test. Keep it in the '
              'test that owns the behaviour and delete the others; a deliberate repeat says '
              'why with "# dup-ok: <reason>" on the line.')
    return 1 if duplicates or errors else 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
