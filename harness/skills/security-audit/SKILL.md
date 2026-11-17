---
generated: true
generated_from: 00_meta/skills/security-audit/SKILL.md
generated_sha: 0030f482b028d9eb
id: security-audit-skill
type: skill
status: active
created: "2026-11-07"
owner: manu
name: security-audit
description: Multi-phase security audit and vulnerability research using coverage-led hunting, adversarial verification ('survive a fight'), and schema-validated ledgers. Auto-triggers on high-risk specs, sensitive attack surface changes, or milestone pre-release audits.
source: https://github.com/cloudflare/security-audit-skill
license: MIT
keywords: [security audit, vulnerability research, penetration testing, threat modeling, attack surface, cwe, audit codebase, security review, pen test]
paths: ['**/auth/**', '**/crypto/**', '**/security/**', '**/mcp/**', '**/permissions/**', '**/tokens/**']
requires: [test]
---

# Security Audit & Vulnerability Research

A structured, defensive, source-first security audit framework. Uncovers vulnerabilities violating concrete trust boundaries and produces verifiable evidence, reproducible proof-of-concepts, and minimal effective fixes.

> **Origin:** adapted from [cloudflare/security-audit-skill](https://github.com/cloudflare/security-audit-skill) (MIT). Adapted for the cross-agent skill pipeline with autonomous trigger rules, zero-debt bitácora ticket integration, and placement model compliance.

---

## Operating Modes

1. **Guidance Mode (Default)**:
   - For focused security inquiries, triage, reviewing specific PR diffs, or inspecting individual functions.
   - Does NOT run the full multi-phase audit or generate ledgers. Operates within the current session context.
2. **Full Audit Mode**:
   - Triggered on explicit audit/pentest requests (`/security-audit`), pre-release gates, or autonomous triggers on high-risk subsystems.
   - Executes the 6-phase lifecycle, populates file-based ledgers, runs validators, and compiles final deliverables into `docs/security/audits/`.

---

## Agent-Side Activation Rule (Autonomous Proactive Trigger)

> **Proactive Mode.** In addition to manual execution via `/security-audit`, agents must **evaluate and trigger this skill autonomously** under the following conditions:

### 1. Critical Surface Trigger
Silently evaluate the diff or touched files. Trigger a focused security audit when changes touch:
- **Authentication & Sessions**: Tokens, JWT, cookies, OAuth, password hashing, session revocation.
- **Authorization & RBAC**: Privilege checks, role elevation, tenant isolation boundaries.
- **Data Protection & Crypto**: Encryption keys, ciphertext handling, signature verification.
- **Execution Sinks**: Subprocess spawns (`exec`, `spawn`, `Popen`), shell command assembly, dynamic deserialization.
- **AI & MCP Tooling**: Tool-calling schemas, MCP server exposed methods, prompt-assembly with untrusted external inputs.

### 2. High-Risk Spec Trigger
When inspecting `specs/<feature-id>/proposal.md` during implementation or verification, if `risk: high` is declared, the agent automatically runs Phase 2 (Hunting) and Phase 3 (Candidate Validation) over the feature's boundary before recommending merge.

### 3. Milestone / Release Trigger
When preparing a version release tag, deploying new public endpoints, or onboarding external third-party repositories into the workspace, the agent proactively offers or executes a full security audit pass.

---

## The Six-Phase Audit Workflow

```
[Phase 1: Recon] ───► [Phase 2: Hunting] ───► [Phase 3: Validation]
                             │                       │
                             ▼                       ▼
[Phase 6: Reporting] ◄─── [Phase 5: Record Check] ◄─── [Phase 4: Output]
```

### Phase 1: Reconnaissance & Mapping
1. Map the system architecture, entry points, trust boundaries, principals, and data flows.
2. Produce `architecture.md` summarizing external attack surfaces and sensitive sinks.
3. Initialize `coverage-ledger.json` breaking the target into bounded units mapped to specific attack classes.
4. Validate ledger integrity using `node scripts/validate-coverage-ledger.cjs`.

### Phase 2: Coverage-Led Hunting
1. Allocate inspection units systematically across attack classes.
2. For each unit, trace untrusted inputs to critical sinks (code-read only).
3. Record raw vulnerability hypotheses as *candidates*. A candidate is NOT a finding until it survives validation.

### Phase 3: Candidate Validation ("Survive a Fight")
> **The Iron Rule:** *Every finding must survive a fight.*
1. Pass each candidate to an independent verification pass (or independent subagent) whose explicit mandate is to **disprove or invalidate** the finding.
2. The verifier checks:
   - Does upstream validation, framework sanitization, or type safety neutralize the vector?
   - Is the sink actually reachable by untrusted actors?
   - Is this an intended feature or theoretical cosmetic issue rather than an exploit?
3. Classify candidates:
   - `confirmed`: Unbroken trace from untrusted source to critical sink with concrete security consequence.
   - `needs_validation`: Plausible path but blocked by missing runtime environment/context.
   - `rejected`: Disproved candidate; recorded to document discarded assumptions and prevent re-testing.

### Phase 4: Structured Output Generation
1. Compile surviving findings into `findings.json`.
2. Map each confirmed finding to its CWE, severity, affected files, line numbers, and impact.
3. Validate format deterministically: `node scripts/validate-findings.cjs`.

### Phase 5: Independent Record Verification
1. Re-check the generated JSON records against current source to ensure zero hallucination of line numbers or APIs.
2. Remove any finding where source code evidence is ambiguous.

### Phase 6: Final Reporting & Ticket Disposition
1. Render report in `docs/security/audits/YYYY-MM-DD-<target>.md`.
2. **Zero Debt Enforcement**: Every `confirmed` finding must be immediately ticketed as an issue in the bitácora board with root cause, reproduction steps, and suggested fix.

---

## Core Attack Classes Reference

When hunting, evaluate against these 10 core domain vectors:

1. **Injection & Parsing**:
   - SQL, NoSQL, shell command injection, LDAP, path traversal, template injection (SSTI), XPath.
2. **Auth & Identity Protocols**:
   - Broken object-level authorization (BOLA/IDOR), missing function-level access control, JWT algorithm confusion, session fixation, CSRF.
3. **AI & LLM Boundaries**:
   - **Indirect Prompt Injection**: Untrusted inputs escaping to execute privileged tool/MCP calls.
   - **Untrusted RAG/MCP Outputs**: Treating LLM generation or external MCP tool results as trusted data.
   - **Action-Binding Failures**: Untrusted text triggering irreversible side-effects without explicit human authorization.
   - **Guardrail Fallacy**: Relying on prompt wording rather than deterministic code-level security checks.
4. **Supply Chain & Dependencies**:
   - Unpinned dependency ranges, install scripts (`preinstall`/`postinstall`) touching network/filesystem, bloated or hijacked dependencies.
5. **Memory Safety & Binary**:
   - Buffer overflows, use-after-free, integer wraps, double free, race conditions on shared raw pointers.
6. **Client-Side & Web Security**:
   - DOM XSS, prototype pollution, permissive CORS (`Access-Control-Allow-Origin: *` with credentials), open redirects.
7. **Cloud & Deployment Configuration**:
   - Over-privileged IAM roles, world-readable storage buckets, hardcoded secret tokens in infrastructure manifests, unauthenticated debug ports.
8. **Data Isolation & Multi-Tenancy**:
   - Cross-tenant data leaks, cache pollution between tenants, unencrypted secrets at rest.
9. **RPC & Messaging**:
   - Insecure deserialization (e.g. pickle, unvalidated JSON parse to prototype), missing authentication on inter-service gRPC/message queues.
10. **Resource Exhaustion (DoS)**:
    - Unbounded loops, unpaginated database queries, quadratic regex (ReDoS), decompression bombs, memory leaks in long-running processes.

---

## Universal Execution Safety

- Source code analysis is **read-only** by default.
- Never run exploit attempts, target-controlled tests, or fuzzers on host environments or live servers.
- Execute dynamic reproductions strictly inside isolated, disposable sandboxes or containers (no external network, scratch-only writes, resource-limited).
- Never dump decrypted credentials or secrets to output transcripts.

---

## Deliverable Format (`docs/security/audits/YYYY-MM-DD-<target>.md`)

```markdown
---
id: "audit-YYYY-MM-DD-<target>"
type: reference
status: active
owner: manu
created: "YYYY-MM-DD"
tags: [security, audit, cwe]
---

# Security Audit: <target> (<date>)

## Executive Summary
- **Scope**: `<repo or module>`
- **Total Units Audited**: `<N>`
- **Confirmed Vulnerabilities**: `<N>`
- **Needs Validation**: `<N>`
- **Rejected Candidates**: `<N>`

## Confirmed Findings
| ID | Severity | CWE | Title | File & Line | Status |
|---|---|---|---|---|---|
| SEC-001 | High | CWE-89 | SQL Injection in user search | src/db/users.ts:42 | Ticketed (#123) |

### SEC-001: <Title>
- **Vulnerability Description**: ...
- **Source Trace**: `entrypoint()` → `controller()` → `sink()`
- **Adversarial Verification Proof**: Why this is exploitable and survived refutation.
- **Remediation**: Smallest effective patch.
```
