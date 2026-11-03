#!/usr/bin/env bats
# #2013 PLAT-001b: the client loads its key into the agent on first use, and on
# macOS keeps the passphrase in the login keychain, so no session runs ssh-add
# by hand (`ssh-add --apple-use-keychain` was a step of the msi acceptance).
# ssh/config is deployed to every OS, and UseKeychain exists only in Apple's
# OpenSSH: any other build rejects the whole file over an unknown keyword
# unless IgnoreUnknown names it first.

# os-sensitive: the keychain half runs only on macOS, so test-macos must run
# this file on every pull request, not only on pushes to main.
# bats file_tags=os-sensitive

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CONF="$REPO/ssh/config"
}

@test "ssh/config parses on this OpenSSH and adds keys to the agent" {
    command -v ssh >/dev/null 2>&1 || skip "no ssh client"
    run ssh -F "$CONF" -G msi </dev/null
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [[ "$output" == *$'\naddkeystoagent true'* ]] || { echo "$output" | grep -i addkeys; false; }
}

@test "UseKeychain is set for every host, after an IgnoreUnknown that names it" {
    block="$(awk '/^Host \*$/{f=1; next} f && /^(Host|Match) /{exit} f' "$CONF")"
    [ -n "$block" ] || { echo "no 'Host *' block"; false; }
    ignore=$(printf '%s\n' "$block" | grep -n '^[[:space:]]*IgnoreUnknown[[:space:]]\{1,\}UseKeychain$' | cut -d: -f1)
    use=$(printf '%s\n' "$block" | grep -n '^[[:space:]]*UseKeychain[[:space:]]\{1,\}yes$' | cut -d: -f1)
    [ -n "$ignore" ] && [ -n "$use" ] || { printf '%s\n' "$block"; false; }
    [ "$ignore" -lt "$use" ] || false
    printf '%s\n' "$block" | grep -q '^[[:space:]]*AddKeysToAgent[[:space:]]\{1,\}yes$'
}

# The shipped line, without the IgnoreUnknown that would hide a keyword Apple's
# ssh rejected: in the shipped file a typo would parse and do nothing.
@test "Apple's OpenSSH accepts the shipped UseKeychain line, so the keychain half is real on macOS" {
    [ "$(uname -s)" = Darwin ] && [ -x /usr/bin/ssh ] || skip "not macOS"
    line="$(awk '/^Host \*$/{f=1; next} f && /^(Host|Match) /{exit} f && /^[[:space:]]*UseKeychain[[:space:]]/' "$CONF")"
    [ -n "$line" ] || false
    printf 'Host *\n%s\n' "$line" > "$BATS_TEST_TMPDIR/apple.conf"
    run /usr/bin/ssh -F "$BATS_TEST_TMPDIR/apple.conf" -G msi </dev/null
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

# kubelab VPN-ACL-015: ts-bridge's control-plane bootstrap reaches the VPS through
# this alias alone. It must offer only its own key and refuse an unpinned host key.
@test "hs-bootstrap offers only its own key and pins the VPS host key" {
    command -v ssh >/dev/null 2>&1 || skip "no ssh client"
    run ssh -F "$CONF" -G hs-bootstrap </dev/null
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    for want in 'user hs-bootstrap' 'identitiesonly yes' 'stricthostkeychecking true' \
                'identityfile ~/.ssh/hs-bootstrap'; do
        [[ "$output" == *$'\n'"$want"* ]] || { echo "missing: $want"; echo "$output" | grep -iE 'user |identit|stricthost|knownhosts'; false; }
    done
    printf '%s\n' "$output" | grep -q '^userknownhostsfile .*/known_hosts_pinned$' || { echo "$output" | grep knownhosts; false; }
    [ "$(printf '%s\n' "$output" | grep -c '^identityfile ')" -eq 1 ] || { echo "$output" | grep '^identityfile'; false; }
}