#!/usr/bin/env bats
# HARNESS-075 (#1124): the routing registry that ADR-032 specified and ADR-035 shaped.
#
# The Go layer owns the schema interpreter and the doctor check; this suite
# guards the properties a shell reader can assert about the shipped artifacts
# themselves — that they exist, parse, and carry the decisions the ADRs record.
#
# The retired-provider cases are structural rather than textual on purpose. The
# `$comment` names both openrouter and codex so their absence reads as a
# decision to whoever wonders where they went, and a grep-based guard would
# fail on that very sentence.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    MAP="$REPO/harness/model-map.json"
    SCHEMA="$REPO/harness/model-map.schema.json"
}

# --- the artifacts exist and parse ---

@test "model-map.json exists and is valid JSON" {
    [ -f "$MAP" ]
    run python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$MAP"
    [ "$status" -eq 0 ]
}

@test "model-map.schema.json exists and is valid JSON" {
    [ -f "$SCHEMA" ]
    run python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$SCHEMA"
    [ "$status" -eq 0 ]
}

@test "model-map.json carries all seven declared blocks" {
    run python3 -c '
import json,sys
d=json.load(open(sys.argv[1]))
need={"$comment","version","pools","harnesses","tiers","chains","services"}
missing=need-set(d)
if missing:
    print("missing:", sorted(missing)); sys.exit(1)
' "$MAP"
    [ "$status" -eq 0 ]
}

# --- the ADR-035 amendments, asserted structurally ---

@test "no retired provider is declared as a pool or referenced by a harness" {
    run python3 -c '
import json,sys
d=json.load(open(sys.argv[1]))
bad=[]
for dead in ("openrouter","codex"):
    if dead in d.get("pools",{}):    bad.append("pools declares "+dead)
    if dead in d.get("harnesses",{}): bad.append("harnesses declares "+dead)
    for h,cfg in d.get("harnesses",{}).items():
        if dead in cfg.get("pools",[]): bad.append(h+" references "+dead)
if bad:
    print("\n".join(bad)); sys.exit(1)
' "$MAP"
    [ "$status" -eq 0 ]
}

@test "every harness pool reference resolves to a declared pool" {
    run python3 -c '
import json,sys
d=json.load(open(sys.argv[1]))
pools=set(d.get("pools",{}))
dangling=[f"{h}.pools[] names {p}" for h,cfg in d.get("harnesses",{}).items()
          for p in cfg.get("pools",[]) if p not in pools]
if dangling:
    print("\n".join(dangling)); sys.exit(1)
' "$MAP"
    [ "$status" -eq 0 ]
}

@test "the top tier has no fallback, so it queues or escalates rather than degrading" {
    run python3 -c '
import json,sys
d=json.load(open(sys.argv[1]))
top=d.get("chains",{}).get("top",[])
sys.exit(0 if len(top)==1 else 1)
' "$MAP"
    [ "$status" -eq 0 ]
}

# --- the contract that keeps declaration honest ---

@test "the \$comment states that concurrency is declared and not enforced" {
    run grep -q "DECLARED, NOT ENFORCED" "$MAP"
    [ "$status" -eq 0 ]
}

@test "model-map.json is not embedded into the Go tree, unlike triggers.json" {
    # An embedded copy would give an absent map a build-time default to fall back
    # to, and the doctor check would then certify that default as healthy — which
    # is what constraint C15 forbids. triggers.json carries exactly that drift
    # today (#1137); this asserts the routing map never grows it.
    [ ! -f "$REPO/cli/internal/harness/model-map.json" ]
    run grep -q "go:embed model-map.json" "$REPO/cli/internal/harness/model_map.go"
    [ "$status" -ne 0 ]
}
