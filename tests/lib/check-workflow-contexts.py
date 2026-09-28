#!/usr/bin/env python3
"""Check the status-check names that workflows report (CI-004, #1739).

1. No two jobs across pull-request workflows report the same name. Two
   reporters of one name make a required check ambiguous: `cli.yml`'s Go lint
   and `ci.yml`'s shell lint both reported `lint`, so which one gated a merge
   depended on which finished last.
2. Every required context in forge/branch-protection.json for this repository
   is reported by exactly one job that always reports on a pull request. A
   required check that never reports stays pending and blocks the merge; that
   is what a `paths:` filter on the workflow, or a job-level skip of a matrix
   job (GitHub then reports the unexpanded name), would do.

Prints one line per problem and exits 1 if there is any.
"""
import itertools
import json
import pathlib
import re
import sys

import yaml

REPO_SLUG = "mlorentedev/dotfiles"
MATRIX_REF = re.compile(r"\$\{\{\s*matrix\.([A-Za-z0-9_-]+)\s*\}\}")
# A commit status a workflow posts through the API, e.g. review-attestation.yml's
# `gh api .../statuses/... -f context="review-attestation"`.
STATUS_CONTEXT = re.compile(r"""-f\s+context=["']?([A-Za-z0-9_.-]+)""")


def triggers(doc):
    # PyYAML reads the bare key `on` as the boolean True.
    return doc.get("on", doc.get(True)) or {}


def pr_trigger(doc):
    """The pull_request trigger's config, {} if it has none, None if absent."""
    on = triggers(doc)
    if isinstance(on, str):
        return {} if on == "pull_request" else None
    if isinstance(on, list):
        return {} if "pull_request" in on else None
    if "pull_request" not in on:
        return None
    return on["pull_request"] or {}


def display_names(job_id, job):
    """Every name the job reports, a matrix name expanded per value."""
    name = str(job.get("name", job_id))
    refs = MATRIX_REF.findall(name)
    if not refs:
        return [name]
    matrix = (job.get("strategy") or {}).get("matrix") or {}
    values = [matrix.get(r) for r in refs]
    if not all(isinstance(v, list) for v in values):
        return [name]  # not statically expandable: report it as written
    out = []
    for combo in itertools.product(*values):
        n = name
        for ref, val in zip(refs, combo):
            n = MATRIX_REF.sub(lambda m, r=ref, v=val: str(v) if m.group(1) == r else m.group(0), n)
        out.append(n)
    return out


def main(root):
    root = pathlib.Path(root)
    problems = []
    reporters = {}  # name -> [(workflow, job_id, always_reports)]
    for wf in sorted((root / ".github" / "workflows").glob("*.yml")):
        text = wf.read_text()
        doc = yaml.safe_load(text) or {}
        pr = pr_trigger(doc)
        if pr is None:
            continue
        filtered = any(k in pr for k in ("paths", "paths-ignore"))
        for ctx in sorted(set(STATUS_CONTEXT.findall(text))):
            reporters.setdefault(ctx, []).append((wf.name, "status:" + ctx, not filtered))
        for job_id, job in (doc.get("jobs") or {}).items():
            is_matrix = bool((job.get("strategy") or {}).get("matrix"))
            skippable_matrix = is_matrix and "if" in job
            for name in display_names(job_id, job):
                reporters.setdefault(name, []).append(
                    (wf.name, job_id, not filtered and not skippable_matrix))

    for name, jobs in sorted(reporters.items()):
        if len(jobs) > 1:
            where = ", ".join(f"{w}:{j}" for w, j, _ in jobs)
            problems.append(f"status check {name!r} is reported by {len(jobs)} jobs: {where}")

    protection = json.loads((root / "forge" / "branch-protection.json").read_text())
    checks = protection["repos"][REPO_SLUG]["protection"]["required_status_checks"]["checks"]
    for check in checks:
        ctx = check["context"]
        jobs = reporters.get(ctx, [])
        if not jobs:
            problems.append(f"required check {ctx!r} is reported by no pull-request job: it would stay pending")
            continue
        for w, j, always in jobs:
            if not always:
                problems.append(
                    f"required check {ctx!r} comes from {w}:{j}, which does not report on every pull request "
                    "(a paths filter, or a skippable matrix job): it would stay pending")

    for p in problems:
        print(p)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1] if len(sys.argv) > 1 else "."))
