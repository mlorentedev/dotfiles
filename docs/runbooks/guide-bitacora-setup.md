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

## 1b. Which repo does an issue belong to? (home repo)

An issue is filed in the repo it **primarily modifies**, then auto-added to this board (the `Repository` field makes the home explicit). Rule of thumb:

| Work is mostly about… | Home repo | Prefix |
|---|---|---|
| Harness / methodology / skills / agents / `AGENTS.md` / SDD | `dotfiles` | HARNESS-* , most OPS-* |
| Vault **content or structure** (patterns, `00_meta/` layout, links) | `knowledge` | CUR-* |
| A specific project's code or infra (e.g. the homelab CI runner) | that project's repo | per-project |

**Tie-breaker** when work spans repos (e.g. a skill whose SSOT is the vault but deploys via dotfiles): file it where the **deploy/runtime contract** lives — dotfiles for harness skills — and reference the rest. When genuinely cross-cutting, default to `dotfiles`. Per-project *infrastructure* (a runner, a cluster service) is the exception: it belongs to that project's repo even if a methodology rule consumes it.

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

# 2. Deploy BOTH workflows (§7) + upload the secret they need
#    copy .github/workflows/add-to-project.yml AND bitacora-status.yml into the repo, then:
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

> `dotf init` runs steps 1–2 automatically for new repos.

## 5. Operating discipline — the status lifecycle (HARNESS-010)

The board is only worth anything if `Status` reflects reality. The cross-agent rule is canonical in
`AGENTS.md` Standing Order #8; this section is the mechanical reference. On every issue you act on:

| When | Status | How |
|------|--------|-----|
| New issue opened | **Backlog** | automatic — *Item added* built-in workflow |
| **You start working it** | **In Progress** | automatic — **self-assign** (`gh issue edit <n> --add-assignee @me`); the `bitacora-status.yml` workflow (§7) flips the field |
| You hit a blocker | **Blocked** | manual — set the field (helper below) + name the blocker in a comment |
| You finish / close it | **Done** | automatic — *issue closed* built-in workflow |

**Automated (HARNESS-010):** *Backlog* (item added), *In Progress* (issue assigned → `bitacora-status.yml`),
and *Done* (issue closed). In normal flow you therefore touch the field **only for `Blocked`** —
self-assign starts it, closing finishes it. `Blocked` stays manual on purpose: there is no reliable
machine signal for "stuck", and naming the blocker is a human judgement.

Set **Priority** (default P2) and **Type** manually; set the **ID** field forward-only. Manual helper
to move issue `#N` (Status options: In Progress `6c133cc8` · Blocked `7705a647` · Done `bb482da4`):

```bash
ITEM=$(gh project item-list 1 --owner mlorentedev --format json --limit 300 \
  | python3 -c "import json,sys;[print(i['id']) for i in json.load(sys.stdin)['items'] \
       if i.get('content',{}).get('number')==N]")
gh project item-edit --id "$ITEM" --project-id PVT_kwHOAM7xJs4BZ6GY \
  --field-id PVTSSF_lAHOAM7xJs4BZ6GYzhU1dtI --single-select-option-id 7705a647   # -> Blocked
```

## 6. Views

- **Board** grouped by Status (Backlog / In Progress / Blocked / Done).
- **By repo** — group by Repository (tasks span all repos).
- **PRs pending review** (OPS-003, created 2026-06-09) — the cross-repo PR dashboard:
  - Type: **Table** (scannable columns beat a one-field Board for PR triage).
  - Filter: `is:pr is:open`. Group by: **Repository**.
  - Columns: Title, Assignees, Status, Repository (hide Priority/Type/ID — issue fields, noise on PRs).
  - PRs land on the board automatically (`add-to-project.yml` `pull_request` trigger, deployed to every repo by `scripts/bitacora-rollout.sh`, which also backfills open ones).
  - Views are UI-only (no API) — pressing **Save** on the tab is what persists the filter.
  - Optional second view for review-blocked PRs: `is:pr is:open review:required`.

## 7. Workflows (deploy BOTH to every linked repo)

Each linked repo carries two workflows; the canonical copies live in
`mlorentedev/dotfiles/.github/workflows/`.

