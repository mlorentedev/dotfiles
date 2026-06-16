---
id: "{{family}}-{{component}}"
type: project
status: active
owner: manu
source_path:
  code: "${PROJECTS_PATH}/<ProductFamily>/<component>"
  onedrive: "${ONEDRIVE_PATH}/Products/<ProductFamily>"
stack: []
tags: [work, sdk, {{family}}]
created: "{{date}}"
---

# {{component}}: Work SDK Context

> **Goal:** [Describe this repo's purpose within the {{family}} product family]

## Technical Stack
- **Language:**
- **Key Tools:**

## Repos in this Family
See [[../00-context|{{family}} family context]] for all repos.

## Critical Links
- **Real repo path:** Fill in source_path above with actual ${PROJECTS_PATH}/... path

## Knowledge Structure
Per [[pattern-knowledge-placement]]: build/operate docs (ADRs, runbooks, troubleshooting, lessons) live in the repo `docs/`; this entry keeps only decide/personal layers.
- [[memory/MEMORY]]: Session memory

## Development Flow
1. Open Claude in the real repo (see source_path)
2. Session hook links this vault entry as MEMORY context
