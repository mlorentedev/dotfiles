#!/usr/bin/env python3
"""Assert every workflow job — and the apt step that hung — carries a ceiling.

Kept as a file rather than inline in the .bats case so the YAML parsing is
readable and testable on its own. Two rules, deliberately separate:

  jobs   every job declares timeout-minutes, so a hang ends without a human.
  apt    the apt step ALSO declares its own, because bounding apt with apt's
         options is not enough: Acquire::http::Timeout is per-scheme and does
         not cover https, and the first attempt at this fix still hung for 16
         minutes on https://archive.ubuntu.com.
"""
import glob
import os
import sys

import yaml


def jobs_without_timeout(root):
    out = []
    for path in sorted(glob.glob(os.path.join(root, ".github", "workflows", "*.yml"))):
        with open(path, encoding="utf-8") as fh:
            doc = yaml.safe_load(fh) or {}
        for name, job in (doc.get("jobs") or {}).items():
            # A `uses:` job calls a reusable workflow, which carries its own.
            if not isinstance(job, dict) or "uses" in job:
                continue
            if "timeout-minutes" not in job:
                out.append("%s:%s" % (os.path.basename(path), name))
    return out


def apt_steps_without_timeout(root):
    with open(os.path.join(root, ".github", "workflows", "ci.yml"), encoding="utf-8") as fh:
        doc = yaml.safe_load(fh)
    out, seen = [], False
    for job_name, job in (doc.get("jobs") or {}).items():
        for step in job.get("steps", []) or []:
            if "apt-get" not in str(step.get("run", "")):
                continue
            seen = True
            if "timeout-minutes" not in step:
                out.append("%s / %s" % (job_name, step.get("name", "<unnamed>")))
    if not seen:
        out.append("no apt step found at all — did it move, or was it removed?")
    return out


def main():
    root = sys.argv[1]
    failed = False
    missing = jobs_without_timeout(root)
    if missing:
        failed = True
        print("jobs with no timeout-minutes — a hang there ends only when a human cancels it:")
        for m in missing:
            print("  " + m)
    unbounded = apt_steps_without_timeout(root)
    if unbounded:
        failed = True
        print("apt steps with no timeout-minutes of their own:")
        for u in unbounded:
            print("  " + u)
        print("apt's own options are not sufficient: Acquire::http::Timeout is")
        print("per-scheme and does not cover https, which is the transport it hung on.")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
