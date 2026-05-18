---
id: "AI-011-opencode-bootstrap"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-16"
tags: [spec, proposal]
template_version: "1.0"
---

# AI-011: Opencode bootstrap (+ AI-013 fold-in: AGENTS.md canonical migration)

> **Naming**: file lives at `<repo>/specs/<feature-id>/proposal.md`. `<feature-id>` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 10_projects/dotfiles/11-tasks.md: AI-011-opencode-bootstrap: opencode.jsonc (Ollama + OpenRouter providers + MCP server registry mirroring Claude's 5 servers), AGENTS.md at repo root, alias oc. Depends on AI-010. -->

Reemplazar aider con OpenCode como secondary AI coding agent del entorno, alineado con ADR-009 (`AGENTS.md` SSOT cross-agent). Aider depende de OpenRouter pay-as-you-go (~$2/mes variable) y Python 3.12 (audioop removed in 3.13, deuda futura). OpenCode + suscripción Go ($10/mes fija, sin auto-recarga) elimina la dependencia de runtime Python para el agente coding, ofrece presupuesto predecible y un catálogo curado de modelos open-weight (Kimi K2.6, DeepSeek V4 Pro, Qwen3.6 Plus, GLM-5.1, MiniMax M2.7) suficiente para el 80-90% del trabajo diario. Claude Code (plan Max x5) sigue siendo el frontier para tareas exigentes. Si no se hace, se mantiene la deuda técnica de Python 3.12, el presupuesto OpenRouter variable, y dos herramientas con perfil de uso solapado compitiendo en la misma franja.

## What

Tras este PR:

- Binary `opencode` instalado en `$HOME/.opencode/bin/` y disponible en PATH tras `setup-linux.sh`.
- `ai/opencode/opencode.jsonc` deployed a `$HOME/.config/opencode/opencode.jsonc` con:
  - Provider `opencode-zen` configurado para **suscripción Go** (catálogo open-weight únicamente, lista de modelos restringida explícitamente para evitar selección accidental de frontier que dispare facturación PAYG).
  - Modelos seleccionables del catálogo Go: `deepseek-v4-pro` (default) y `kimi-k2.6` — el usuario A/B testeará ambos durante el primer mes de uso.
  - Provider `openrouter` adicional para consumo del crédito $5 existente + frontier on-demand sin auto-recarga (autodetecta `OPENROUTER_API_KEY` ya cargado por `load-secrets.sh`).
  - Slot comentado para provider Ollama (placeholder hasta AI-010 — sin código activo en este PR).
  - Mirror de los 5 MCP servers de `mcp-servers.json` (SSOT cross-OS).
- `AGENTS.md` en raíz del repo como **SSOT canonical cross-agent** (Standing Orders, Decision Hierarchy, MCP rules portables, SDD snippet, Operational Rules). Migración completa de contenido portable desde `ai/claude/CLAUDE.md` y `ai/gemini/GEMINI.md` consolidando lo mejor de ambos. **Fold-in de AI-013-copilot-instructions-refresh** — este PR cierra ambos specs.
- `ai/claude/CLAUDE.md`, `ai/gemini/GEMINI.md`, `ai/copilot/copilot-instructions.md`, `.github/copilot-instructions.md` shrunk a pointers que delegan a `AGENTS.md`, conservando solo extensiones agent-specific (Claude: auto-maintenance/claude-mem/skills/TaskCreate; Gemini: meta-instruction + multimodality + sub-agents; Copilot: interaction style + POSIX focus).
- **Bug fixes encontrados al refactorizar (zero-debt rule)**: en ambos `copilot-instructions.md` se sustituye la cadena rota `the knowledge base (\`~/Projects/knowledge/\` or \`%USERPROFILE%\Apps\knowledge\\\`)` (template-placeholder no resuelto + ruta `Apps` incorrecta) por `~/Projects/knowledge/` (Linux) / `%USERPROFILE%\Projects\knowledge\` (Windows) consistente con CLAUDE.md.
- Alias `oc='opencode'` en `.zsh/aliases.zsh`.
- `scripts/healthcheck.sh` con nueva sección OpenCode (binary + config deployed + versión reportada).
- Tests `tests/opencode.bats` cubren install block, alias, config deployment, idempotencia, y regresión "aider block intact" (PR1 mantiene coexistencia).

## Out of scope

- Eliminación de aider (PR separada — atomic PRs <300 líneas).
- Provider Ollama activo (depende de AI-010-ollama-native).
- Skills-to-commands port (AI-012-opencode-commands-port).
- Setup Windows (`setup-windows.ps1`) — el sprint Windows sigue su propio orden.
- Auth secret automation: la API key de Zen/Go se introduce vía `/connect` en primera ejecución (paso manual documentado en runbook). Encriptación con age + `sensitive/opencode_zen.secret.age` queda para follow-up cuando el formato de `auth.json` se estabilice (opencode aún beta).
- Lock-file en Hive MCP server (auto-commit a git). Cambio en repo `hive`, no en `dotfiles`. Follow-up post-merge.
- A/B comparison Kimi K2.6 vs DeepSeek V4 Pro — conversación aparte tras 2-4 semanas de uso real.

## Risks / open questions

- **Versión opencode beta** — el binario sigue en pre-1.0 y el formato de `opencode.jsonc` puede cambiar. Mitigación: declarar `$schema` URL en el config + smoke test de versión en healthcheck. *Aceptado.*
- **`/connect` flow es manual e interactivo** — no automatizable en `setup-linux.sh` sin guardar API key plain. *Aceptado*: runbook documenta el paso manual; binary y config sí son automáticos. Re-evaluar cuando opencode añada env var equivalente para Zen/Go.
- **MCP server conflict con Claude Code** — *Resuelto 2026-05-16*: mirror completo de los 5 MCP servers en `opencode.jsonc`. La solución estructural es un lock-file en el auto-commit a git de Hive MCP server — **cambio aparte en repo hive, follow-up no bloqueante a este PR**. Hasta que aterrice, el runbook documenta como mitigación temporal: no ejecutar `oc` y `claude` en paralelo sobre el mismo repo.
- **Zen PAYG auto-recarga oculta** — si Zen tiene método de pago conectado y se llama a modelos fuera del catálogo Go (Sonnet/GPT-5/Opus/Gemini Pro), dispararía billing PAYG con auto-recarga $20 si balance < $5. **Mitigación en 3 capas (defensa en profundidad):**
  1. **Config-level** (en `opencode.json`): lista explícita de modelos = catálogo Go bajo el provider `opencode-zen`; el `/models` picker no muestra frontier de Zen.
  2. **Workspace-level** (en Zen dashboard, documentado en runbook): set workspace PAYG cap to $0 si está disponible para cuentas individuales.
  3. **Payment-level** (en Zen dashboard, documentado en runbook): no conectar payment method al workspace de PAYG; solo el de la suscripción Go. Sin card vinculada, las llamadas a frontier *fallan* en lugar de facturar.
  Si necesita frontier puntual → provider `openrouter` (capped a balance $5) o Claude Code Max.
- **Plan Go cap mensual de $60 de valor consumido** — improbable agotarlo en uso personal. Solo observación primer mes. Nota: $60 es el techo de uso, no de facturación; el cargo sigue siendo $10/mes fijo.

## Acceptance criteria

- [ ] Tras `./setup-linux.sh` en clean install: `command -v opencode` resuelve a `$HOME/.opencode/bin/opencode` y `opencode --version` reporta versión sin error.
- [ ] Segunda ejecución de `./setup-linux.sh` (idempotencia): no re-descarga binario, sin diff en `$HOME/.config/opencode/opencode.jsonc`, cero `2>/dev/null || true` introducidos por este PR.
- [ ] `source ~/.zshrc && command -v oc` resuelve a `opencode`.
- [ ] `./scripts/healthcheck.sh` incluye sección OpenCode con verificación de binary + config deployed + versión.
- [ ] `~/.local/bin/bats tests/opencode.bats` pasa todos los tests (relacionales, no solo de presencia per pattern-setup-script-idempotence).
- [ ] `AGENTS.md` en raíz del repo legible por OpenCode al lanzar `oc` sobre el repo.
- [ ] Runbook `10_projects/dotfiles/40-runbooks/guide-opencode-go-setup.md` publicado con flujo `/connect` documentado.
- [ ] `opencode` lista al menos 2 providers configurados (`opencode-zen` + `openrouter`) tras `setup-linux.sh`.
- [ ] `opencode` en el TUI lista exclusivamente modelos del catálogo Go bajo el provider `opencode-zen` (no aparecen Sonnet/GPT-5/Opus/Gemini Pro). Frontier solo accesible vía provider `openrouter`.
- [ ] `tests/opencode.bats` incluye test de regresión asegurando que el bloque aider en `setup-linux.sh` sigue intacto (PR1 mantiene coexistencia).
- [ ] `opencode.jsonc` verificado contra `.gitattributes` — sin sufrir CRLF en deploy cross-OS.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (backlog entry)
- Related ADR: `10_projects/dotfiles/30-architecture/adr-009-multi-agent-runtime.md`
- Related patterns: `00_meta/patterns/pattern-setup-script-idempotence.md`, `00_meta/patterns/pattern-spec-driven-development.md`
- Sister spec (post-merge): PR2 — aider sunset + ADR-009 amend
- Follow-up tracked: lock-file en Hive MCP auto-commit (repo hive)
- External: [opencode.ai docs](https://opencode.ai/docs/), [opencode Go subscription](https://opencode.ai/go)
