---
id: "{{repo}}"
type: project
status: active
owner: manu
repo_url: ""
stack: [{{stack}}]
tags: []
created: "{{date}}"
---

# {{repo}}: Project Context

> **Goal:** [Concise, single-sentence project description]

## Technical Stack
- **Language:** {{stack}}
- **Frameworks:**
- **Infrastructure:**
- **Key Tools:**

## Critical Links
- **Repository:** [fill in]
- **Production:**

## Knowledge Structure
Per [[pattern-knowledge-placement]]: build/operate docs live in the **repo** (`docs/adr/`, `docs/runbooks/`, `docs/troubleshooting/`, `docs/lessons.md`, `specs/<id>/`); this store keeps only the decide/personal layers — [[10-roadmap]] (strategy) and [[memory/MEMORY]] (AI memory). Task state lives in the bitácora GitHub Project (ADR-018), not here.

## Development Flow
1. Feature branch from `main`
2. PR with tests
3. Merge -> deploy
