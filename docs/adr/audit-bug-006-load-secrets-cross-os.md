---
id: audit-bug-006-load-secrets-cross-os
type: audit
status: active
created: "2026-05-19"
---

# BUG-006 — load-secrets cross-OS completeness audit

> **SUPERSEDED (2026-06-20) by [ADR-021](adr-021-cli-orchestration-roadmap.md) §"Supersedes".**
> The recommended remediation below — porting the missing functions to
> `load-secrets.ps1` (the "BUG-008/009/010" parity PRs) — is **moot**. ADR-021's
> strangler-fig converges secrets into a single cross-platform `dotf secrets`
> noun, which **deletes** `load-secrets.ps1` rather than growing it. Adding `.ps1`
> parity now is dual-maintenance debt the convergence will throw away. The
> cross-OS *gap analysis* in this audit stays valid as the contract `dotf secrets`
> must reconstruct (the Linux superset). See [AUDIT-007](audit-007-cli-convergence-state.md) PR 8.

> Surfaced by [AUDIT-002](audit-002-cross-os-duplication.md) as a ratio anomaly: `load-secrets.sh` 1058 LOC vs `load-secrets.ps1` 254 LOC (0.24 ratio). Audit answers the question: is the Windows side missing features (DEFECT), or is the Linux side bloated (REFACTOR)? Generated 2026-05-19.

## TL;DR

**Verdict: 95% DEFECT, 5% REFACTOR.** The Windows side is missing **10 of 13 public functions** present on Linux. Bloat in `load-secrets.sh` is minimal — function sizes are proportional to their work. The 4× LOC ratio reflects a **real cross-OS contract gap**, not Linux-side cruft.

**Recommended action: 3 port PRs (full CRUD parity → maintenance helpers → security helpers).** No refactor PR needed on the sh side. Detailed below.

## Public function inventory

### `load-secrets.sh` (13 public + 7 private = 20 total)

Public:

| Function | LOC | Lines | Role |
|---|---:|---|---|
| `secrets_load` | 37 | 208-244 | Load all enabled secrets into env (CRUD: Read all) |
| `secrets_refresh` | 27 | 245-271 | Force-reload all secrets (drop cache) |
| `secrets_list` | 69 | 272-340 | List configured secrets and current state |
| `secrets_show` | 108 | 341-448 | Show decrypted value of a single secret (debug) |
| `secrets_add` | 65 | 449-513 | Add a new env-secret (CRUD: Create) |
| `secrets_check` | 72 | 514-585 | Validate mapping ↔ files consistency |
| `secrets_clean` | 55 | 586-640 | Remove orphan `.secret.age` files not in mapping |
| `secrets_rotate` | 69 | 641-709 | Rotate the age key + re-encrypt all secrets |
| `secrets_remove` | 85 | 710-794 | Remove a secret (CRUD: Delete) |
| `secrets_sync` | 54 | 795-848 | Sync secrets state to repo (git add + commit hook) |
| `secrets_add_file` | 77 | 849-925 | Add a file-secret (`@VAR=filename>dest`) |
| `secrets_help` | 93 | 926-1018 | Show help text + usage |
| `secrets_audit` | 13 | 1032-1057 | Print audit trail of secret access |

Private helpers: `_is_file_secret`, `_parse_file_secret_value`, `_secrets_load_file_entry`, `_get_secrets_repo_dir`, `_secrets_sync_to_repo`, `_secrets_load_entry`, `_secrets_audit_log`.

### `load-secrets.ps1` (5 public + 3 private = 8 total)

Public:

| Function | Lines | Maps to sh equivalent |
|---|---|---|
| `Import-EnvSecret` | 55-72 | helper of `secrets_load` |
| `Import-FileSecret` | 73-117 | helper of `secrets_load` (file-secret branch) |
| `Import-AllSecrets` | 118-143 | **`secrets_load`** ✓ |
| `Invoke-SecretsRefresh` | 144-176 | **`secrets_refresh`** ✓ |
| `Show-SecretsList` | 177-end | **`secrets_list`** ✓ |

Private helpers: `Test-IsFileSecret`, `Expand-TildePath`, `Invoke-AgeDecrypt`.

## Gap analysis

### Functions present in `.sh`, MISSING in `.ps1`

10 public functions absent from PowerShell:

| sh function | PS1 equivalent | Classification | Priority |
|---|---|---|:---:|
| `secrets_show` | — | DEFECT (debug helper) | P2 |
| `secrets_add` | — | **DEFECT (CRUD Create)** | **P1** |
| `secrets_check` | — | **DEFECT (validation)** | **P1** |
| `secrets_clean` | — | DEFECT (maintenance) | P2 |
| `secrets_rotate` | — | DEFECT (security) | P2 |
| `secrets_remove` | — | **DEFECT (CRUD Delete)** | **P1** |
| `secrets_sync` | — | Linux-only (git-coupled) — investigate | P3 |
| `secrets_add_file` | — | **DEFECT (CRUD Create file-secret)** | **P1** |
| `secrets_help` | — | DEFECT (usability) | P2 |
| `secrets_audit` | — | DEFECT (security) | P2 |

### Functions present in BOTH (cross-OS contract honoured)

