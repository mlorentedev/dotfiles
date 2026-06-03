---
name: mcp-builder
description: Guide for creating high-quality MCP (Model Context Protocol) servers that let LLMs interact with external services through well-designed tools. Use when building MCP servers to integrate external APIs or services, in Python (FastMCP) or Node/TypeScript (MCP SDK).
source: https://github.com/anthropics/skills (mcp-builder)
license: Apache-2.0
---

# MCP Server Development Guide

Create MCP servers that enable LLMs to interact with external services through well-designed tools. Quality is measured by how well the server enables LLMs to accomplish real-world tasks.

## High-level workflow (4 phases)

### Phase 1 — Research & planning
- **API coverage vs workflow tools:** balance comprehensive endpoint coverage with specialized workflow tools. When uncertain, prioritize comprehensive API coverage.
- **Naming & discoverability:** clear, action-oriented names with consistent prefixes (`github_create_issue`, `github_list_repos`).
- **Context management:** concise tool descriptions; support filtering/pagination; return focused, relevant data.
- **Actionable errors:** guide the agent toward a fix with specific next steps.
- **Study the protocol:** start at `https://modelcontextprotocol.io/sitemap.xml`, fetch pages with a `.md` suffix for markdown. Review architecture, transports (streamable HTTP, stdio), and tool/resource/prompt definitions.
- **Recommended stack:** TypeScript SDK (strong typing, good model support); streamable HTTP + stateless JSON for remote servers, stdio for local.

### Phase 2 — Implementation
- **Project structure:** per the language guide (TS: package.json/tsconfig; Python: module layout + deps).
- **Core infrastructure:** authenticated API client, error-handling helpers, response formatting (JSON/Markdown), pagination.
- **Per tool:**
  - *Input schema* — Zod (TS) or Pydantic (Python); constraints + clear descriptions + examples.
  - *Output schema* — define `outputSchema` where possible; return `structuredContent`.
  - *Implementation* — async I/O, actionable errors, pagination; return both text and structured data.
  - *Annotations* — `readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`.

### Phase 3 — Review & test
- Review for DRY, consistent error handling, full type coverage, clear descriptions.
- **TypeScript:** `npm run build`; test with `npx @modelcontextprotocol/inspector`.
- **Python:** `python -m py_compile your_server.py`; test with MCP Inspector.

### Phase 4 — Evaluations
Create ~10 evaluation questions that are **independent, read-only, complex (multi-tool), realistic, verifiable (single string-comparable answer), and stable**. Process: inspect tools → explore data (read-only) → generate questions → verify answers yourself. Output as XML:

```xml
<evaluation>
  <qa_pair>
    <question>...complex, realistic, multi-tool question...</question>
    <answer>3</answer>
  </qa_pair>
</evaluation>
```

## Key SDK references (fetch on demand)
- Python SDK: `https://raw.githubusercontent.com/modelcontextprotocol/python-sdk/main/README.md`
- TypeScript SDK: `https://raw.githubusercontent.com/modelcontextprotocol/typescript-sdk/main/README.md`

The upstream skill bundles four reference files (Python/FastMCP guide, Node/TS guide, MCP best practices, evaluation harness with runnable scripts) — consult them for complete worked examples.

---
*Vendored from [anthropics/skills](https://github.com/anthropics/skills) `mcp-builder` (Apache-2.0, © 2026 Anthropic, PBC). Adapted for the cross-agent skill pipeline; the `reference/` + `scripts/` bundles remain upstream. See `harness/skills/ATTRIBUTION.md`.*
