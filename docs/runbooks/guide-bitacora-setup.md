# Bitácora — GitHub Projects setup & operating guide

> The **bitácora** is a single user-level GitHub Project (#1) that holds **task state**
> across all repos (Flow v2 / ADR-018). This runbook is the SSOT for setting it up,
> linking a repo, and operating it day-to-day. Referenced by OPS-002 (#258) and OPS-003 (#266).

## 1. Model (Flow v2, ADR-018)

Task state lives in **one** place — the bitácora — never in vault `11-tasks.md`. Each fact lives in exactly one layer:

| Layer | Home |
|-------|------|
| **Task state** (backlog, status, priority) | **bitácora** GitHub Project #1 |
| Durable design (specs, ADRs) | repo `specs/`, `docs/adr/` |
| Session progress | repo scratchpad / `/handoff` continuity block |
| Cross-project brain (patterns, methodology lessons) | the vault — **no task state** |

- **Board:** https://github.com/users/mlorentedev/projects/1 — owner `mlorentedev`, number `1`
- **Project ID:** `PVT_kwHOAM7xJs4BZ6GY`

## 2. Fields & IDs (canonical reference)

The single source for field/option IDs — copy from here, do not re-discover.

| Field | Field ID | Type | Options (name:id) |
|-------|----------|------|-------------------|
| Status | `PVTSSF_lAHOAM7xJs4BZ6GYzhU1dtI` | single-select | Backlog `f65b1174` · In Progress `6c133cc8` · Done `bb482da4` · Blocked `7705a647` |
| Priority | `PVTSSF_lAHOAM7xJs4BZ6GYzhU1dxM` | single-select | P0 `339619b9` · P1 `d605c92a` · P2 `224db21f` · P3 `625bdc81` |
| Type | `PVTSSF_lAHOAM7xJs4BZ6GYzhU1dy4` | single-select | spec `aa456a84` · bug `6d7801f0` · chore `b9fadab6` · ideas `bbb28860` |
| ID | `PVTF_lAHOAM7xJs4BZ6GYzhU2ni8` | text | `AREA-NNN-slug` (CUR-008) |

> Re-list anytime: `gh project field-list 1 --owner mlorentedev --format json`.

## 3. The ID convention (CUR-008)

- Format `AREA-NNN-slug` (e.g. `harness-006-context-refresh`). **Forward-only — no backfill** of pre-existing issues.
- Carried in BOTH the issue **title** (`HARNESS-006: …`) and the `ID` custom field.
- `#NNN` (issue number) = technical pointer; `AREA-NNN` = stable human ID. The two spaces coexist by design.

## 4. Link a repo to the bitácora (one-time, per repo)

```bash
# 1. Link the repo to the project
gh project link 1 --owner mlorentedev --repo mlorentedev/<repo>

# 2. Deploy the workflow (§7) + upload the secret it needs
#    copy .github/workflows/add-to-project.yml into the repo, then:
age --decrypt -i ~/.config/age/key.txt sensitive/github.bitacora.secret.age \
  | gh secret set BITACORA_PAT --repo mlorentedev/<repo>

# 3. (optional) Backfill existing open issues onto the board
gh issue list --repo mlorentedev/<repo> --state open --limit 200 --json number \
  | python3 -c "import json,sys;[print(r['number']) for r in json.load(sys.stdin)]" \
  | while read n; do
      gh project item-add 1 --owner mlorentedev \
        --url "https://github.com/mlorentedev/<repo>/issues/$n" >/dev/null
    done
```

> Once SELF-001 lands, `init-project` runs steps 1–2 automatically for new repos.

## 5. Operating discipline — the status lifecycle (HARNESS-010)

The board is only worth anything if Status reflects reality. On every issue you act on:

| When | Action |
|------|--------|
| New issue opened | the *Item added* workflow sets **Backlog** automatically |
| **You start working it** | set Status → **In Progress** (do this BEFORE the first edit) |
| You hit a blocker | set Status → **Blocked** + name the blocker in a comment |
| You finish / close it | close the issue → the *issue closed* workflow sets **Done** |

Set **Priority** (default P2) and **Type** manually; set the **ID** field forward-only. Helper to move issue `#N`:

```bash
ITEM=$(gh project item-list 1 --owner mlorentedev --format json --limit 300 \
  | python3 -c "import json,sys;[print(i['id']) for i in json.load(sys.stdin)['items'] \
       if i.get('content',{}).get('number')==N]")
gh project item-edit --id "$ITEM" --project-id PVT_kwHOAM7xJs4BZ6GY \
  --field-id PVTSSF_lAHOAM7xJs4BZ6GYzhU1dtI --single-select-option-id 6c133cc8   # -> In Progress
```

## 6. Views

- **Board** grouped by Status (Backlog / In Progress / Blocked / Done).
- **By repo** — group by Repository (tasks span all repos).
- **PRs pending review** (OPS-003) — filter `is:pr is:open`, group by repo.

## 7. `add-to-project.yml` (workflow template)

Deploy verbatim to every linked repo. Adds opened **issues and PRs** to the bitácora.

```yaml
name: Add to bitácora
on:
  issues:
    types: [opened, reopened]
  pull_request:
    types: [opened, reopened]
jobs:
  add-to-project:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/add-to-project@v1.0.2   # pin a real release — @v1 does NOT resolve
        with:
          project-url: https://github.com/users/mlorentedev/projects/1
          github-token: ${{ secrets.BITACORA_PAT }}
```

> **Gotcha (2026-06-06):** `actions/add-to-project@v1` is unresolvable — there is no floating
> `v1` tag. Pin `@v1.0.2` (latest v1) or `@v2.0.0`. `BITACORA_PAT` needs `project` + `repo` scope.

## 8. Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| "Add to bitácora" fails in ~4s, `unable to resolve action … v1` | invalid version pin | §7 — pin `@v1.0.2` |
| Issue/PR never lands on the board | workflow missing, secret absent, or PAT expired | redo §4 step 2 |
| `item-edit`: *could not resolve to ProjectV2 node* | wrong project-id / field-id | §2 reference |
| `gh pr edit` errors with `projectCards` deprecation | classic-projects GraphQL path | use `gh api` instead |

## References

- ADR-018 (de-vault task placement), epic #244 (Flow v2), OPS-002/003, CUR-008 (ID scheme), HARNESS-010 (status lifecycle).
