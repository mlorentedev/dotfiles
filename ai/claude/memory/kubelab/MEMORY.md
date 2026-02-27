# KubeLab — Local Session Memory

> Machine-local cache. Durable knowledge lives in repo CLAUDE.md + vault.
> This file supplements CLAUDE.md with session-specific state and operational details.

## Product Portfolio Framework (2026-02-21)
- **4 layers**: L0 infra → L1 platform (kubelab-*) → L2 apps → L3 tools
- **Master index**: vault `10_projects/kubelab/portfolio.md`
- **11 products**: kubelab, kubelab-cli, kubelab-gateway, kubelab-memory, kubelab-agents, kubelab-console, imaging-suite, cubernautas, pollex, yt-intel, sec-scan
- **Lifecycle**: idea → spec → incubating → active → stable
- **Streams F, H, P**: extracted to product specs in vault (2026-02-21)
- **Priority**: L0 infra first. Products after staging is stable.

## Current State (2026-02-26)
- Branch: feature/blog-restruct
- Dev stack: 100% operational
- Cluster: 3 nodes Ready (k3s-server, agent-1, agent-2)
- **PROD-K3S-000f DONE**: prod.enc.yaml + prod.oidc-jwks.pem generated, 19 secrets in SOPS
- **ADR-014 accepted**: Secrets management — SOPS+toolkit now, Sealed Secrets after ArgoCD
- **k8s_secrets.py**: added crowdsec-bouncer + api-secrets to SECRET_DEFINITIONS (was missing)
- **Prod overlay**: secrets.yaml removed from kustomization.yaml (ADR-014: toolkit-only)
- **Stream E expanded**: SEAL-001..004 (Sealed Secrets) added after ARGO-006
- Dry-run validated: all 5 K8s secrets resolve correctly from SOPS
- **Next**: ts-bridge v1.3.0 (VPN-001) → PROD-K3S-001 (install K3s on VPS)

## Current State (2026-02-25) — end of VPN consolidation session
- Dev stack 100% operational: all services healthy
- STAGE-006d complete: CrowdSec agent + bouncer running on K8s staging
- New dev services: minio, gitea, portainer, n8n — all accessible
- **ADR-012 accepted**: Environment strategy — K3s everywhere, dual domain
- **ADR-013 accepted**: VPN consolidation — single Headscale control plane
- **Prod overlay complete**: IngressRoutes, ConfigMap patches, secrets template, values file — kustomize validated
- PROD-K3S-000 through 000d: DONE (access policy, ingress, patches, secrets, values)
- **Stream T added**: Mobile Ops (Termux + Tailscale + Crush/Go) — 3 tracks in roadmap
- **Stream V added**: VPN Consolidation — 5 phases, 13 tasks (VPN-001..013)
- **ts-bridge Stream D added**: Headscale compatibility (HC-001..004, release v1.3.0)
- **Pending**: PROD-K3S-000e (Headlamp), 000f (populate SOPS secrets), 001-006 (actual VPS migration)