3 paired:
- `secrets_load` ↔ `Import-AllSecrets`
- `secrets_refresh` ↔ `Invoke-SecretsRefresh`
- `secrets_list` ↔ `Show-SecretsList`

The PowerShell side covers loading + refreshing + listing, but not creating, validating, deleting, or maintaining. A Windows user has **read-only access to secrets via the dotfiles tooling**; any mutation requires manual `age` invocation.

### Linux-side bloat analysis

| Function | LOC | Smell? |
|---|---:|---|
| `secrets_load` | 37 | clean |
| `secrets_refresh` | 27 | clean |
| `secrets_list` | 69 | acceptable (formatted output) |
| `secrets_show` | 108 | **moderate** — multiple branches for env-secret vs file-secret rendering, color formatting, masking. Acceptable for a debug command. |
| `secrets_help` | 93 | acceptable (embedded multi-line help text) |
| All others | 13-85 | proportional to work |

No clear bloat. `secrets_show` is the only candidate for simplification (could extract a `_render_secret_entry` helper) but the gain would be <30 LOC and the function is already self-contained.

**Verdict on bloat**: ~30 LOC of optional simplification possible in `secrets_show`, NOT worth a refactor PR on its own. If `secrets_show` gets ported to ps1, factor the rendering helper in BOTH languages at port time.

## Sequenced PR list

| # | PR | Scope | Estimated diff | Priority |
|:---:|---|---|---:|:---:|
| 1 | **BUG-008-secrets-ps1-crud-parity** | Port `secrets_add`, `secrets_remove`, `secrets_add_file` to ps1. Closes the CRUD gap (Create + Delete for env + file secrets). | ~200 LOC ps1 + bats parity asserts | **P1** |
| 2 | **BUG-009-secrets-ps1-validation** | Port `secrets_check` to ps1. Windows users gain mapping ↔ file consistency validation. | ~60 LOC ps1 + bats | **P1** |
| 3 | **BUG-010-secrets-ps1-maintenance** | Port `secrets_clean`, `secrets_show`, `secrets_help` to ps1. Maintenance + debug parity. | ~120 LOC ps1 + bats. Consider `_render_secret_entry` helper extraction in BOTH languages. | P2 |
| 4 (defer) | **secrets_rotate ps1** | Security-sensitive port; needs design review (Windows key storage conventions). | ~100 LOC, design first | P3 |
| 5 (defer) | **secrets_audit ps1** | Audit trail port; Windows audit log conventions differ. | ~50 LOC | P3 |
| 6 (defer / investigate) | **secrets_sync ps1?** | Is this Linux-only-by-design (git workflow) or should Windows have it? Investigate before porting. | TBD | P3 |

## Anti-recommendations

- **Do NOT refactor `load-secrets.sh` first to make porting easier.** The functions are already well-decomposed. Refactor risk + zero functionality gain.
- **Do NOT port all 10 functions in a single PR.** The ~480 LOC would exceed atomic-PR discipline (SDD-001's 300 LOC ceiling) and tangle CRUD parity with security helpers. Split as above.
- **Do NOT add new test coverage to `load-secrets.sh` as part of these PRs.** Existing `tests/load-secrets.bats` is the contract; each port PR mirrors the relevant subset for ps1 with new tests in `tests/load-secrets-ps1.bats` (or extend an existing ps1-test file).

## Empirical verification

```bash
$ wc -l scripts/load-secrets.sh scripts/load-secrets.ps1
 1058 scripts/load-secrets.sh
  254 scripts/load-secrets.ps1
 1312 total

$ grep -c '^[a-z_]\+()' scripts/load-secrets.sh
20

$ grep -c '^function ' scripts/load-secrets.ps1
8
```

Function-count ratio matches the LOC ratio (8/20 = 0.4 vs 254/1058 = 0.24), with the LOC gap slightly bigger because `secrets_help` (93 LOC of help text) has no ps1 equivalent yet.

## Observations (not action items)

- **The Windows-side secrets workflow is currently "read-only by tool, write by hand"**. A Windows user wanting to add a secret must run `age -r <pubkey>` manually and edit `env-mapping.conf` directly. This is functional but inconsistent with the Linux UX. BUG-008 closes that gap.
- **`secrets_show` is the only sh function with mild bloat signals.** Plan the refactor at port-time, not as a standalone PR.
- **No sh-side defects found.** All 20 functions are well-decomposed and proportional to their work. REFACTOR-001's `scripts/` audit will not target `load-secrets.sh`.

## Closing

- [ ] BUG-008-secrets-ps1-crud-parity → P1, open next.
- [ ] BUG-009-secrets-ps1-validation → P1, after BUG-008.
- [ ] BUG-010-secrets-ps1-maintenance → P2.
- [ ] Tick BUG-006 in the project task backlog with this report's path + finding link.

## References

- [AUDIT-002](audit-002-cross-os-duplication.md) — surfaced the ratio anomaly that triggered this audit.
- [AUDIT-004](dotfiles-architecture-map.md) — locates `load-secrets.{sh,ps1}` in the runtime hooks layer.
- [adr-002-age-over-gpg](adr-002-age-over-gpg.md) — the encryption choice underlying both files.
- Pattern: secrets-security.md in `00_meta/patterns/` (the security envelope these functions operate within).
