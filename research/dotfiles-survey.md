# Dotfiles Survey — fmontes, holman, mathiasbynens

> Investigación comparativa contra los dotfiles propios. Fecha: 2026-05-25.
> Branch: `worktree-research+dotfiles-survey`.

## TL;DR

- Los 3 repos son **macOS-only**, **sin secretos cifrados**, **sin tests**, **sin AI tooling**, **sin SDD**. El usuario está claramente por delante en disciplina de ingeniería.
- Aun así, cada uno representa una **filosofía** distinta de la que se pueden extraer ideas concretas:
  - **fmontes** → bootstrap moderno, one-liner curl, fases numeradas, Brewfile como SSOT de paquetes.
  - **holman** → topical / convention-over-configuration (descubrimiento por sufijo de filename, cero edición central al añadir herramienta).
  - **mathiasbynens** → funciones bash refinadas tipo swiss-army, patrón `.extra` para overrides locales, `.macos` masivo.
- Ver § "Top 6 ideas a aplicar" abajo para la lista priorizada por ROI.

---

## Comparación at-a-glance

| Dimensión | fmontes | holman | mathiasbynens | **usuario (este repo)** |
|---|---|---|---|---|
| Plataforma | macOS-only | macOS-first | macOS-only | **Linux + Windows** |
| Bootstrap | `curl \| bash` → `01-...07-post-install.sh` | `script/bootstrap` + topical install.sh | `bootstrap.sh` (rsync) | `setup-linux.sh` / `setup-windows.ps1` |
| Idempotencia | parcial (solo symlinks) | sí (readlink check) | no (rsync ciego) | **sí + healthcheck** |
| Secretos | `.gitignore` + onboarding manual 1Password | `*.local.symlink` gitignored | `.extra` no-commiteado | **age + `env-mapping.conf` + loader runtime** |
| Tests | 0 | 0 | 0 | **147 bats + shellcheck + CI** |
| Healthcheck post-install | 0 | 0 | 0 | **`scripts/healthcheck.sh` cross-OS** |
| AI tooling versionado | instala binarios, cero configs | 0 (repo pre-LLM) | 0 | **Claude skills + opencode + aider + MCP + AGENTS.md** |
| Disciplina arquitectónica | numbered phases | topical/discovery | `.archivo` por concern | **SDD spec-gate + ADRs + audits + vault** |
| Edad / actividad | reciente | clásico 2014 | clásico mantenido | activo |

---

## Filosofías por repo

### fmontes/dotfiles — "factory reset to coding in 5 minutes"

Pequeño y moderno. Entry-point `curl -fsSL https://fmontes.com/install.sh | bash`, divide la instalación en 7 scripts numerados (`01-xcode`, `02-homebrew`, ..., `07-post-install`) ejecutados secuencialmente. `Brewfile` declarativo (31 brews + 10 casks: cursor, ghostty, raycast, orbstack, codex, ollama-app). `home/` mirror 1:1 de `$HOME` para symlinks. Instala Claude Code vía npm + Cursor + LM Studio como casks, pero **cero configs versionadas** para ellos.

### holman/dotfiles — "topical, convention-over-configuration"

El abuelo de los dotfiles modernos. 15 carpetas temáticas (`git/`, `homebrew/`, `zsh/`, ...). Cero registro central: el bootstrap descubre archivos por **sufijo**:

- `*.symlink` → symlink a `$HOME/.<basename>`
- `*.zsh` → cargado por `zshrc.symlink` con orden `path.zsh` primero, `completion.zsh` al final, resto en medio
- `install.sh` → ejecutado por `script/install`
- `bin/*` → al `$PATH`

Añadir herramienta = `mkdir foo && touch foo/aliases.zsh`. Cero edición del bootstrap. Tiene además un **prompt interactivo de resolución de colisiones** en symlinks (`[s]kip/[S]kip all/[o]verwrite/[O]verwrite all/[b]ackup/[B]ackup all`) que es UX excelente.

### mathiasbynens/dotfiles — "refinamiento artesanal"

El más viejo de los tres con mantenimiento más activo. `.bash_profile` itera sobre `~/.{path,bash_prompt,exports,aliases,functions,extra}` con guards `[ -r ] && [ -f ]`. Cada concern en su archivo:

- `.path` — `$PATH`
- `.exports` — env-vars
- `.aliases` — aliases
- `.functions` — funciones útiles
- `.extra` — overrides locales no-commiteados, cargado al final (puede pisar todo)

