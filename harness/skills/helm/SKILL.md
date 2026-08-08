---
id: helm-skill
type: skill
status: active
created: "2026-06-02"
owner: manu
name: helm
description: Create, test, and package Helm charts — Chart.yaml, templates, _helpers.tpl, dependencies/subcharts, lint/template/dry-run, and repo/OCI publishing. Use when the user mentions Helm charts, helm lint/template/package, or Kubernetes packaging.
source: https://github.com/laurigates/claude-plugins (helm-chart-development)
license: MIT
---

# Helm Chart Development

Create, test, and package custom Helm charts with best practices for maintainability and reusability. (This skill covers chart *authoring*; release install/upgrade, multi-env values overrides, and deployment debugging are separate concerns.)

## Chart creation & structure

```bash
helm create mychart    # scaffolds Chart.yaml, values.yaml, charts/, templates/ (+ _helpers.tpl, tests/), .helmignore
```

**Chart.yaml essentials:**
```yaml
apiVersion: v2          # Helm 3
name: mychart
version: 0.1.0          # chart version (SemVer)
appVersion: "1.0.0"     # app version
type: application       # application | library
dependencies:
  - name: postgresql
    version: "12.1.9"
    repository: https://charts.bitnami.com/bitnami
    condition: postgresql.enabled
```

## Validation & testing

```bash
helm lint ./mychart --strict                                   # warnings as errors
helm template myrelease ./mychart --values values.yaml         # render locally
helm template myrelease ./mychart --show-only templates/deployment.yaml --validate
helm install myrelease ./mychart --dry-run --debug             # server-side validation
# chart tests (helm.sh/hook: test pods):
helm install myrelease ./mychart -n test && helm test myrelease -n test --logs && helm uninstall myrelease -n test
```

Chart test pod (`templates/tests/test-connection.yaml`) uses annotations `"helm.sh/hook": test` and `"helm.sh/hook-delete-policy": hook-succeeded,hook-failed`.

## Template helpers (`_helpers.tpl`)

```yaml
{{- define "mychart.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "mychart.labels" -}}
helm.sh/chart: {{ include "mychart.chart" . }}
{{ include "mychart.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
```

## Dependencies / subcharts

```bash
helm dependency update ./mychart    # download per Chart.yaml -> Chart.lock
helm dependency build ./mychart     # build from existing Chart.lock
```
Configure subchart values under the dependency's key in the parent `values.yaml` (e.g. `postgresql: { auth: {...}, primary: { persistence: { size: 10Gi } } }`). Use `condition:` to toggle and `file://../common-library` for local deps.

## Packaging & distribution

```bash
helm package ./mychart --dependency-update --destination ./dist/
helm repo index ./repo/
helm push mychart-0.1.0.tgz oci://registry.example.com/charts   # Helm 3.8+ OCI
```

## Agentic cheat-sheet

| Context | Command |
|---------|---------|
| Lint (strict) | `helm lint ./mychart --strict` |
| Render one template | `helm template myapp ./mychart --show-only templates/deployment.yaml` |
| Dry-run validation | `helm install myapp ./mychart --dry-run --debug 2>&1 \| head -100` |
| Package | `helm package ./mychart --dependency-update` |

References: [Helm charts](https://helm.sh/docs/topics/charts/) · [Best practices](https://helm.sh/docs/chart_best_practices/) · [Template guide](https://helm.sh/docs/chart_template_guide/) · [chart-testing](https://github.com/helm/chart-testing). The upstream skill bundles a deeper `REFERENCE.md` (values/schema/testing detail).

---
*Vendored from [laurigates/claude-plugins](https://github.com/laurigates/claude-plugins) `helm-chart-development` (MIT, © 2026 Lauri Gates). Adapted for the cross-agent skill pipeline; the companion `REFERENCE.md` remains upstream. See `harness/skills/ATTRIBUTION.md`.*
