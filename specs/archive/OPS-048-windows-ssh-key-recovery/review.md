---
spec: "OPS-048-windows-ssh-key-recovery"
verdict: "PASS WITH GAPS"
reviewed_sha: "3bc1708a06d6a2052d95ba42226f18d1683c6daf"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-25"
---

## Adversarial review

**Scope**: OPS-048-windows-ssh-key-recovery
**Sources**: `specs/OPS-048-windows-ssh-key-recovery/{proposal,tasks,verification}.md`, PR diff 1e68866...HEAD

### Spec and task alignment
- All acceptance criteria are demonstrably implemented and covered by Pester tests.
- Runbook clearly dictates out-of-band trust bootstrapping as required by AC4.
- Idempotent script behaviors successfully achieve AC1 and AC2 goals.
- Unhandled edge cases around underlying Windows APIs introduce theoretical gaps in error handling and setup states.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | host-reconciliation | `New-Item` fallback for `C:\ProgramData\ssh` creates a hijackable directory if `-SkipSystemConfiguration` is used and OpenSSH hasn't locked it | `C:\ProgramData` grants `Users` `Create files`. `New-Item -ItemType Directory` inherits this without locking it down | UNTESTED | code |
| Minor | THEORETICAL | acl-reconciliation | `Set-PrivateKeyAcl` / `Set-AdministratorAuthorizedKeysAcl` crash on files with orphaned SID owners, preventing ACL repair | `[Security.Principal.NTAccount]"`<SID>`".Translate(...)` throws if the owner string is already an unresolved SID | UNTESTED | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Criteria met on happy path; theoretical directory permission gap on edge cases. |
| Verification       | A | Pester test suite and live evidence provided for each criterion. |
| Scope              | A | Diff matches proposal exactly; no scope creep. |
| Reliability        | C | Unhandled error path for orphaned SIDs and directory creation fallback. |
| Maintainability    | B | Acceptable structure; some PowerShell ACL API quirks exposed. |
| Handoff-readiness  | A | Spec updates and runbook included, explicit out-of-scope items respected. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- Set explicit locked-down ACLs on `C:\ProgramData\ssh` if `Update-AuthorizedKeyFile` has to create it with `New-Item`.
- Handle the case where `$currentAcl.Owner` is already a SID string in the ACL scripts to avoid `.Translate` exceptions.
- The contract set is closed. Disposition these gaps in `verification.md` or track them in follow-up tickets; do not edit the contract files. `dotf spec archive` is advisable.