`.macos` con 1200+ líneas de `defaults write` cubriendo Finder, Dock, Spotlight, Safari, Mail, Hot corners, Time Machine, etc. Funciones tipo swiss-army en `.functions`: `mkd`, `targz` (elige zopfli/pigz/gzip por tamaño), `dataurl`, `server`, `getcertnames`.

---

## Top 6 ideas a aplicar (ranked por ROI)

### 1. Patrón `.extra` / `.local` para overrides no-commiteados — **Tier 1**

Ambos mathiasbynens (`.extra`) y holman (`gitconfig.local.symlink`) usan esto. Es el reemplazo más simple y limpio para hardcodear paths o credenciales personales en archivos versionados.

**Aplicación concreta**:

```bash
# Al FINAL de .zshrc y .bashrc, después de todo lo demás:
[ -r "$HOME/.zshrc.local" ] && [ -f "$HOME/.zshrc.local" ] && . "$HOME/.zshrc.local"
```

Y `.zshrc.local` en `.gitignore`. Ventaja: complementa (no reemplaza) al sistema age — age es para secretos cifrados versionables, `.local` es para tweaks de máquina (host-specific aliases, paths a `/mnt/...` solo en una VM, etc.) que no merecen entrar al repo.

Esfuerzo: ~10 min. Riesgo: bajo.

### 2. Funciones swiss-army de mathiasbynens — **Tier 1**

Pequeñas, autocontenidas, sin deps. Encajan limpiamente en `.zsh/aliases.zsh` o un nuevo `.zsh/functions.zsh`:

- `mkd()` — `mkdir -p` + `cd` en un comando
- `targz()` — elige compresor por tamaño (zopfli/pigz/gzip)
- `dataurl()` — file → `data:` URL base64
- `server()` — Python HTTP server + abre navegador
- `getcertnames()` — extrae CN + SANs de un cert SSL en `host:port`
- `gz()` — tamaño antes/después de gzipping para ver ratio

Esfuerzo: ~30 min. Riesgo: bajo. Compatibilidad: validar shellcheck + bats antes de commitear, respetar tabla "Prohibited Patterns" del CLAUDE.md de proyecto.

### 3. Loop de sourcing condicional — **Tier 1**

Reemplazar bloques estilo:

```bash
source ~/.zsh/aliases.zsh
source ~/.zsh/exports.zsh
source ~/.zsh/functions.zsh
```

Por el patrón mathiasbynens:

```bash
for f in $HOME/.zsh/{aliases,exports,functions,prompt,completions}.zsh; do
  [ -r "$f" ] && [ -f "$f" ] && . "$f"
done
unset f
```

Más limpio, tolera archivos opcionales, fácil añadir nuevos concerns. Esfuerzo: ~15 min.

### 4. Resolución interactiva de colisiones de symlinks (holman) — **Tier 2**

Cuando `setup-linux.sh` encuentra que un destino ya existe como archivo regular, hoy el comportamiento es heurístico. Adoptar el prompt de holman:

```
File already exists: ~/.zshrc, what do you want to do?
[s]kip, [S]kip all, [o]verwrite, [O]verwrite all, [b]ackup, [B]ackup all
```

Implementable en `scripts/utils.sh` como helper `prompt_collision()` reutilizable por `link_file()`. Esfuerzo: ~1-2 h. Riesgo: medio (UX cambia, tests bats necesitan modo `--force` no-interactivo para CI).

### 5. One-liner curl-install (fmontes) — **Tier 2**

Añadir un `install.sh` mínimo al repo que hace clone-or-update + delega a `setup-linux.sh`. Después servirlo desde un dominio propio (o `raw.githubusercontent.com`). Ventaja: factory reset a máquina nueva → un comando.

```bash
curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.sh | bash
```

Esfuerzo: ~30 min. Riesgo: bajo. Consideración de seguridad: documentar que el usuario debe inspeccionar antes de ejecutar `curl | bash` (standard caveat).

### 6. Patrón topical / discovery de holman — **Tier 3 (filosófico)**

El mayor cambio arquitectónico de los tres. Implicaría refactorizar `setup-linux.sh` monolítico a:

- Cada herramienta vive en su carpeta (`git/`, `tmux/`, `claude/`, `opencode/`, ...)
- Bootstrap descubre por glob: `find . -name install.sh`, `*.symlink`, `*.zsh`
- Añadir herramienta = crear carpeta, cero edición del bootstrap

**Veredicto**: probablemente **NO** vale la pena en este repo. Razones:
- El usuario ya tiene `ai/<agent>/` topical para configs de IA (que ES el patrón holman aplicado a su dominio).
- `scripts/` ya está bien categorizado (audit-005, 9 categorías funcionales documentadas).
- Cross-OS (Linux + Windows) complica el descubrimiento por sufijo: ¿`*.symlink` también en Windows? ¿`install.sh` y `install.ps1` en cada carpeta?
- El sistema SDD del usuario es la SSOT real, no la convención filename.

