---
generated: true
generated_from: 00_meta/skills/teardown/SKILL.md
generated_sha: 9789e43dfada974f
id: teardown-skill
type: skill
status: active
created: '2026-10-03'
owner: manu
name: teardown
description: Use when performing a full 360-degree teardown of an external product, repository, tool, framework, or business idea, combining deep technical architecture with product, UX, monetization, and master sales/business frameworks (Hormozi, Dunford, Cagan, Walling, Levels, Naval, Welsh, Kendall).
keywords: [teardown, product teardown, tech teardown, evaluate product, business model, hormozi, dunford, cagan, walling, naval, welsh, kendall, analisis de producto, desglosar herramienta]
paths: ['10_projects/**/research/**', '00_meta/research/**', 'research/**']
---

# Teardown — 360° Technical, Product & Business Evaluation

> Perform evidence-based, 360-degree teardowns of external technologies, libraries, tools, frameworks, and products. Evaluates both **deep engineering architecture** (protocols, concurrency, secrets, failure modes) and **product/business viability** through the mental models of leading product, sales, and bootstrapping masters (Hormozi, Dunford, Cagan, Walling, Levels, Naval, Welsh, Kendall). Supports parallel multi-model benchmarking in the vault.

---

## When to use

- Explicit trigger: `/teardown <url-or-topic>`, "haz un teardown de X", "desglosa este producto/repo", "teardown de Y", "analiza este producto/repo desde el punto de vista técnico y de negocio".
- When evaluating a new open-source repo, CLI tool, developer framework, or SaaS platform.
- When vetting potential product ideas, SaaS opportunities, or market dynamics for our sovereign portfolio discovered dynamically via vault SSOT (`10_projects/*/context.md`).
- When benchmarking multiple LLM models in parallel to analyze the same external target without git collisions.

## When NOT to use

- Quick factual lookups answerable with a single `grep` or quick documentation query.
- Reviewing or debugging our own internal codebases (use `systematic-debugging` or `adversarial-review`).
- Writing feature specifications for tickets already approved (use `spec`).

---

## Core Principles

1. **Dual-Engine Rigor (Code + Business)**: Never evaluate technology in an engineering vacuum. Beautiful code that solves the wrong problem or has zero distribution is useless. An ugly bash script that solves a painful JTBD and generates cash flow is instructive.
2. **Evidence Over Marketing**: Inspect source code, dependency trees, git commits, pricing pages, and actual user complaints.
3. **"A mention in conversation is not an exit" & Closed-Loop Actionability**:
   - Research must NEVER end as dead notes in the vault.
   - Every teardown must culminate in a concrete, actionable verdict: **Adopt (ticketed)**, **Extract Mechanism (ticketed)**, **Queue Execution Task (roadmapped)**, or **Pass/Reject (with stated rationale)**.
   - **Board Pre-Flight (mandatory, and it comes BEFORE any proposal)**: Run **both** queries, always, and paste their output. One search is not a detector, and a negative is only established by the scan:
     1. `gh issue list --state all --limit 500 --search "<topic>"`
     2. `gh issue list --state all --limit 500 --json number,title` — read the `<STREAM>-<NUM>` ids on the stream, open and closed.
     The `--limit` is not optional: the CLI default is 30, so a matching issue at position 31 or later is invisible to a query that omits it. Two outcomes, and only two — and the second is reachable only once step 2 has been run:
     - **A ticket already exists**: the deliverable is to **execute or extend it**, never to propose a second one. Report its number, state, and what is missing from it.
     - **No ticket exists**: allocate the `<STREAM>-<NUM>` id from the scan in step 2, never from memory or a cached list, and name the query that produced it.
     A text search that finds nothing does **not** establish that nothing exists: a ticket whose title and body omit the topic's wording is precisely the miss that produces a duplicate. Proposing an id that is already taken, for work whose ticket is already open, is a failed teardown regardless of the analysis quality.
   - **Proactive Ticket Proposal (only when the pre-flight found nothing)**: When a mechanism donor is extracted or adoption is recommended **and the Board Pre-Flight reached "No ticket exists"**, the agent MUST formulate a ready-to-file ticket proposal directly in the chat synthesis (Repo, `<STREAM>-<NUM>` title, labels, problem, proposed scope, acceptance criteria) and ask the user for confirmation to open it. When the pre-flight found a ticket, the deliverable is that ticket — not a proposal, and not a proposal plus a note.
   - If an operational pipeline action is identified (e.g. applying to a network, submitting a profile, running an experiment when entering a project phase): add a concrete task to the target project's `roadmap.md` or `sources-inbox.md`.
