#!/usr/bin/env bats
# #2013 P7: the files setup copied into $HOME are ai/deploy.json entries, so
# one implementation deploys them on every OS (`dotf deploy`, `dotf converge`),
# backs up a file the machine already had, and doctor compares them through
# checkDeployManifest. A deploy_file call into $HOME would bring back a second
# deployer, and a second mapping doctor no longer reads.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

@test "setup-linux.sh deploys no file into \$HOME with deploy_file" {
    run grep -nE 'deploy_file[[:space:]]+"\$DOTFILES_DIR/[^"]+"[[:space:]]+"\$HOME/' "$REPO/setup-linux.sh"
    [ "$status" -eq 1 ] || false
}

@test "the bash and readline rc files are deploy entries for linux and darwin" {
    run python3 - "$REPO/ai/deploy.json" <<'PY'
import json, sys
m = json.load(open(sys.argv[1]))
by = {c["name"]: c for c in m["configs"]}
for name in ("bashrc", "profile", "inputrc"):
    c = by.get(name)
    assert c, f"no {name} entry"
    assert c["src"] == "." + name, c
    assert c["dst"] == "{HOME}/." + name, c
    assert c.get("platforms") == ["linux", "darwin"], c
    assert not c.get("render"), c
assert m["version"] >= 4, m["version"]
PY
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "setup re-deploys the rc files through dotf at its tail" {
    grep -qE 'for _rc in zshrc bashrc profile; do' "$REPO/setup-linux.sh"
}
