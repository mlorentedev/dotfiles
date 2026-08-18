---
generated: true
generated_from: 00_meta/skills/terraform/SKILL.md
generated_sha: 38f52acfb023c8df
id: terraform-skill
type: skill
status: active
created: '2026-06-02'
owner: manu
name: terraform
description: Use when writing, reviewing, or debugging Terraform/OpenTofu modules,
  tests, CI, scans, or state ops — diagnoses the failure mode (identity churn, secret
  exposure, blast radius, CI drift, compliance, state corruption, provider-upgrade
  risk, testing blind spots) with version-aware guards, then generates fixes.
source: https://github.com/antonbabenko/terraform-skill (terraform-skill)
license: Apache-2.0
keywords: [terraform, opentofu, tf, terraform plan, terraform apply, hcl, iac]
paths: ['**/*.tf', '**/*.tfvars', '**/terraform/**']
---
# Terraform / OpenTofu

Diagnose first, then fix. Every response states runtime assumptions, identifies the risk category, proposes remediation with tradeoffs, gives validation commands, and includes rollback notes for destructive changes.

## Response contract
1. **Assumptions & version floor** — `terraform` vs `tofu`, version, providers, state backend, criticality.
2. **Risk category addressed** — one of the failure modes below.
3. **Remediation & tradeoffs** — what was chosen, what was traded off.
4. **Validation plan** — exact commands (`fmt -check`, `validate`, `plan -out`, policy checks).
5. **Rollback notes** — undo procedure + evidence retention for state-mutating changes.

## Failure-mode routing
| Category | Symptoms |
|----------|----------|
| Identity churn | addresses shift, `count` index churn, missing `moved` blocks |
| Secret exposure | secrets in defaults/state/logs/CI |
| Blast radius | oversized stacks, shared prod/non-prod state |
| CI drift | local plan ≠ CI plan, unpinned versions |
| Compliance gaps | no policy stage, no approval, no evidence |
| Testing blind spots | plan-only validation of computed/set values |
| State corruption | stuck lock, backend migration, drift |
| Provider upgrade/lifecycle | breaking bumps; `removed` blocks for retiring resources |
| Bootstrap misuse | `null_resource` + `local-exec`/`remote-exec` |

## Module hierarchy & layout
resource → resource module → infrastructure module → composition. Keep modules small, single-responsibility. Separate environments from modules; use `examples/` as docs *and* test fixtures.
```
environments/  modules/  examples/
my-module/: README.md main.tf variables.tf outputs.tf versions.tf examples/{minimal,complete}/ tests/module_test.tftest.hcl
```
**Naming:** descriptive resource names; reserve `this` for genuine singletons; context-prefixed variables; standard files. **Block order:** resource = `count`/`for_each` → args → `tags` → `depends_on` → `lifecycle`; variable = `description` → `type` → `default` → `validation` → `nullable` → `sensitive`.

## count vs for_each
| Scenario | Use |
|----------|-----|
| Boolean create/don't | `count = cond ? 1 : 0` |
| Items may reorder/remove | `for_each = toset(list)` |
| Reference by key / named | `for_each = map` |

**Never use a list index as long-lived identity** — removing a middle element reshuffles every later address.

## Testing decision matrix
| Situation | Approach | Tools |
|-----------|----------|-------|
| Quick syntax | static | `validate`, `fmt` |
| Pre-commit | static + lint | `tflint`, `trivy`, `checkov` |
| 1.6+, simple logic | native | `terraform test` |
| Pre-1.6 / Go expertise | integration | Terratest |
| Security/compliance | policy-as-code | OPA, Sentinel |
| Cost-sensitive | mock providers (1.7+) | native + mocks |

Native test rules (1.6+): `command = plan` for input-derived values; `command = apply` for computed values (ARNs, generated names) and set-type nested blocks (cannot be `[0]`-indexed — use `for` expressions or materialize via apply).

## State management
**Never local state in teams/prod.** Use a remote backend (locking, encryption, versioning, audit):
```hcl
terraform {
  backend "s3" {
    bucket = "my-tf-state"
    key    = "prod/vpc/terraform.tfstate"
    region = "us-east-1"
    encrypt      = true
    use_lockfile = true   # native S3 locking, 1.10+ (else dynamodb_table)
  }
}
```
Organize per-environment + per-component (hybrid). Split state when different teams/cadences or >500 resources; combine when tightly coupled and <100.

## Version management
| Component | Strategy | Example |
|-----------|----------|---------|
| Runtime | pin minor | `required_version = "~> 1.9"` |
| Providers | pin major | `version = "~> 5.0"` |
| Modules (prod) | pin exact | `version = "5.1.2"` |
Commit `.terraform.lock.hcl`. Keep provider/runtime upgrades in a separate PR from functional changes.

## Security & compliance
Run `trivy config .` and `checkov -d .`. **Don't** store secrets in variables/`.tfvars`, use the default VPC, skip encryption, open SGs to `0.0.0.0/0`, or use inline `ingress`/`egress`. **Do** source secrets from cloud secret managers, use `write_only`/`*_wo` args (1.11+), dedicated VPCs, least-privilege SGs, separate `aws_vpc_security_group_{ingress,egress}_rule`. Note: `sensitive = true` masks display only — the value still lives in state.

## CI/CD
Stages: **validate → test → plan → apply** (with environment protection). Apply the **reviewed plan artifact** (don't re-run `plan` in the apply job). Mock providers on PR validation; real-cloud integration only on main/scheduled; tag + auto-clean test resources.

## Modern features (verify the runtime floor before emitting)
`try()` 0.13+ · `nullable=false` 1.1+ · `moved` 1.1+ · `optional()` defaults 1.3+ · `import` blocks 1.5+ · `check` 1.5+ · native `test` 1.6+ · mock providers 1.7+ · `removed` 1.7+ · cross-variable validation 1.9+ · S3 native lock 1.10+ · `write_only` 1.11+. Both Terraform and OpenTofu (1.6+) are supported.

The upstream skill bundles eight `references/*.md` deep-dives (testing, modules, CI/CD, security, state, code patterns, LSP, quick-reference) — consult them for full worked examples.

---
*Vendored from [antonbabenko/terraform-skill](https://github.com/antonbabenko/terraform-skill) (Apache-2.0, © 2026 Anton Babenko). Adapted for the cross-agent skill pipeline; the `references/` deep-dives remain upstream. See `harness/skills/ATTRIBUTION.md`.*
