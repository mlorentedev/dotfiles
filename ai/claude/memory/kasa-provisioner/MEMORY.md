# kasa-provisioner — Session Memory

## Project

Offline provisioning + local control of TP-Link Kasa/Tapo smart plugs. Zero cloud dependency.

## Target Device

- **Model:** EP10(US) HW 1.0 / FW 1.0.5
- **Protocol:** Legacy XOR (IOT.XOR) — TCP port 9999
- **Auth:** None required (legacy, no KLAP)
- **IP on LAN:** 10.0.0.167
- **MAC:** 3C:52:A1:D1:B0:27
- **AP SSID:** TP-LINK_Smart Plug_B027
- **State field:** `relay_state` (int 1/0), NOT `device_on`

## Repo Structure

```
src/kasa_provisioner/
  domain/models.py         ← WifiConfig, DeviceInfo, DeviceState, ProtocolType
  domain/exceptions.py     ← Domain exception hierarchy
  application/bootstrap.py ← AP provisioning use case (LegacyClient)
  application/discovery.py ← LAN discovery via python-kasa Discover
  application/control.py   ← Power control via Device.connect()
  infra/legacy_client.py   ← Raw XOR/TCP client (bootstrap AP only)
  cli/main.py              ← Typer CLI (bootstrap, discover, control, status)
tests/unit/
  test_legacy_client.py
  test_bootstrap.py
.github/workflows/ci.yml   ← ruff + mypy + pytest
README.md, .gitignore
```

## Key Decisions

- **LegacyClient**: TCP, reserved for bootstrap AP phase only (192.168.0.1)
- **python-kasa Device.connect()**: used for ALL post-LAN control (legacy + KLAP)
- **StrEnum in Typer**: use `str` args + manual enum conversion (StrEnum breaks make_metavar)
- **hw_version in python-kasa 0.9**: use `device.sys_info.get('hw_ver')`, not `device.hw_version`
- **Click version**: pinned `<8.2.0` — Click 8.2+ changed `make_metavar()` signature, breaks Typer 0.12
- **WiFi automation (Phase 3)**: pywifi (cross-platform: Windows + Linux), NOT nmcli

## Bootstrap Bug — CRITICAL FINDING

The Kasa app provisions via **UDP** (no 4-byte header). Our LegacyClient uses TCP with 4-byte header.
The EP10 silently ignores TCP set_stainfo in AP mode — AP never disappears.

**T-022 DONE (commit 165ee18):**
- `set_wifi_udp()` added to LegacyClient — UDP datagram, raw XOR, NO 4-byte length prefix
- Dual namespace: `netif` + `smartlife.iot.common.softaponboarding`
- BootstrapUseCase now uses UDP
- **Pending: real-device test** (factory reset EP10 → connect AP → run bootstrap)

**Workaround still available:** Mobile app for initial provisioning, then kasa-provisioner for LAN control.

## Working Commands

```bash
source .venv/bin/activate   # or: poetry run kasa-provisioner ...
kasa-provisioner discover
kasa-provisioner status 10.0.0.167
kasa-provisioner control 10.0.0.167 on
kasa-provisioner control 10.0.0.167 off
kasa-provisioner control 10.0.0.167 toggle
```

## Tests

10/10 passing. `poetry run pytest`

## Vault

`~/Projects/knowledge/10_projects/kasa-provisioner/`
- `00-context.md` — project overview
- `10-roadmap.md` — 4 phases (Phase 1 + 2 mostly done)
- `11-tasks.md` — T-001..T-023, ~80% done
- `30-architecture/` — ADR-001 (hybrid protocol), ADR-002 (pywifi), ADR-003 (bootstrap)
- `40-runbooks/guide-provisioning.md`
- `50-troubleshooting/ep10-bootstrap-fw105.md` — UDP vs TCP root cause
- `90-lessons.md` — L-001..L-008

## Phase Status

| Phase | Status |
|-------|--------|
| 1 — Core infra | ✅ Done (bootstrap UDP fix pending) |
| 2 — KLAP | ✅ Done (via python-kasa) |
| 3 — WiFi automation | 🔲 Not started |
| 4 — Hardening | 🔲 Not started |