## Key fixes this session
- MinIO console 503: MINIO_SERVER_URL must be http://localhost:9000 in dev (not https public URL — port 443 unreachable inside Docker)
- Grafana provisioning: Docker Compose needs explicit volume mounts; K8s uses ConfigMaps (different mechanism)
- mkcert multi-level subdomain: _get_default_domains() now auto-adds any *_DOMAIN env var with 2+ subdomain levels as explicit SAN
- Gitea admin bootstrap: GITEA_ADMIN_* env vars only work on FIRST start (empty DB). After that: docker exec --user git gitea gitea admin user create ...
- n8n healthcheck: image has no curl/wget — use node -e "require('http').get(...)"
- CrowdSec acquis.yaml: must have at least one real datasource + emptyDir volume for the path
- SOPS encrypt path_regex: can only encrypt files matching the defined path pattern (infra/config/secrets/*.enc.yaml)
- setup-local-dns: was skipping all if any domain existed; rewritten to check each domain individually

## Current State (2026-02-24)
- Branch: feature/blog-restruct
- Stream A: 81%, Stream B: ~80%
- B5 staging deploy COMPLETE: web/api/blog + Grafana/Loki/Vector + Redis + Authelia on K3s
- Prod K8s overlay created and validated (dry-run OK)
- K3s TLS SAN configured: `100.64.0.4` in cert, kubeconfig uses CA cert (no insecure-skip)
- DNS resilience: Ansible playbook deployed to all 7 homelab nodes (/etc/hosts fallback)
- Uptime Kuma: 33 monitors, 2 status pages, 6 tags configured
- `toolkit monitoring` subcommand: backup/restore/status for Uptime Kuma
- RPi3 cron backup: daily kuma-YYYYMMDD.db, 7-day retention

## Completed (2026-02-22)
- [x] STAGE-005a: Grafana K8s manifests
- [x] STAGE-005b: Loki K8s manifests
- [x] STAGE-005c: Vector DaemonSet + RBAC
- [x] STAGE-005d: Grafana UI + Loki datasource + logs verified ✓ 2026-02-22
- [x] STAGE-006a: Redis on K3s (session backend for Authelia) ✓ 2026-02-22
- [x] OBS-001: Grafana working on staging ✓ 2026-02-22
- [x] OBS-002: Loki receiving logs via Vector ✓ 2026-02-22
- [x] MON-004: Staging monitors in Uptime Kuma (via Tailscale)
- [x] MON-005: Prod + infra monitors (33 total, 2 status pages)
- [x] RPi4 NAT fix: nftables wildcard `oifname "enx*"`
- [x] Uptime Kuma DNS fix: compose.yml with `dns: [100.64.0.5, 1.1.1.1]`
- [x] toolkit monitoring CLI: backup/restore/status commands
- [x] RPi3 cron backup configured
- [x] DNS resilience playbook: /etc/hosts on all 7 nodes ✓ 2026-02-22
- [x] K3s TLS SAN: 100.64.0.4 added, insecure-skip-tls-verify removed ✓ 2026-02-22
- [x] SSH config: vps-lan→vps-pub, legacy aliases removed ✓ 2026-02-22
- [x] MON-006: Slack alerts configured (Incoming Webhook → #alerts) ✓ 2026-02-22
- [x] lessons.md: all 21 entries translated from Spanish to English ✓ 2026-02-22
- [x] Git remote: already renamed to kubelab.git (confirmed) ✓ 2026-02-22
- [x] MON-007: status.kubelab.live Traefik route created (deploy to VPS pending) ✓ 2026-02-22
- [x] OBS-003: Loki retention already configured (168h + compactor) ✓ 2026-02-22
- [x] OBS-004: Grafana dashboard provisioning + KubeLab Logs Overview JSON ✓ 2026-02-22
- [x] DOC-008: Already covered by ADR-011 (K3s migration strategy) ✓ 2026-02-22

## Completed (2026-02-24)
- [x] STAGE-006b: Authelia on K3s (full deploy + branding + secrets) ✓ 2026-02-24
- [x] Fix credentials.py key path mismatch (apps.authelia → apps.services.security.authelia) ✓ 2026-02-24
- [x] Authelia custom branding: logo.svg/png + favicon.svg/ico in authelia-assets/ ✓ 2026-02-24
- [x] Fix staging.yaml missing `disable_require_tls` for AutheliaGenerator ✓ 2026-02-24
- [x] Fix Authelia K8s: enableServiceLinks + automountServiceAccountToken ✓ 2026-02-24
- [x] Fix authelia-assets: migrated from broken inline binaryData to configMapGenerator ✓ 2026-02-24
- [x] api-secrets: created empty secret as temporary fix ✓ 2026-02-24

## Completed (2026-02-24) — continued
- [x] STAGE-006c: IngressRoutes + Authelia middleware for grafana/loki ✓ 2026-02-24
- [x] Grafana IngressRoute moved from manual kubectl to declarative in grafana.yaml ✓ 2026-02-24
- [x] catch-all IngressRoute moved from manual kubectl to nginx-errors.yaml ✓ 2026-02-24
- [x] Loki IngressRoute created with Authelia middleware ✓ 2026-02-24

## Completed (2026-02-25) — Dev stack fixes
- [x] Dev TLS cert: regenerated for `kubelab.test` + `mlorente.test` (was `cubelab.test`) ✓
- [x] /etc/hosts: all `*.kubelab.test` + `crowdsec.kubelab.test` added via `make setup-local-dns` ✓
- [x] `make regen-certs` target: regenerate cert + reinstall CA + restart Traefik in one command ✓
- [x] `_get_default_domains`: now reads `APPS_PLATFORM_WEB_DOMAIN` → includes `mlorente.test` automatically ✓
- [x] OIDC JWKS PEM: generated via `openssl genrsa` for Authelia dev ✓
- [x] Jekyll `cache_dir: /tmp/.jekyll-cache` in `_config.dev.yml` → root permission issue prevented ✓
- [x] web node_modules stale volume: fixed via `docker rm -vf web` + rebuild ✓
- [x] Volume validation `PermissionError` on `/var/lib/docker/containers`: fixed in `docker_service.py` ✓
- [x] CrowdSec `enable_bouncer_middleware` config option: false in dev (bouncer incompatible + blocks localhost) ✓

## Completed (2026-02-25) — STAGE-006d ✓ 2026-02-25
- [x] CrowdSec K8s manifests created: `infra/k8s/base/services/crowdsec.yaml` ✓
  - CrowdSec agent (PVCs + ConfigMap acquis + Deployment + Service)
  - Bouncer (Deployment + Service + Middleware CRD forwardAuth)
- [x] `crowdsec-bouncer` secret in staging/secrets.yaml (key from SOPS staging.enc.yaml) ✓
- [x] IngressRoutes (api/web/blog) updated with `crowdsec-bouncer` middleware ✓
- [x] kustomization.yaml updated ✓
- [x] acquis.yaml: valid file datasource + emptyDir volume mount → agent starts cleanly ✓
- [x] Applied to cluster: all pods 1/1 Running, zero restarts ✓

## Completed (2026-02-25) — Dev services expansion
- [x] n8n: added to common.yaml, compose files created, DNS entry, healthcheck via node ✓
- [x] minio, gitea, portainer: started in dev, Traefik routing configured ✓
- [x] Auth design: portainer/gitea/minio/n8n → own auth (no Authelia double-auth) ✓
- [x] Gitea compose.base.yml: full GITEA__ env var config (disable registration, admin bootstrap) ✓
- [x] SOPS dev secrets: gitea admin_user/admin_password + n8n encryption_key added ✓
- [x] n8n encryption_key: in dev.yaml + SOPS. Needs toolkit credentials generator for staging.

## Dev stack (2026-02-25) — ALL healthy
| Service | URL | Auth |
|---------|-----|------|
| api, web, blog | *.kubelab.test / mlorente.test | — |
| grafana | grafana.kubelab.test | Authelia |
| authelia | auth.kubelab.test | — |
| portainer | portainer.kubelab.test | Portainer own |
| gitea | gitea.kubelab.test | Gitea own (admin/645610515) |
| minio | minio.kubelab.test + console.minio.kubelab.test | MinIO root (admin/645610515) |
| n8n | n8n.kubelab.test | n8n own (create on first login) |
| crowdsec, loki, redis | internal | — |
| github-runner | — | needs fresh token (not critical) |

## Plan for 2026-02-26
1. **ts-bridge v1.3.0** (HC-001..004): Add `TS_CONTROL_URL` env var, test against Headscale, release — unblocks Stream V
2. **PROD-K3S-000f**: Populate prod secrets from SOPS (crowdsec, grafana, authelia, api) — unblocks VPS migration
3. **PROD-K3S-001**: Install K3s single-node on VPS (if time permits)

## Next priorities (after 2026-02-26)
- PROD-K3S-002..006: Apply prod overlay, verify, update CI, decommission Compose
- VPN consolidation Phase 1-3: Headscale users/ACL → Headplane → migrate Windows PCs
- MAIL-001..002: Zoho domain alias + Authelia SMTP migration (quick win, low priority)
- Add n8n DNS: `echo "127.0.0.1 n8n.kubelab.test" | sudo tee -a /etc/hosts` (pending user action)
- Headlamp (K8s UI) for staging/prod — add to K8s manifests
- Future: Templatize infra services (authelia, grafana, loki, vector, redis, traefik) via K8sGenerator

## Infrastructure Facts
- **DNS provider**: Cloudflare (NOT Hetzner). All `*.kubelab.live` public DNS records managed in Cloudflare.
- **Mini PCs BIOS**: Acemagic and Beelink use AMI Aptio BIOS. Auto-power-on: set "State after G3" → "S0 State".
- **Documentation language**: Always English. Chat in Spanish, docs in English.

## Kustomize Overlay Pattern (2026-02-25)
- Base manifests (`infra/k8s/base/`) use staging domains (hardcoded in ConfigMaps + IngressRoutes)
- Staging overlay: uses base as-is (staging domains match)
- Prod overlay: `patches.yaml` overrides base resources (grafana-config, authelia-config, IngressRoutes)
- Overlay-only resources (api/web/blog ConfigMaps, IngressRoutes, Secrets) go in `resources:` list
- Base-override resources go in `patches: [{path: patches.yaml}]` — NOT in resources (causes duplicate error)
- Validated: `kubectl kustomize infra/k8s/overlays/prod/` builds clean, all domains correct

## Key Gotchas Learned (2026-02-25)
- **mkcert wildcard scope**: `*.kubelab.test` does NOT cover `mlorente.test` — sibling root domains need explicit SAN entry. Fix: `make regen-certs`
- **CrowdSec `fbonalair` bouncer**: incompatible with CrowdSec v1.7.6 — LAPI returns 403 on all bouncer queries. Root cause unclear (stale volume OR image incompatibility). Disable in dev via `enable_bouncer_middleware: false`. Plan B for K8s: `ghcr.io/maxlerebourg/crowdsec-bouncer-traefik-plugin`
- **CrowdSec bouncer pre-auth pattern**: pre-generate API key, inject via same K8s Secret into both agent (`BOUNCER_KEY_` env var) and bouncer (`CROWDSEC_BOUNCER_API_KEY`). Zero manual steps post-deploy.
- **Jekyll `.jekyll-cache` permissions**: if any container run creates cache as root, subsequent non-root runs fail. Fix: `cache_dir: /tmp/.jekyll-cache` in `_config.dev.yml`
- **Docker anonymous volume permissions**: stale volumes persist across container restarts. Fix: `docker rm -vf <container>` to force volume recreation
- **`APPS_PLATFORM_WEB_DOMAIN`**: the env var prefix for the web app domain is `APPS_PLATFORM_WEB_`, not `APPS_WEB_`
- **CrowdSec acquis.yaml in K8s**: must have at least one enabled datasource or agent exits fatally. Use a file source + emptyDir volume at `/var/log/traefik/` so the path exists; watcher idles cleanly with no real log file

## Key Gotchas Learned (2026-02-24)
- **Docker bridge DNS**: containers on bridge networks can't resolve split-DNS domains. Fix: explicit `dns:` in compose.yml
- **nftables + USB Ethernet**: never hardcode `enx*` interface names — use wildcards
- **K3s TLS SAN**: must be configured BEFORE first start (or restart to regen certs). Add all access IPs (LAN + Tailscale)
- **Uptime Kuma v2.x**: no UI backup/import. SQLite DB (`kuma.db`) is the only backup method
- **kuma.db in Git**: blocked by pre-commit (>1MB). Gitignored. Use `toolkit monitoring backup` locally
- **Ansible on Jetson**: Python 3.6 too old for modules. Use `legacy_python: true` + `raw` module
- **DNS cascade failure**: All LAN nodes depend on Pi-hole (RPi4). Fix: /etc/hosts via Ansible for critical domains
- **Authelia K8s Service Links**: K8s injects `AUTHELIA_*` env vars from Service discovery → config conflict. Fix: `enableServiceLinks: false`
- **Authelia K8s RO rootfs**: Image has read-only `/run` → SA token mount fails. Fix: `automountServiceAccountToken: false`
- **Binary assets in K8s**: Never inline base64 in YAML. Use kustomize `configMapGenerator` with `files:`
- **Authelia secrets key path**: `apps.services.security.authelia.*` (3 levels deep, easy to get wrong)

## RPi 4 Network Config
- USB Ethernet (`enx*` wildcard): WAN primary, metric 100
- WiFi (`wlan0`): WAN backup, metric 600
- eth0: LAN to switch, 172.16.1.1 (static)
- Pi-hole: compose at /opt/pihole/, volumes external:true
- nftables: `oifname "enx*" masquerade` (generic wildcard)

## LAN IPs (172.16.1.0/24)
rpi4=.1, ace1=.2, bee=.3, jet1=.4, ace2=.5, k3s-server=.10, k3s-agent-1=.11, k3s-agent-2=.12

## Headscale VPN IPs (100.64.0.0/24)
msi=.1, vps=.2, bee=.3, k3s-server=.4, rpi4=.5, rpi3=.6, k3s-agent-1=.7, jet1=.8, k3s-agent-2=.9

## Venv Gotcha
- After repo rename (cubelab→kubelab), venv shebangs break. Fix: `rm -rf .venv && poetry install`
