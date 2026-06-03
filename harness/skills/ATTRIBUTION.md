# Vendored skill attribution (SKILLS-001)

Some skills in the cross-agent skill pipeline are **vendored** (adapted) from external,
permissively-licensed open-source repositories. The editable source of truth lives in the
vault (`00_meta/skills/<name>/SKILL.md`); these `harness/skills/<name>/` records are the
committed, generated snapshots (see the repo README / ADR-013 "generate-and-commit").

For each vendored skill we vendor **only the `SKILL.md`** (adapted for our pipeline and
trimmed of broken sibling/reference links). The upstream `references/` deep-dive files are
**not** copied — consult the source repo for those. Full license texts live in each source
repository; this file is the NOTICE-equivalent that preserves attribution as the licenses require.

| Skill | Source repo | Upstream skill | License | Copyright |
|-------|-------------|----------------|---------|-----------|
| `terraform` | [antonbabenko/terraform-skill](https://github.com/antonbabenko/terraform-skill) | terraform-skill | Apache-2.0 | © 2026 Anton Babenko |
| `helm` | [laurigates/claude-plugins](https://github.com/laurigates/claude-plugins) | helm-chart-development | MIT | © 2026 Lauri Gates |
| `mcp-builder` | [anthropics/skills](https://github.com/anthropics/skills) | mcp-builder | Apache-2.0 | © 2026 Anthropic, PBC |
| `golang-pro` | [jeffallan/claude-skills](https://github.com/jeffallan/claude-skills) | golang-pro | MIT | © 2025 Jeff Allan |
| `async-python-patterns` | [wshobson/agents](https://github.com/wshobson/agents) | async-python-patterns | MIT | © 2024 Seth Hobson |

All sources verified by reading the repository `LICENSE` file at vendor time (2026-06-03).
Apache-2.0 and MIT both permit redistribution with attribution; this file plus the per-skill
footer satisfy the notice requirement. If a skill is updated upstream, re-vendor the `SKILL.md`
into the vault source and re-run `compile-harness.sh --refresh`.
