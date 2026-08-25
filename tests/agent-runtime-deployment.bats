#!/usr/bin/env bats
# CLI-042 PR E — the deployment of the agent executor seam, as IaC.
#
# AC7: the NaN credential never lands in a file — the hive daemon takes it from
#      `dotf secrets run`, not from an EnvironmentFile or an environment.d
#      fragment.
# AC8: the drop of Ollama and OpenRouter from hive's WORKER is complete, not
#      partial — asserted as a class over hive-vault's declared environment
#      rather than as a grep for one endpoint string.
# AC9: `dotf doctor` catches "probes present but can serve nothing" at
#      diagnostic time (the Go half lives in cli/internal/doctor).

load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    DROPIN="$DOTFILES_DIR/systemd/hive.service.d/10-dotf-secrets.conf"
    export DROPIN
}

# --- AC7: the drop-in is the credential seam ---

@test "AC7: the hive.service drop-in exists in the repo's systemd SSOT" {
    [ -f "$DROPIN" ]
}

@test "AC7: the drop-in invokes hive serve through dotf secrets run" {
    grep -qE '^ExecStart=%h/\.local/bin/dotf secrets run .* -- %h/\.local/bin/hive serve$' "$DROPIN"
}

# A second ExecStart on a Type=simple unit is a load-time ERROR unless the list
# is reset first. Without the empty assignment the drop-in does not override the
# base unit -- systemd refuses to load it, and the daemon keeps running with no
# credential, which is the exact state AC7 exists to end.
@test "AC7: the drop-in resets ExecStart before setting its own" {
    run grep -c '^ExecStart=' "$DROPIN"
    [ "$output" -eq 2 ]
    # The reset must come first, or it clears the value we just set.
    run grep -n '^ExecStart=$' "$DROPIN"
    reset_line="${output%%:*}"
    run grep -n '^ExecStart=%h' "$DROPIN"
    set_line="${output%%:*}"
    [ "$reset_line" -lt "$set_line" ]
}

# A bare `run` injects the entire registry. This child is a long-lived daemon:
# it would hold every mapped secret for the life of the session, for the sake of
# one.
@test "AC7: the injection is scoped with --only, not the whole registry" {
    grep -qF -- '--only NAN_API_KEY' "$DROPIN"
}

# The worker contract has TWO halves. hive reads HIVE_WORKER_BASE_URL and
# HIVE_WORKER_API_KEY -- NOT NAN_API_KEY -- and a daemon holding one half serves
# exactly as little as one holding neither. Measured 2026-08-24: worker_status
# reported `Configured: no — set HIVE_WORKER_BASE_URL` while systemd called the
# unit active (running).
@test "AC7: the unit declares the worker base URL, the non-secret half" {
    grep -qE '^Environment=HIVE_WORKER_BASE_URL=https://' "$DROPIN"
}

# --only takes the registry ID here, and an id selects every var the secret
# exposes -- which is how HIVE_WORKER_API_KEY reaches hive from a flag naming
# only NAN_API_KEY. The registry must therefore actually expose both names, or
# the drop-in injects a variable hive ignores and the worker stays unconfigured.
@test "AC7: the registry exposes HIVE_WORKER_API_KEY from the same secret" {
    run python3 - "$DOTFILES_DIR/secrets/registry.yaml" <<'PY'
import sys, yaml
doc = yaml.safe_load(open(sys.argv[1]))
for s in doc.get("secrets", []):
    if s.get("id") == "NAN_API_KEY":
        env = s.get("expose", {}).get("env")
        names = env if isinstance(env, list) else [env]
        print("OK" if "HIVE_WORKER_API_KEY" in names else "MISSING:" + str(names))
        break
else:
    print("NO_SUCH_SECRET")
PY
    [ "$status" -eq 0 ]
    [ "$output" = "OK" ]
}

# ADR-025: no literal home path. %h is what the two units already in systemd/
# use, and a --user unit's minimal PATH is why the binary is spelled absolutely
# rather than as a bare `dotf`.
@test "AC7: the drop-in uses %h, never a literal home, and never a bare binary" {
    refute_grep '^ExecStart=.*/home/' "$DROPIN"
    refute_grep '^ExecStart=dotf ' "$DROPIN"
}

