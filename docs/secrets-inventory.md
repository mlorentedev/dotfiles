# Secrets inventory & migration map

> Working artifact for the secrets-management redesign (branch `arch/secrets-management`).
> Three sources: **(a) dotfiles age files**, **(b) env-var mapping**, **(c) Bitwarden**.
> (a)+(b) are filled below from `secrets/registry.yaml` + `sensitive/*.secret.age`.
> Fill column **In Bitwarden?** from: `bw list items --pretty | jq -r '.[] | (.folder // "-") + " / " + .name'` (names only — no values).
>
> Goal: one SSOT backend per secret (no duplication). The registry was derived from this table and is now the SSOT; this file is the migration's record, not a mirror of it.

## Legend

- **Plane**: `app` (service API keys/tokens your apps consume) · `personal` (recovery/backup codes, app-passwords, logins) · `floor` (boot/DR — needed before you can reach Bitwarden).
- **Target backend**: `bw` (Bitwarden — live SSOT) · `age-floor` (stays local: boot/DR root) · the age **DR export** escrows *everything* regardless.
- **Consumers**: best-guess from the env var name — **confirm before rotating** (rotation blast radius, #321).
- **Registry id**: the entry in `secrets/registry.yaml` that now owns the secret. Where it lives in Bitwarden (`bw.folder`, `bw.item`, `bw.field`) is declared there, once, and `dotf secrets drift` checks it against the store. This table named the Bitwarden items too, and they drifted from the registry unnoticed (CLI-078 review round 4), so it no longer repeats them.

## (a)+(b) dotfiles secrets

> **"In Bitwarden?" column confirmed 2026-06-28** via a names/structure-only `bw list items`
> dump (no values). Vault = 152 items, still **all in "No Folder"**. Legend: `Yes — <item> (<field>)`
> = value present · `No` = age-only, `migrate` will create the canonical item fresh · `n/a` = floor.

| # | Logical secret | age file | Env var / file target | Plane | Consumer(s) (confirm) | Registry id | In Bitwarden? (2026-06-28) | Flags |
|---|---|---|---|---|---|---|---|---|
| 1 | GitHub token (shared) | `github.token` | `GITHUB_PERSONAL_ACCESS_TOKEN` + `RELEASE_TOKEN` | app | CLI/API, goreleaser+CI release | `GITHUB_PERSONAL_ACCESS_TOKEN`, `RELEASE_TOKEN` | Yes — `GitHub` 8-in-1 (PAT, release-token) | **#321 one-token-many-uses → split per purpose**, by `dotf secrets reconcile` copying each from its `GitHub` field (`bw.from`, CLI-080); both now bw-backed |
| 2 | Bitácora PAT | `github.bitacora` | `BITACORA_PAT` | app | bitácora board/Project writes, 20 repos | `BITACORA_PAT` | Yes — `GitHub` 8-in-1 (bitacora token) | per-purpose ✔ already split |
| 3 | DockerHub token | `dockerhub.token` | `DOCKERHUB_TOKEN` | app | CI image push | `DOCKERHUB_TOKEN` | Yes — `DockerHub` (PAT field) | |
| 4 | DockerHub username | `dockerhub.username` | `DOCKERHUB_USERNAME` | app | CI image push | `DOCKERHUB_USERNAME` | Yes — `DockerHub` (login username) | one item with `DOCKERHUB_TOKEN` |
| 5 | Cloudflare API token | `cloudflare.api-token` | `CLOUDFLARE_API_TOKEN` | app | DNS/infra automation | `CLOUDFLARE_API_TOKEN` | **No** (age-only) | net-new in bw |
| 6 | Hetzner API key | `hetzner.api-key` | `HETZNER_API_TOKEN` | app | infra provisioning | `HETZNER_API_TOKEN` | Yes — `Hetzner` login (key field) | |
| 7 | Hetzner SSH key | `hetzner.ssh` | `HETZNER_SSH_KEY` | floor/app | server access | `HETZNER_SSH_KEY` | Yes — `Hetzner` (SSH Key, type 5) | native SSH Key item |
| 8 | OpenAI API key | `chatgpt.api-key` | `OPENAI_API_KEY` | app | LLM calls | `OPENAI_API_KEY` | **No** — only `auth.openai.com` account login | **naming drift: store says "chatgpt", service is OpenAI; good canary (net-new)** |
| 9 | OpenRouter API key | `openrouter.api.key` | `OPENROUTER_API_KEY` | app | LLM router | `OPENROUTER_API_KEY` | Yes — **2 entries**: `OPEN ROUTER API KEY` (note) + `openrouter.ai` (login) → **MERGE** | **naming drift: double-dot `.api.key`** |
| 10 | NaN API key | `nan.api-key` | `NAN_API_KEY` | app | NaN cloud engine | `NAN_API_KEY` | Yes — `cloud.nan.builders` (api-key field) | |
| 11 | Stripe API key | `stripe.api-key` | `STRIPE_API_KEY` | app | payments | `STRIPE_API_KEY` | Partial — `Stripe` login (no explicit api-key field) | confirm where the api key lives |
| 12 | YouTube API key | `youtube.api-key` | `YOUTUBE_API_KEY` | app | yt-metrics | `YOUTUBE_API_KEY` | **No** (age-only) | net-new in bw |
| 13 | Beehiiv API key | `beehiiv.api-key` | `BEEHIIV_API_KEY` | app | newsletter | `BEEHIIV_API_KEY` | **No** (age-only) | net-new in bw |
| 14 | Beehiiv DNS records | `beehiiv.dns-records` | `BEEHIIV_DNS_RECORDS` | app | DNS config | `BEEHIIV_DNS_RECORDS` | **No** (age-only) | not a key — reference data (Secure Note) |
| 15 | PyPI token | `pypi.token` | `PYPI_TOKEN` | app | package publish | `PYPI_TOKEN` | Yes — `pypi.org` (API token field) | |
| 16 | Tailscale auth key | `tailscale.auth-key` | `TS_AUTHKEY` | app/infra | VPN join | `TS_AUTHKEY` | Yes — `login.tailscale.com` (auth-key field) | short-lived by nature |
| 17 | Pollex API key | `pollex.api-key` | `POLLEX_API_KEY` | app | pollex NaN engine (#237) | `POLLEX_API_KEY` | Yes — `POLLEX_API_KEY` (note) | |
| 18 | X (Twitter) credential | `x.api-key`, `x.api-key-secret`, `x.access-token`, `x.access-token-secret`, `x.bearer-token`, `x.client-id`, `x.client-secret` | `X_*` (7 vars) | app | X API | `X_*` (7 ids) | **No** — only `x.com` personal login (no API item) | **7 files → 1 item with 7 custom fields**; net-new in bw |
| 19 | Zoho app passwords | `zoho.app-passwords` | `ZOHO_APP_PASSWORDS` | personal | mail clients | `ZOHO_APP_PASSWORDS` | Yes — `mail.zoho.com` (app-specific passwords) | |
| 20 | Kubelab kubeconfig | `kubelab.kubeconfig` | file → `~/.kube/kubelab.config` | app/infra | kubectl | `KUBECONFIG` | **No** — `Kubelab` note holds Slack/Gmail-Authelia only | file secret; `migrate` refuses files |
| 21 | SSH key (id_ed25519) | `id_ed25519` | file → `~/.ssh/id_ed25519` | **floor** | git clone at bootstrap | `SSH_KEY` | n/a | **age-floor**: keep in floor — boot dependency (needed before bw) |
| 22 | Gmail backup codes | `gmail.backup-code` | file → `~/.secrets/...` | personal | account recovery | `GMAIL_BACKUP_CODE` | **No** — `Gmail` login exists, no codes field | file secret |
| 23 | ChatGPT backup code | `chatgpt.backup-code` | file | personal | account recovery | `CHATGPT_BACKUP_CODE` | **No** (age-only) | file secret |
| 24 | ChatGPT recovery code | `chatgpt.recovery-code` | file | personal | account recovery | `CHATGPT_RECOVERY_CODE` | **No** (age-only) | collapse with #23 |
| 25 | Stripe backup code | `stripe.backup-code` | file | personal | account recovery | `STRIPE_BACKUP_CODE` | Yes — `Stripe` (backup-codes field) | file secret |
| 26 | Zoho recovery code | `zoho.recovery-code` | file | personal | account recovery | `ZOHO_RECOVERY_CODE` | **No** — no recovery-code field on `mail.zoho.com` | collapse with #19 |
| — | Ollama API key | (commented out) | `OLLAMA_API_KEY` | app | homelab LLM (VPN) | — (not in the registry) | **No** (not yet encrypted) | not yet encrypted |

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
Done for every one listed here in June (`cloudflare.api-token`, `chatgpt.api-key`, `youtube.api-key`, `beehiiv.api-key` and `.dns-records`, the ChatGPT and Gmail codes, `kubelab.kubeconfig`): on 2026-09-23 each is `backend: bw` in the registry. Still age-backed on that date: `ZOHO_APP_PASSWORDS` and `ZOHO_RECOVERY_CODE` (personal, with dormant `bw:` blocks), and the floor. `dotf secrets ls` prints the live backend of each.

### Stays in age (floor — never only-in-bw)
The **age private key** (offline-rooted), `bw-master-password.age`, `id_ed25519` boot key.

## Critical actions surfaced by the inventory

1. 🔴 **age keys live in bw = circular DR dependency.** Authoritative copy must go OFFLINE; bw copy is convenience only.
2. **Bitwarden has zero folders.** Introduce `Dotfiles/apps` and `Dotfiles/infra` and move the ~20 dev/infra items in (leave ~125 personal as-is for now).
3. **Inconsistent structure** (fields-in-login vs standalone notes). The registry must address each uniformly as `item › field` or dedicated item.

## De-duplication rule

A secret in **both** age and Bitwarden → pick **one** live SSOT (default `bw`, for cross-device), point the registry there, retire the age copy. The age **DR export** still escrows everything (derived, not authoritative). `floor` items are the only ones authoritative in age.
