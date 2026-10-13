---
id: "lesson-400-a-fixture-in-a-shape-the-tool-never-returns"
type: lesson
status: active
title: "A fixture in a shape the tool never returns tests nothing"
created: "2026-10-13"
---

# A fixture in a shape the tool never returns tests nothing

## Context
`knowledge-gate` and `spec-gate` skip a pull request that carries the `dependencies` label
and is authored by a dependency bot. The bot list holds `dependabot[bot]`, the login the REST
API and webhooks use. The gates get the author from `scripts/spec-gate-pr.sh`, which reads
it with `gh pr view --json author`.

## The Trap
For a GitHub App, `gh` returns the author as `{"is_bot": true, "login": "app/dependabot"}`,
not `dependabot[bot]`. The skip never matched a real Dependabot PR. Its tests passed anyway,
because the adapter's stub `gh` answered with the REST login. The fixture was written from the
shape the gate wanted, not from the shape `gh` produces. A second defect hid behind the
first: the `dependencies` label that `dependabot.yml` asks for did not exist in the repo, and
Dependabot drops labels that do not exist. Every Dependabot PR went red on `knowledge-gate`,
and #2010 got through only because someone added a Knowledge section to its body by hand.

## The Solution
The adapter normalises `app/<slug>` to `<slug>[bot]`, and only when `gh` also reports
`is_bot`, so a human login passes through verbatim. A test feeds the adapter the JSON `gh`
actually returned for #2348, and it fails on the old adapter. The label now exists.

## Takeaways
- Capture a fixture from the tool's real output, and cite where it came from. A fixture
  typed from the consumer's expectations proves only that the consumer agrees with itself.
- When a gate grants an exemption, test the exemption against a real case once. An exemption
  that never fires is indistinguishable from one that is never needed.