4. **Sovereign Stack Alignment**: Does it respect our foundational doctrine: Zero AI attribution, Zero manual ops, Frontmatter Law, JIT secrets isolation (ADR-028), one writer per worktree with external-sibling placement (dotfiles `AGENTS.md` Standing Order #9, `using-git-worktrees`, `runbook-worktree-safety` — **not** ADR-032, which is cross-harness agent orchestration), and progressive disclosure?
5. **The Vault as SSOT for Domain Auto-Discovery (NEVER Default Lazily to `dotfiles`)**:
   - **Doctrinal Invariant**: The Knowledge Vault is the canonical Single Source of Truth ([[00_meta/_ssot|_ssot.md]], [[00_meta/patterns/pattern-knowledge-placement|pattern-knowledge-placement]]). Project scopes and domain boundaries are **canonically defined inside the vault**, never inferred from ad-hoc filesystem folders, git clones, or hardcoded skill lists.
   - `dotfiles` is **NOT** the default sink for all research. It belongs ONLY to developer workstation OS, shell environments (zsh, tmux), Go CLI (`dotf`), git worktree workflows, and local harness orchestration.
   - **Dynamic SSOT Discovery Protocol**:
     1. **Query the Vault SSOT**: Agents must query the live vault registry before assigning any destination:
        - *Hive-first access*: Call `vault_query` for documents with `type: project` (or `vault_list` on `10_projects/`).
        - *Filesystem fallback*: Inspect `10_projects/*/context.md` (plus zone anchors in `30_career/`, `20_certifications/`, `50_work/`).
        - Extract from each project's `context.md`: `id`, frontmatter `tags`, declared stack, and strategic scope.
     2. **Semantic Affinity Matching Against Vault SSOT**:
        - If the candidate topic matches the scope and tags of an active project in the vault SSOT (e.g. ATS scrapers/CV builders -> discovered `resume`; Kubernetes/homelab -> discovered `kubelab`; autonomous agent broker -> discovered `iris`; camera sensors -> discovered `imagingsuite`; personal portfolio/blog -> discovered `web`; shell/workstation CLI -> discovered `dotfiles`):
          - Route the teardown report to `10_projects/<discovered-project>/research/YYYY-MM-DD-<topic>-<model-token>.md` — one naming rule, stated once in full under Parallel Multi-Model Benchmarking below. A `-synthesis.md` is a later pass, never the first one and never this pass's filename.
          - Route follow-up tickets, mechanisms, and tasks to that project's issue tracker or `roadmap.md`.
        - If the topic aligns with personal career strategy, compensation, or job search: route to `30_career/`.
        - If the topic represents a **completely new domain / product venture** not registered in the vault SSOT:
          - Flag it explicitly as a **New Venture / Greenfield Opportunity**.
          - Propose scaffolding a new project entry (`10_projects/<slug>/context.md`) following [[00_meta/patterns/pattern-project-structure|pattern-project-structure]] or place as a foundational prestudy in `00_meta/research/` pending user initiation. Never contaminate an unrelated project.
        - If cross-cutting universal architecture or meta-system doctrine: route to `00_meta/research/`.
   - **Language Discipline**: Conversation with the user is in **Spanish** (or user preference); all durable vault artifacts, filenames, and commit messages are in **English only**.

---

## Dual-Engine Evaluation Framework

Every comprehensive evaluation covers two synchronized engines:

### Engine A: Deep Technical & Engineering Rigor (5 Angles)

1. **Paradigm & Invariants**: Declarative vs imperative, stateful vs stateless, push vs pull, client-orchestrated vs server-orchestrated.
2. **Architecture & Internal Mechanics**: Language, runtime, dependency footprint, IPC/protocols (REST, gRPC, WebSocket, NATS, loopback proxy), state storage (SQLite, KV, filesystem).
3. **Operational Profile & Resource Costs**: Self-hosted vs cloud-managed, compute/RAM footprint, token burn rate, licensing (MIT, Apache 2.0, BSL).
4. **Security, Governance & Secrets**: Secrets isolation (JIT injection vs ambient env vars), telemetry/egress, transcript sanitization, sandboxing (user privileges, rootless containers, UFW firewall).
5. **Adversarial Red-Team & Failure Modes**: Search GitHub issues, Reddit, HN. Where does it break under enterprise pressure, rate limits, network drops, or scale?

---

### Engine B: Product, Market & Business Lens

1. **Jobs To Be Done (JTBD) & User Journey**:
   - What is the real, emotional and functional job the user is hiring this product to do?
   - What is the "Magic Moment" in the user experience?
   - Where is the onboarding friction and cognitive load?
2. **Business Model & Monetization Mechanics**:
   - How does it make money or build enterprise value? (SaaS subscription, open-core, usage-based tokens, affiliate/referrals, consulting funnel, brand building).
   - Unit economics and margin profile: High fixed costs vs zero marginal cost.
3. **Category & Market Positioning**:
   - What category does it claim vs what category does it actually compete in?
   - What are the customer's true alternatives (DIY shell scripts, Excel, enterprise SaaS at $500/mo, manual labor)?
4. **The Advisory Council of Masters**:
   Evaluate the target through the specific lenses of recognized product and sales masters:

| Master / Referente | Core Mental Model | Evaluative Question for this Target |
|:---|:---|:---|
| **Alex Hormozi** | $100M Offers & Value Equation | Is there a Grand Slam Offer? (Dream Outcome × Likelihood / Time Delay × Effort). How is it packaged, priced, and de-risked? |
| **April Dunford** | Strategic Positioning | What is the category? What are the competitive alternatives? What unique value does only this deliver to which specific segment? |
| **Marty Cagan** | The 4 Big Product Risks | Where are the **Value Risk** (will they buy/use?), **Usability Risk**, **Feasibility Risk**, and **Business Viability Risk**? |
| **Rob Walling** | Stair-Step Bootstrapping | Can this generate immediate cash flow before multi-year SaaS? What is the step-1 product vs step-3 platform? |
| **Pieter Levels** | Solo Founder Velocity | Is it overengineered? Can an ugly MVP ship in days? Where is the programmatic SEO / public virality? |
| **Naval Ravikant** | Permissionless Leverage | Where is the specific knowledge? Does it leverage code and media with zero marginal cost of reproduction? |
| **Justin Welsh** | Distribution & Content Loops | What is the distribution engine? Does the product distribute itself, or does it depend on personal brand / outbound sales? |
| **Larry Kendall** | Ninja Selling (Value First) | Does it deliver unconditional, high-trust value upfront before asking for the sale or sign-up? |

---

## Portfolio Fit & Strategic Opportunity

Map the findings against our sovereign portfolio dynamically discovered via `10_projects/*/context.md`.

1. **Donor Classification**:
   - **Architecture Donor**: Solves an unsolved structural problem in an existing or new project; warrants adopting its architectural blueprint.
   - **Mechanism Donor**: Overall architecture is superseded, but it contains 1-2 brilliant micro-mechanisms or UX patterns to extract into a discovered project.
   - **Incompatible / Anti-Pattern**: Fundamentally conflicts with our core doctrine or adds unnecessary baggage for the target domain.
   - **Reference / Mental Model**: Valuable conceptual pattern; zero code.
2. **Product Opportunity for Us**:
   - Does this validate an unmet market demand that an existing project or a new greenfield project could solve better, cleaner, or more securely?
   - What can we build, productize, or avoid?

---

## Parallel Multi-Model Benchmarking

**Pre-flight, before writing anything.** Two cheap checks, both mandatory:

1. **Is a pass already here?** Read the target's `research/` directory **and** its `_index.md` — in practice the directory and the index disagree, so reading one is not reading the other. If a pass on this target already exists, the default deliverable is an **independent evaluation of it or a verification of it**, not a near-duplicate.
2. **Is the ticket already open?** See Board Pre-Flight under Core Principle 3.

**The naming rule, stated once — this is its SSOT.** Every evaluation pass is `10_projects/<project>/research/YYYY-MM-DD-<topic>-<model-token>.md`.

**`<model-token>` is provider-qualified and path-safe; `Evaluator:` carries the exact runtime.** They are two encodings of one fact, and collapsing them is a defect in either direction:

- **`<model-token>`** — `<PI_PROVIDER>-<PI_MODEL>` with every `/` replaced by `-`. Provider `nan` + model `deepseek-v4-flash` → `nan-deepseek-v4-flash`. The provider is the segment before the first `-`, so the token is reversible and two providers serving one model name cannot collide. A `/` in a filename is a path separator: using the runtime verbatim writes the file into a subdirectory that does not exist.
- **`Evaluator:`** — the exact runtime as the harness declares it, e.g. `nan/deepseek-v4-flash`. Never abbreviated, never a family name.

Both are **resolved from the harness, never self-described**: read `PI_PROVIDER` / `PI_MODEL` (under pi) or the local equivalent. A model cannot observe its own identity, so anything it writes about itself is a guess. This is not tidiness: `harness/reviewer-pool.json` is an allow-list of models permitted to sign a `review.md`, and `dotfiles#1907` binds reviewer identity to runtime provenance — a provenance field that can be invented is not a provenance field. When the runtime cannot be resolved, write `unresolved` plus the reason in `Evaluator:` and `unresolved` as the token; never guess a family or a vendor.

**The synthesis is a different pass, and it has a precondition.** `YYYY-MM-DD-<topic>-synthesis.md` merges consensus and the union of adversarial and business findings across passes, and it may be authored **only once the set of passes is closed and every pass read**. A `-synthesis.md` carries no `<model>` token (it merges several) and is never the artifact of a single evaluation run. Declaring a synthesis early closes the set by implication and silently promotes one model's pass into the repository's voice.

---

## Output Formats

### 1. In-Chat Synthesis (Spanish / User Language)
- **Resumen Ejecutivo**: Problema real, JTBD y veredicto.
- **Radiografía Técnica**: Puntos clave de arquitectura, seguridad y fallos ocultos.
- **Análisis de Producto & Negocio**: UX, modelo de monetización y fricción.
- **Tabla del Consejo de Referentes**: Qué dirían Hormozi, Dunford, Cagan, Walling, Levels, Naval, Welsh y Kendall.
- **Oportunidad Estratégica para Nuestro Portfolio**: Qué aprendemos comparado contra el portfolio vivo descubierto en el Vault.
- **Propuesta de Ticket(s) Lista para Abrir (Obligatorio en Adopt / Extract Mechanism)**:
  - Repositorio de destino y Stream ID: e.g. `mlorentedev/kubelab`, `CI-RUNNER-002: ...`
  - Labels recomendadas (e.g. `feature`, `infra`, `ci`)
  - Problema / Contexto, Alcance propuesto y Criterios de Aceptación
  - Pregunta directa al usuario: *"¿Quieres que abra este ticket en <repo> ahora mismo?"*
- **Descarte Justificado (en caso de Pass/Reject)**: Razones técnicas y económicas explícitas para no re-estudiarlo en vano.

### 2. Vault Report Template (English — `type: analysis`)

```markdown
---
id: <topic>-analysis-<model-token>
type: analysis
status: active
created: "YYYY-MM-DD"
owner: manu
tags: [research, <project>, <topic>, product, business, <keywords>]
---

# <Topic / Product Name> — Technical, Product & Business Analysis (<exact runtime, e.g. nan/deepseek-v4-flash>)

> **Source:** Web & codebase inspection of `<URL / Repo>` (inspected YYYY-MM-DD).
> **Evaluator:** <the exact provider-qualified runtime, e.g. `nan/deepseek-v4-flash` — read from the harness environment, never self-described>
> **Target Scope:** <10_projects/<project> or 00_meta>
> **Classification:** **<Architecture Donor | Mechanism Donor | Incompatible | Reference>** — <Rationale>.

---

## 1. Executive Summary & Paradigm Comparison

| Dimension | <Candidate> | Sovereign Stack Baseline |
| :--- | :--- | :--- |
| **Core Paradigm** | ... | ... |
| **Value Invariant** | ... | ... |
| **Trust & Security Model** | ... | ... |

---

## 2. Granular Architectural & Technical Analysis (Engine A)

### Plane 1: Delivery, Footprint & Dependencies
- ...

### Plane 2: IPC, Protocols & Loopback Routing
- ...

### Plane 3: Concurrency, State & Rate Limiting
- ...

### Plane 4: Security, Secrets & Sandboxing
- ...

### Plane 5: Adversarial Red-Team & Production Failure Modes
- ...

---

## 3. Product, Market & Business Analysis (Engine B)

### 3.1 Jobs To Be Done & User Experience
- **Functional & Emotional JTBD**: ...
- **The "Magic Moment"**: ...
- **Onboarding & Workflow Friction**: ...

### 3.2 Business Model & Monetization Dynamics
- **Revenue Mechanics**: (SaaS, open-core, affiliate, consulting hook).
- **Unit Economics & Moat**: ...

### 3.3 The Advisory Council of Masters (Referentes)

| Referente / Master | Evaluative Lens | Verdict on this Target |
|:---|:---|:---|
| **Alex Hormozi** | Offer & Value Equation | ... |
| **April Dunford** | Positioning & Alternatives | ... |
| **Marty Cagan** | The 4 Big Product Risks | ... |
| **Rob Walling** | Bootstrapping & Stair-Step | ... |
| **Pieter Levels** | Solo Velocity & Scrappiness | ... |
| **Naval Ravikant** | Leverage & Specific Knowledge | ... |
| **Justin Welsh** | Distribution & Content Loops | ... |
| **Larry Kendall** | Ninja Selling (Value First) | ... |

---

## 4. Portfolio Fit & Strategic Opportunity

- **Ecosystem Fit (`dotfiles`, `knowledge`, `hive`, `iris`, `kubelab`)**: ...
- **Mechanism Donors to Extract**:
  1. ...
- **Strategic Product Opportunity for Our Own Portfolio**: ...

---

## 5. Actionable Roadmap & Triage Decision

- **Verdict**: <ADOPT | EXTRACT MECHANISM | QUEUE TASK | PASS | HOLD>
- **Target Project**: <e.g. 10_projects/dotfiles, 10_projects/resume, 10_projects/iris>
- **Ticket / Task Link**: <e.g. `dotfiles#2002` CLI-097, `resume/roadmap.md` line X, or stated rejection rationale>
- **Execution Criteria**: <Immediate backlog, sprint milestone, or phase trigger>
```
