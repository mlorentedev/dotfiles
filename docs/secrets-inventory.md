# Secrets inventory & migration map

> Working artifact for the secrets-management redesign (branch `arch/secrets-management`).
> Three sources: **(a) dotfiles age files**, **(b) env-var mapping**, **(c) Bitwarden**.
> (a)+(b) are filled below from `sensitive/env-mapping.conf` + `sensitive/*.secret.age`.
> Fill column **In Bitwarden?** from: `bw list items --pretty | jq -r '.[] | (.folder // "-") + " / " + .name'` (names only — no values).
>
> Goal: one SSOT backend per secret (no duplication). `dotf secrets` reads the registry derived from this table.

## Legend

- **Plane**: `app` (service API keys/tokens your apps consume) · `personal` (recovery/backup codes, app-passwords, logins) · `floor` (boot/DR — needed before you can reach Bitwarden).
- **Target backend**: `bw` (Bitwarden — live SSOT) · `age-floor` (stays local: boot/DR root) · the age **DR export** escrows *everything* regardless.
- **Consumers**: best-guess from the env var name — **confirm before rotating** (rotation blast radius, #321).

## (a)+(b) dotfiles secrets

| # | Logical secret | age file | Env var / file target | Plane | Consumer(s) (confirm) | Target bw item | In Bitwarden? | Flags |
|---|---|---|---|---|---|---|---|---|
| 1 | GitHub token (shared) | `github.token` | `GITHUB_PERSONAL_ACCESS_TOKEN` + `RELEASE_TOKEN` | app | CLI/API, goreleaser+CI release | **split** → `apps/github-cli-pat`, `apps/github-release-pat` | ? | **#321 one-token-many-uses → split per purpose** |
| 2 | Bitácora PAT | `github.bitacora` | `BITACORA_PAT` | app | bitácora board/Project writes, 20 repos | `apps/github-bitacora-pat` | ? | per-purpose ✔ already split |
| 3 | DockerHub token | `dockerhub.token` | `DOCKERHUB_TOKEN` | app | CI image push | `apps/dockerhub-token` | ? | |
| 4 | DockerHub username | `dockerhub.username` | `DOCKERHUB_USERNAME` | app | CI image push | field on `apps/dockerhub-token` | ? | collapse into the token item |
| 5 | Cloudflare API token | `cloudflare.api-token` | `CLOUDFLARE_API_TOKEN` | app | DNS/infra automation | `infra/cloudflare-api-token` | ? | |
| 6 | Hetzner API key | `hetzner.api-key` | `HETZNER_API_TOKEN` | app | infra provisioning | `infra/hetzner-api-token` | ? | |
| 7 | Hetzner SSH key | `hetzner.ssh` | `HETZNER_SSH_KEY` | floor/app | server access | `infra/hetzner-ssh` (SSH Key type) | ? | native SSH Key item |
| 8 | OpenAI API key | `chatgpt.api-key` | `OPENAI_API_KEY` | app | LLM calls | `apps/openai-api-key` | ? | **naming drift: store says "chatgpt", service is OpenAI** |
| 9 | OpenRouter API key | `openrouter.api.key` | `OPENROUTER_API_KEY` | app | LLM router | `apps/openrouter-api-key` | ? | **naming drift: double-dot `.api.key`** |
| 10 | NaN API key | `nan.api-key` | `NAN_API_KEY` | app | NaN cloud engine | `apps/nan-api-key` | ? | |
| 11 | Stripe API key | `stripe.api-key` | `STRIPE_API_KEY` | app | payments | `apps/stripe-api-key` | ? | |
| 12 | YouTube API key | `youtube.api-key` | `YOUTUBE_API_KEY` | app | yt-metrics | `apps/youtube-api-key` | ? | |
| 13 | Beehiiv API key | `beehiiv.api-key` | `BEEHIIV_API_KEY` | app | newsletter | `apps/beehiiv-api-key` | ? | |
| 14 | Beehiiv DNS records | `beehiiv.dns-records` | `BEEHIIV_DNS_RECORDS` | app | DNS config | `apps/beehiiv-dns` (Secure Note) | ? | not a key — reference data |
| 15 | PyPI token | `pypi.token` | `PYPI_TOKEN` | app | package publish | `apps/pypi-token` | ? | |
| 16 | Tailscale auth key | `tailscale.auth-key` | `TS_AUTHKEY` | app/infra | VPN join | `infra/tailscale-auth-key` | ? | short-lived by nature |
| 17 | Pollex API key | `pollex.api-key` | `POLLEX_API_KEY` | app | pollex NaN engine (#237) | `apps/pollex-api-key` | ? | |
| 18 | X (Twitter) credential | `x.api-key`, `x.api-key-secret`, `x.access-token`, `x.access-token-secret`, `x.bearer-token`, `x.client-id`, `x.client-secret` | `X_*` (7 vars) | app | X API | **collapse → 1 item** `apps/x-twitter` (7 custom fields) | ? | **7 files → 1 item** |
| 19 | Zoho app passwords | `zoho.app-passwords` | `ZOHO_APP_PASSWORDS` | personal | mail clients | `personal/zoho-app-passwords` | ? | |
| 20 | Kubelab kubeconfig | `kubelab.kubeconfig` | file → `~/.kube/kubelab.config` | app/infra | kubectl | `infra/kubelab-kubeconfig` (Secure Note/attachment) | ? | file secret |
| 21 | SSH key (id_ed25519) | `id_ed25519` | file → `~/.ssh/id_ed25519` | **floor** | git clone at bootstrap | **age-floor** (needed before bw) | n/a | keep in floor — boot dependency |
| 22 | Gmail backup codes | `gmail.backup-code` | file → `~/.secrets/...` | personal | account recovery | `personal/gmail` (field) | ? | |
| 23 | ChatGPT backup code | `chatgpt.backup-code` | file | personal | account recovery | `personal/openai` (field) | ? | |
| 24 | ChatGPT recovery code | `chatgpt.recovery-code` | file | personal | account recovery | `personal/openai` (field) | ? | collapse with #23 |
| 25 | Stripe backup code | `stripe.backup-code` | file | personal | account recovery | `personal/stripe` (field) | ? | |
| 26 | Zoho recovery code | `zoho.recovery-code` | file | personal | account recovery | `personal/zoho` (field) | ? | collapse with #19 |
| — | Ollama API key | (commented out) | `OLLAMA_API_KEY` | app | homelab LLM (VPN) | `infra/ollama-api-key` | ? | not yet encrypted |

**Floor (stays local, never only-in-bw):**
- The **age private key** (root of DR — offline backups only).
- `bw-master-password.age` (Bitwarden unlock — can't live inside bw).
- SSH boot key (#21) if needed before bw is reachable.

## (c) Bitwarden — dev/infra cross-section (the ~20 we manage)

> Captured 2026-06-25 via `bw list items` (names/structure only, no values). The vault has **~145 items, all in "No Folder"** (zero organization). **~125 are personal logins** (banks, travel, gov, shopping) — **OUT OF SCOPE, untouched.** Only the dev/infra cross-section below is `dotf secrets` territory.

| Bitwarden item | Type | Fields (structure) | Plane | Overlaps dotfiles age? | Decision |
|---|---|---|---|---|---|
| **AGE-SECRET-KEY-CI** | note | — | **floor/ROOT** | (the key itself) | 🔴 **move authoritative copy OFFLINE** — must NOT live only in bw (circular dep) |
| **AGE-SECRET-KEY-PERSONAL** | note | — | **floor/ROOT** | (the key itself) | 🔴 same — offline DR root, bw copy = convenience only |
| **GitHub** | login | PAT, Runner token, Recovery codes, evalkit-sdk-ci…private-key.pem, hermes-nan-vaule, bitacora token, release-token, kubelab-dispatch-token | app | `github.token`, `github.bitacora` | **split per-purpose (#321)**; bw=SSOT; retire age copies |
| **DockerHub** | login | PAT | app | `dockerhub.token`, `.username` | bw=SSOT; retire age |
| **Hetzner** | SSH Key (5) | — | infra | `hetzner.ssh` | bw=SSOT; retire age |
| **Hetzner** | login | key, login | app | `hetzner.api-key` | bw=SSOT; retire age |
| **cloud.nan.builders** | login | api-key, telegram_kubelab_bot, chat_id | app | `nan.api-key` | bw=SSOT; retire age |
| **login.tailscale.com** | login | auth-key | app/infra | `tailscale.auth-key` | bw=SSOT; retire age |
| **OPEN ROUTER API KEY** + **openrouter.ai** | note + login | — | app | `openrouter.api.key` | **merge the 2 bw entries**; bw=SSOT; retire age |
| **POLLEX_API_KEY** | note | — | app | `pollex.api-key` | bw=SSOT; retire age |
| **Stripe** | login | backup-codes | app/personal | `stripe.api-key`, `stripe.backup-code` | bw=SSOT; retire age |
| **pypi.org** | login | Recovery codes, API token | app | `pypi.token` | bw=SSOT; retire age |
| **mail.zoho.com** | login | iPhone/Pixel App-Specific Password, client-id, client-secret | personal/app | `zoho.app-passwords`, `zoho.recovery-code` | bw=SSOT; retire age |
| **AWS** + **signin.aws.amazon.com** | login | kubelab-terraform | infra | — | net-new infra (not in age) |
| **Brightdata** | login | api-key | app | — | net-new |
| **PEXELS-API-KEY** | note | — | app | — | net-new |
| **api.wikimedia.org** | login | access-token, client-id, client-secret | app | — | net-new |
| **dashboard.ngrok.com** | login | Recovery codes, Auth token | app/infra | — | net-new |
| **Kubelab** | note | Slack Webhook, Gmail-Authelia staging | infra | `kubelab.kubeconfig`(?) | net-new + maybe add kubeconfig |
| **TS-BRIDGE**, **TS-BRIDGE-HEADSCALE** | note | — | infra | — | net-new |
| **SSH Key - Dell Work** | SSH Key (5) | — | infra | — | net-new |
| **grafana/status.kubelab.live** | login | — | infra | — | dashboards |

### age secrets NOT yet in bw → migrate TO bw
`cloudflare.api-token`, `chatgpt.api-key`(→OpenAI), `youtube.api-key`, `beehiiv.api-key`+`.dns-records`, `chatgpt.backup/recovery-code`, `gmail.backup-code`, `kubelab.kubeconfig`. (Add as fields on the matching login, or as a `apps/`/`infra/` item.)

### Stays in age (floor — never only-in-bw)
The **age private key** (offline-rooted), `bw-master-password.age`, `id_ed25519` boot key.

## Critical actions surfaced by the inventory

1. 🔴 **age keys live in bw = circular DR dependency.** Authoritative copy must go OFFLINE; bw copy is convenience only.
2. **Bitwarden has zero folders.** Introduce `apps/` `infra/` `personal/` and move the ~20 dev/infra items in (leave ~125 personal as-is for now).
3. **Inconsistent structure** (fields-in-login vs standalone notes). The registry must address each uniformly as `item › field` or dedicated item.

## De-duplication rule

A secret in **both** age and Bitwarden → pick **one** live SSOT (default `bw`, for cross-device), point the registry there, retire the age copy. The age **DR export** still escrows everything (derived, not authoritative). `floor` items are the only ones authoritative in age.
