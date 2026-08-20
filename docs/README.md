# Documentation

Project-bound knowledge for `dotfiles`, kept in-repo (docs-as-code). The *operate/build* layer lives here so it is versioned with the code and readable by any agent in-context — no external knowledge store required.

- [`architecture.md`](architecture.md) — **where does X live**: the normative repo tree, CI-guarded (`tests/architecture-md.bats`)
- [`adr/`](adr/) — Architecture Decision Records (the *why* behind decisions) and the repo audits (`audit-*.md`) / architecture map
- [`runbooks/`](runbooks/) — operational procedures (setup, secrets, tooling; indexed at [`runbooks/_index.md`](runbooks/_index.md))
- [`troubleshooting/`](troubleshooting/) — known issues and their fixes (indexed at [`troubleshooting/_index.md`](troubleshooting/_index.md))
- [`lessons/`](lessons/) — accumulated gotchas and post-mortems (indexed at [`lessons/_index.md`](lessons/_index.md); legacy stub: [`lessons.md`](lessons.md))
- [`secrets-inventory.md`](secrets-inventory.md) — dated secrets-migration snapshot (see its own banner for current status)

## I want to…

| …do this | Read |
|---|---|
| Understand where a file/directory belongs | [`architecture.md`](architecture.md) |
| Add or rotate a secret | [`runbooks/guide-secrets-governance.md`](runbooks/guide-secrets-governance.md) |
| Understand *why* a decision was made | [`adr/`](adr/) — find it by topic, newest ADR wins on conflict |
| Fix something that's broken | [`troubleshooting/`](troubleshooting/) — symptom → cause → fix |
| Avoid repeating a past mistake | [`lessons/`](lessons/) |
| Set up a new machine's AI tooling | [`runbooks/`](runbooks/) — see the runbook for the tool in question |
| Check whether a doc's claims are still trusted | the relevant `adr/audit-*.md` — each records what was verified and when |

The *decide/position* layer (roadmap, prestudy, strategy) and session memory live in the maintainer's cross-project knowledge store and are intentionally not committed here.
