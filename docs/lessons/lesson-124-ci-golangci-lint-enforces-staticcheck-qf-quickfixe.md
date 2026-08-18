---
id: lesson-124-ci-golangci-lint-enforces-staticcheck-qf-quickfixe
type: lesson
status: active
created: "2026-06-24"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 124: CI golangci-lint enforces staticcheck QF* quickfixes a stale local version skips — heed the gopls hints

**Context**: Two PRs in the CLI-025 chain passed locally (`golangci-lint run` exit 0) but failed the CI `lint` job: an `errcheck` on an unchecked `fmt.Fprint`, and `QF1002` ("could use tagged switch") on a `switch { case x == "": … }`.

**Problem**: The CI action pins golangci-lint **v2.12.2**, which runs the staticcheck `QF*` (quickfix) category; the older binary on the dev machine did not flag them. The editor's `gopls` analyzer DID surface `QF1002` as a hint — but a hint reads as style noise, so it shipped and only CI caught it.

**Solution**: Treat gopls `QF*`/style hints as CI-enforced, not advisory — clear them before pushing. When practical, match CI's golangci-lint version locally; otherwise the cheap habit is: when the editor underlines a staticcheck quickfix, apply it rather than dismiss it.

---
