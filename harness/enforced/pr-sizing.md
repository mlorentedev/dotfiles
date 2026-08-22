- **Atomic PRs, ~300 LOC hard cap** (excl. tests, generated files, lockfiles, fonts, vendored classes). One logical change per PR; "while I was here I also…" is a red flag — split it.
- **When the cap collides with shipping a functional unit,** hunt first for a seam where *every* intermediate merge is functional. Only when no such seam exists does the functional unit win over the cap — never split so that an intermediate PR merges uncalled code.
- The resolution is then one over-cap PR with the overage **declared** in its description: the real figure, code split from docstrings so the size is legible, the rejected seam named, and any spec that claimed the work would fit corrected. A cap exceeded *quietly* is the failure; a cap exceeded in the open is a reviewed decision.
- This is the default *proposal*, never a bypass of the review gate in §1 nor of the escalation an over-cap diff triggers. Surface the decision; do not ship it silently.

### Verifying oversized lockfile diffs (Set Comparison)

Format churn across package manager versions (e.g. `uv`, `poetry`, `npm`, `cargo`) can produce massive multi-hundred line lockfile diffs for a single version bump, burying unintended dependency movements.
* **Never visually read huge lockfile diffs.**
* **Verify versions moved:** `git diff <lockfile> | grep -E '^[+-]version = '` to confirm only the intended target dependencies changed.
* **Verify hash integrity:** Extract invariant hash sets with `grep -oE 'sha256:[a-f0-9]{64}' | sort -u` on both revisions and `comm` the results. Any unexpected set delta must be investigated before committing.