Si se quiere robar parcialmente: aplicar el patrón **solo a `ai/<agent>/`** que ya está topical de facto — añadir un descubrimiento automático que skip-hardcoded en `setup-linux.sh`. Eso sí es una ganancia incremental sensata (~3-4 h, candidato a SDD).

---

## Anti-ideas (NO copiar)

| Patrón | Por qué evitarlo |
|---|---|
| fmontes: bootstrap sin idempotencia real | El healthcheck + drift detector del usuario ya cubre esto mejor |
| holman: `*.symlink` glob con `find -maxdepth 2` | Frágil en repo con muchas subcarpetas; `setup-linux.sh` explícito es más auditable |
| mathiasbynens: rsync ciego del repo a `$HOME` | Pierde traza simbólica; `setup-linux.sh` usa symlinks y eso es superior |
| Todos: cero gestión de secretos real | El sistema age del usuario es objetivamente mejor; no degradar |
| Todos: cero tests | El usuario tiene 147 bats; no bajar la barra |

---

## Validación: cosas que el usuario YA hace mejor

Útil tener esto explícito para no segundo-adivinarse:

1. **Cross-OS Linux + Windows con paridad enforced** — los 3 repos son macOS-only sin pretensión de portabilidad. El SDD-004 (session-start-config SSOT) y audit-002 (cross-OS duplication) son disciplinas que ninguno tiene.
2. **Secretos cifrados con age + `env-mapping.conf` + loader runtime + file-secrets (`@VAR=file>dest`)** — los 3 repos tienen, como mucho, `.gitignore` + onboarding manual.
3. **Disciplina de testing**: 147 bats (bash+zsh), shellcheck, healthcheck, CI con spec-gate, drift detector. Los 3 repos: cero.
4. **AI tooling como ciudadano de primera**: Claude skills + AGENTS.md SSOT cross-agent + opencode go + aider tiers + MCP servers (hive, claude-mem, drawio, context7) + vault Obsidian. fmontes instala binarios pero no versiona configs. holman y mathiasbynens son pre-LLM.
5. **SDD spec discipline**: specs/<id>/ folder enforced por CI label `skip-sdd`, ADRs, audits, lecciones — los 3 repos no tienen nada equivalente.

---

## Posibles follow-ups SDD

Si alguna de estas ideas avanza a implementación, son candidatos a spec formal:

| Spec candidata | Scope estimado | LOC aprox |
|---|---|---|
| `IDEAS-001-local-override` | `.zshrc.local` + `.bashrc.local` cargados al final, `.gitignore` actualizado, test bats | ~30 LOC + 2 tests |
| `IDEAS-002-shell-functions` | Añadir `mkd`/`targz`/`dataurl`/`server`/`getcertnames`/`gz` en `.zsh/functions.zsh` (nuevo) + sourced loop | ~80 LOC + tests |
| `IDEAS-003-sourcing-loop` | Refactor de sourcing en `.zshrc`/`.bashrc` al patrón loop, guards `[ -r ]` | ~20 LOC, refactor con drift-detect check |
| `IDEAS-004-collision-prompt` | `prompt_collision()` en `utils.sh`, integrar en `link_file()`, flag `--force` para CI | ~60 LOC + bats con stdin-mock |
| `IDEAS-005-curl-bootstrap` | `install.sh` raíz con clone-or-update + README | ~40 LOC |
| `IDEAS-006-ai-topical-discovery` | Auto-descubrir `ai/<agent>/install.sh` en `setup-linux.sh` (parcial topical aplicado solo a `ai/`) | ~50 LOC, evaluar contra hardcoded |

Las tres primeras (Tier 1) probablemente NO necesitan SDD individual — son <50 LOC cada una. Podrían agruparse en un solo PR `chore: borrow refinements from holman/mathiasbynens` con `skip-sdd` label.

Las tres últimas (Tier 2/3) sí justifican spec formal cada una.

---

## Notas de método

- Investigación hecha vía 3 agentes paralelos `general-purpose` con WebFetch.
- Cero clonado local de los repos investigados.
- Reportes individuales archivados en el trace de la sesión Claude (no incluidos en este resumen).
- Próximo paso sugerido: si alguna idea entra a backlog, abrir spec o ticket en el vault `~/Projects/knowledge/10_projects/dotfiles/11-tasks.md`.
