---
id: lesson-057-byte-equivalence-assertions-require-script-dir-con
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 057: Byte-equivalence assertions require SCRIPT_DIR control, not just literal diff

**Context:** SDD-004 (PR #97). Claimed acceptance criterion: refactor preserves byte-identical output. First attempt at the assertion ran `git show main:scripts/claude-session-start.sh > $(mktemp /tmp/...sh)` then diffed PRE vs POST. False-positive diff appeared (claude-mem heal block missing in PRE, vault-health line different in PRE).</context>
<parameter name="problem">The `mktemp /tmp/...sh` for PRE put the script in /tmp, which changed its SCRIPT_DIR resolution. The pre-refactor script looked for sibling helpers (`claude-mem-heal.sh`, `vault-health.sh`, `doctor.sh`) in /tmp/, didn't find them, so silently skipped those injectors. POST ran from real `scripts/` directory and found them. The diff was 100% methodology artifact — both versions of the script behaved IDENTICALLY when given identical sibling-script paths. Worse failure mode masked by this: a real refactor regression could be hidden under the same diff noise.</parameter>
<parameter name="solution">For any refactor that asserts byte-identical output via PRE-vs-POST diff: the PRE script copy MUST live in the same directory as POST so `SCRIPT_DIR`-relative lookups (sibling scripts, configs, fixtures) resolve identically. Pattern: write PRE to `<script-dir>/<name>.sh.pre-refactor`, run, diff, delete. Captured in tests/session-start-config.bats #14 (byte-equivalence test) + verification.md row 2. Sibling caveat: if the script's output ALSO includes live state queries (vault-health unresolved-links count changes per minute), pure twice-run-deterministic isn't guaranteed — but PRE-vs-POST at the SAME MOMENT cancels that drift, so the SCRIPT_DIR fix is sufficient. Reframe of R1 from "literal byte-equivalence" to "code-controlled byte-equivalence at fixed SCRIPT_DIR".</parameter>
<parameter name="tags">["byte-equivalence", "testing-methodology", "SDD-004", "verify-before-act", "refactor-safety-net"]
**Problem:** 
**Solution:**
