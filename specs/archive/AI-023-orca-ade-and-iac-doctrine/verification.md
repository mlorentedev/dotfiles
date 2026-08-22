---
id: "AI-023-orca-ade-and-iac-doctrine-verification"
type: spec-verification
status: implementing
created: "2026-08-22"
spec: "specs/AI-023-orca-ade-and-iac-doctrine/proposal.md"
tags: [spec, verification]
---

# Verification: AI-023-orca-ade-and-iac-doctrine

## Test Evidence

1. `bash scripts/compile-harness.sh --check` -> exit 0 (no harness drift).
2. `bats tests/cli-embed-templates.bats tests/harness-generated-sha.bats tests/compile-harness-real.bats` -> 9/9 passed.
3. `go test ./...` in `cli/` -> all packages passed.
