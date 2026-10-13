#!/usr/bin/env bats
# The pinned known_hosts (#2356): ssh/config sends a host to it with
# UserKnownHostsFile and refuses any other key with StrictHostKeyChecking yes.
# A pinned host with no line can never be reached, and a line ssh cannot parse
# is ignored silently, so both fail here instead of at the first login.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    PINS="$REPO/ssh/known_hosts"
    CONFIG="$REPO/ssh/config"
}

# "<host>|<hostname>" for every Host block that pins, one per line.
_pinned_hosts() {
    awk '
        /^Host /       { if (pin) print host "|" name; host = $2; name = $2; pin = 0; next }
        /^[[:space:]]*Hostname /                 { name = $2 }
        /^[[:space:]]*StrictHostKeyChecking yes/ { pin = 1 }
        END            { if (pin) print host "|" name }
    ' "$CONFIG"
}

@test "ssh-known-hosts: every pinned line parses as a host key" {
    run ssh-keygen -lf "$PINS"
    [ "$status" -eq 0 ]
    keys=$(grep -c -v -e '^#' -e '^$' "$PINS" | tr -d ' ')
    [ "${#lines[@]}" -eq "$keys" ]
}

@test "ssh-known-hosts: every host that pins has a key for its Hostname" {
    rows=$(_pinned_hosts)
    [ -n "$rows" ]
    while IFS='|' read -r host name; do
        ssh-keygen -F "$name" -f "$PINS" >/dev/null || {
            echo "$host pins, but ssh/known_hosts has no key for $name"
            return 1
        }
    done <<< "$rows"
}

@test "ssh-known-hosts: every host that pins reads the deployed pin file" {
    dst=$(jq -r '.. | objects | select(.src? == "ssh/known_hosts") | .dst' "$REPO/ai/deploy.json")
    [ "$dst" = "{HOME}/.ssh/known_hosts_pinned" ]
    while IFS='|' read -r host _; do
        file=$(awk -v h="$host" '
            /^Host / { in_block = ($2 == h); next }
            in_block && /^[[:space:]]*UserKnownHostsFile / { print $2 }
        ' "$CONFIG")
        [ "$file" = "~/.ssh/known_hosts_pinned" ] || {
            echo "$host pins without UserKnownHostsFile ~/.ssh/known_hosts_pinned (got '$file')"
            return 1
        }
    done <<< "$(_pinned_hosts)"
}

# Each key carries the fingerprint it was verified against on the comment line
# above it. A key edited by one character is still valid base64 and parses, so
# only this comparison catches it.
@test "ssh-known-hosts: every key matches the fingerprint recorded above it" {
    prev=""
    while IFS= read -r line; do
        case "$line" in
            '# SHA256:'*) prev=${line#\# }; continue ;;
            '#'* | '') continue ;;
        esac
        got=$(printf '%s\n' "$line" | ssh-keygen -lf - | awk '{print $2}')
        [ -n "$prev" ] && [ "$got" = "$prev" ] || {
            echo "key for ${line%% *} is $got, recorded as '${prev:-nothing}'"
            return 1
        }
        prev=""
    done < "$PINS"
}
