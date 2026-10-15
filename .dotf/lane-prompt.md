Carril de fixes de dotfiles: bugs que descubren otros proyectos, sin bloquear los demás carriles. Regla: un bug = una rama fix/<slug> = un worktree = una PR. Nunca acumules bugs en una rama.

Tu bug: #{{.Number}} {{.Title}}. Este worktree ya existe, en la rama {{.Branch}}, creado desde origin/main.{{if .Remaining}} Detrás en la cola:{{range .Remaining}} #{{.}}{{end}}.{{end}} Si otro proyecto reporta un bug, entra en la cola (.dotf/lane.json) por prioridad.

Por bug: reproduce, escribe el test que falla, arregla, aplica mutación, `go build/vet/test`, `GOOS=windows go vet`, golangci-lint (el pin). Asígnate el issue al empezar. PR con label skip-sdd, "## SDD skip rationale" y "## Knowledge" (lesson nueva en docs/lessons, regenerada con `dotf lessons fmt`; coge el siguiente número libre mirando origin/main y las PRs abiertas). Draft, ready, triaje, `dotf pr land`.

Primer push de una rama sin PR: `SDD_LABELS=skip-sdd SDD_PR_BODY=... git push` (el spec-gate solo lee la PR cuando existe).

Cola de PRs entre sesiones: aterriza una a la vez y solo la cabeza de la cola actualiza su rama (dotf pr land rechaza BEHIND). Compruébalo con `gh pr list --state open`. Permiso de merge para PRs propias: coordinado con las otras sesiones, CI verde, sin conflictos y comentarios de reviewers atendidos. No toques las ramas ni las PRs de otros carriles.

Al aterrizar NO borres tu worktree (cierra tu pane y tu sesión). Haz /handoff, ejecuta `dotf lane next` para arrancar al sucesor y para.

Lee AGENTS.md antes de la primera herramienta. Responde en español; todo artefacto durable en inglés.