# The base unit's Restart=on-failure/RestartSec=1 would exhaust systemd's default
# start limit in ~5s on a boot where the vault is still locked, leaving the unit
# `failed` until someone runs reset-failed. The drop-in must neutralise that, or
# AC7's trade (no credential on disk) turns into a daemon that never comes back.
@test "AC7: the drop-in disables the start limit and backs the retry off" {
    grep -qF 'StartLimitIntervalSec=0' "$DROPIN"
    run grep -oE '^RestartSec=[0-9]+' "$DROPIN"
    [ -n "$output" ]
    [ "${output#RestartSec=}" -ge 10 ]
}

# StartLimitIntervalSec is a [Unit] directive; RestartSec is a [Service] one.
# systemd warns and ignores the former if it is put in [Service], which would
# silently restore the failure this drop-in exists to prevent.
@test "AC7: StartLimitIntervalSec is under [Unit], RestartSec under [Service]" {
    run awk '/^\[/{s=$0} /^StartLimitIntervalSec=/{print s}' "$DROPIN"
    [ "$output" = "[Unit]" ]
    run awk '/^\[/{s=$0} /^RestartSec=/{print s}' "$DROPIN"
    [ "$output" = "[Service]" ]
}

@test "AC7: no credential value is written into the drop-in itself" {
    refute_grep 'NAN_API_KEY=' "$DROPIN"
    refute_grep '^Environment=.*_KEY=' "$DROPIN"
    refute_grep '^EnvironmentFile=' "$DROPIN"
}

@test "AC7: setup-linux.sh deploys the drop-in directory" {
    grep -qF 'hive.service.d' "$DOTFILES_DIR/setup-linux.sh"
}

# --- AC8: the provider drop is complete, as a class ---

# The criterion is about hive's WORKER, so the assertion is scoped to the
# hive-vault entry's declared environment -- not a repo-wide grep, which would
# fire on ai/opencode/opencode.jsonc, where ollama and openrouter are the TUI's
# own providers and must survive. #1223's lesson applies: a guard that greps a
# whole file passes or fails for the wrong reason.
@test "AC8: hive-vault's declared env names no dropped provider" {
    run python3 - "$DOTFILES_DIR/ai/agy/mcp_servers.json" <<'PY'
import json, re, sys
doc = json.load(open(sys.argv[1]))
env = doc.get("mcpServers", {}).get("hive-vault", {}).get("env", {})
bad = [k for k in env if re.search(r'ollama|openrouter', k, re.I)]
bad += [k for k, v in env.items() if isinstance(v, str) and re.search(r'ollama|openrouter', v, re.I)]
print("OFFENDERS:" + ",".join(sorted(set(bad))) if bad else "CLEAN")
PY
    [ "$status" -eq 0 ]
    [ "$output" = "CLEAN" ]
}

# The removal must stay removed in the deployers too: either script re-adding a
# dropped provider into hive-vault's env would put the credential back on disk
# without touching the JSON this suite reads.
@test "AC8: neither setup script assigns a dropped provider into hive-vault's env" {
    refute_grep 'mcpServers\["hive-vault"\]\.env\.OPENROUTER_API_KEY' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'mcpServers\["hive-vault"\]\.env\.HIVE_OLLAMA_ENDPOINT' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'hive-vault"\.env\.OPENROUTER_API_KEY' "$DOTFILES_DIR/setup-windows.ps1"
    refute_grep 'hive-vault"\.env\.HIVE_OLLAMA_ENDPOINT' "$DOTFILES_DIR/setup-windows.ps1"
}

# The scope boundary, asserted rather than trusted: the spec explicitly keeps
# opencode's provider catalogue, and a future over-eager "remove ollama
# everywhere" sweep would be an unrelated regression wearing AC8's clothes.
@test "AC8: opencode's own provider catalogue is NOT collateral" {
    grep -qiE 'ollama|openrouter' "$DOTFILES_DIR/ai/opencode/opencode.jsonc"
}
