# Security policy

## Supported versions

Only the latest release of `dotf` gets security fixes. Fixes ship as a new release, never as a patch to an older one. The install scripts in `scripts/` are supported as they are on `main`.

| Version | Supported |
|---|---|
| Latest release | Yes |
| Anything older | No: upgrade with `./install.sh` or `install.ps1` |

## Reporting a vulnerability

Do not open a public issue or pull request for a vulnerability.

1. **Preferred:** report it privately through GitHub, at [Security → Report a vulnerability](https://github.com/mlorentedev/dotfiles/security/advisories/new). The report stays private until a fix is published.
2. **If you cannot use GitHub:** email mlorentedev@gmail.com with `SECURITY` in the subject. Email is not encrypted, so send a short summary only. The maintainer replies to agree on a private way to share the full details.

A report through GitHub should include:

- the affected version (`dotf version`) and operating system;
- the steps to reproduce;
- what an attacker gains.

An email should carry only the version, the operating system and a one-line impact. The steps to reproduce go over the private channel agreed in the reply.

Never include a real credential, token or secret value in a report. A redacted example works just as well.

## What to expect

This is a personal project maintained by one person, so the times below are targets, not guarantees.

- An acknowledgement within 7 days.
- A first assessment, and a plan or a reason for declining, within 30 days.
- Credit in the advisory when the fix is published, unless you ask not to be named.

## Scope

In scope:

- the `dotf` CLI (`cli/`);
- the install and bootstrap scripts (`install.sh`, `install.ps1`, `setup-linux.sh`, `setup-windows.ps1`);
- secret handling: `dotf secrets`, `secrets/registry.yaml` and the backup and escrow paths;
- the GitHub Actions workflows in `.github/workflows/`.

Out of scope:

- personal configuration files, such as shell rc files and editor settings, unless they expose a secret or run untrusted code;
- vulnerabilities in third-party tools this repository installs. Report those upstream.