**Multi-repo rollout (OPS-002, #258):** run `./scripts/bitacora-rollout.sh` — idempotent, one
pass over every non-archived, non-fork repo: project link + `BITACORA_PAT` secret + both
workflows (diff-aware) + open issue/PR backfill. `--check` = dry-run. The sections below
remain the manual path for a single repo.

> **Decision (2026-06-09, #258):** board-add mechanism = this per-repo Action, NOT the
> built-in project "Auto-add" workflow — git-native (IaC, no UI-only config), uniform and
> portable via `dotf init`, covers PRs (OPS-003), and avoids the plan-dependent
> auto-add workflow limit.

### 7a. `add-to-project.yml` — puts opened **issues and PRs** on the board

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

> **Gotcha (2026-06-25):** `BITACORA_PAT` must be a **classic** PAT carrying the `project`
> scope (plus `repo`). **Fine-grained PATs cannot write user-owned Projects v2** — both
> `add-to-project` and the Projects v2 GraphQL (§7b) fail (403 / `unknown owner type`),
> regardless of the fine-grained "Projects: read & write" permission. If `add-to-project`
> 403s after a token rotation, regenerate `BITACORA_PAT` as a **classic** token.

### 7b. `bitacora-status.yml` — moves an **assigned** issue to *In Progress* (HARNESS-010)

Fires on `issues: [assigned]`, so self-assigning at pickup flips `Status` automatically (§5). It is
idempotent (`addProjectV2ItemById` returns the existing item) and guarded to skip closed issues. Uses
`actions/github-script` for the Projects v2 GraphQL — **not `gh project`**, which fails with
`unknown owner type` under a fine-grained PAT. IDs are the canonical ones from §2.

```yaml
name: Bitácora status — In Progress on assign
on:
  issues:
    types: [assigned]
permissions: {}
jobs:
  in-progress:
    if: github.event.issue.state == 'open'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/github-script@v9
        with:
          github-token: ${{ secrets.BITACORA_PAT }}   # project + repo scope
          script: |
            const projectId = 'PVT_kwHOAM7xJs4BZ6GY';
            const fieldId = 'PVTSSF_lAHOAM7xJs4BZ6GYzhU1dtI';
            const optionId = '6c133cc8';                 // "In Progress"
            const contentId = context.payload.issue.node_id;
            const added = await github.graphql(
              `mutation($projectId: ID!, $contentId: ID!) {
                 addProjectV2ItemById(input: { projectId: $projectId, contentId: $contentId }) { item { id } }
               }`, { projectId, contentId });
            await github.graphql(
              `mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
                 updateProjectV2ItemFieldValue(input: {
                   projectId: $projectId, itemId: $itemId, fieldId: $fieldId,
                   value: { singleSelectOptionId: $optionId } }) { projectV2Item { id } }
               }`, { projectId, itemId: added.addProjectV2ItemById.item.id, fieldId, optionId });
```

> **Gotcha (2026-06-07):** `gh project ... --owner` → `unknown owner type` under a fine-grained PAT.
> Drive Projects v2 from `actions/github-script` (or raw `gh api graphql`) instead.

## 8. Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| "Add to bitácora" fails in ~4s, `unable to resolve action … v1` | invalid version pin | §7a — pin `@v1.0.2` |
| Issue/PR never lands on the board | workflow missing, secret absent, or PAT expired | redo §4 step 2 |
| Assigning an issue does **not** move it to *In Progress* | `bitacora-status.yml` missing in that repo, or PAT lacks `project` scope | §7b — deploy it (OPS-002 rollout) / check `BITACORA_PAT` |
| `item-edit`: *could not resolve to ProjectV2 node* | wrong project-id / field-id | §2 reference |
| `gh pr edit` errors with `projectCards` deprecation | classic-projects GraphQL path | use `gh api` instead |
| `gh project item-list` / `gh issue list --json` returns empty or `API rate limit exceeded` | GraphQL pool exhausted (Projects v2 + `--json` are GraphQL; bulk `item-list` burns it fast) | §8a — REST for issue data; board fields have no REST path |

### 8a. GraphQL rate-limit fallback

The GraphQL pool is **separate** from the REST core pool, and Projects v2 queries cost
many GraphQL points each — a couple of full `gh project item-list` sweeps can drain it.
When drained, `gh project item-list` and `gh issue list --json` fail or return empty while
REST is usually still fresh. Check both pools first (the `rate_limit` endpoint is free):

```bash
gh api rate_limit --jq '{core:.resources.core.remaining, graphql:.resources.graphql.remaining, reset:(.resources.graphql.reset|todate)}'
```

- **Issue data** (number, title, labels, state, body) → fall back to the **REST** issues
  endpoint (separate pool). `gh issue list --json` does **not** help — it is GraphQL.

  ```bash
  gh api "repos/OWNER/REPO/issues?state=open&per_page=100" --paginate \
    --jq '.[] | select(.pull_request==null) | "#\(.number) [\(.labels|map(.name)|join(","))] \(.title)"'
  ```

- **Board fields** (Status / Priority / Type / the custom `ID`) → **GraphQL-only; there is
  no REST fallback.** Degrade gracefully: use repo **labels as a proxy** for nature/priority,
  read local state, or **wait for `graphql.reset`**. Do not spin-retry — it only burns more.
- **Prevention:** fetch the board JSON **once** with a high `--limit` into a file and parse it
  locally (repeated `python`), instead of looping `gh project item-list` or calling it per item.

## References

- ADR-018 (de-vault task placement), epic #244 (Flow v2), OPS-002/003, CUR-008 (ID scheme).
- HARNESS-010 (status lifecycle): `AGENTS.md` Standing Order #8 (doctrine) + `bitacora-status.yml` (§7b automation).
