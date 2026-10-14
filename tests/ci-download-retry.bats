#!/usr/bin/env bats
# A CI download retries, and lands in a file before anything reads it.
#
# The incident: on 2026-10-11 the `test` job of #2338 failed on
# `curl: (35) Recv failure: Connection reset by peer` while fetching the age
# release, a network blip with nothing to do with the change. The download was
# `curl … | tar xz`, so it could not simply retry either: a retry after a partial
# transfer would feed tar a second copy behind the first half. So every
# `curl -fsSL` that fetches a file in CI writes it with `-o` and carries
# `--retry`.
#
# How long it retries matters as much (BUG-139, #2484). `--retry 3` alone
# backs off 1, 2 and 4 seconds, and on 2026-10-14 a GitHub release download
# answered 500 for longer than those 7 seconds. `--retry-delay` is a fixed
# wait (measured against a server that always answers 500: 3 retries at 2 s
# took 6 s), so `--retry 5 --retry-delay 10` keeps asking for about 50 s.
RETRY='--retry 5 --retry-delay 10 --retry-all-errors'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

@test "every CI download retries and writes a file" {
    files=("$REPO"/.github/workflows/*.yml "$REPO/tests/Dockerfile.integration")
    [ "${#files[@]}" -gt 1 ] || false
    no_retry=$(grep -nH 'curl -fsSL' "${files[@]}" | grep -vF -e "$RETRY" || true)
    [ -z "$no_retry" ] || { printf 'a download without %s:\n%s\n' "$RETRY" "$no_retry" >&2; false; }
    piped=$(grep -nH 'curl -fsSL' "${files[@]}" | grep -v -e ' -o ' || true)
    [ -z "$piped" ] || { printf 'a download piped instead of written to a file:\n%s\n' "$piped" >&2; false; }
}

@test "the guard sees the downloads it is about" {
    n=$(grep -cF "curl -fsSL $RETRY" "$REPO/.github/workflows/ci.yml" | tr -d ' ')
    [ "$n" -ge 6 ] || { echo "found $n retried downloads in ci.yml, want at least 6" >&2; false; }
    n=$(grep -cF "curl -fsSL $RETRY" "$REPO/tests/Dockerfile.integration" | tr -d ' ')
    [ "$n" -ge 4 ] || { echo "found $n retried downloads in Dockerfile.integration, want at least 4" >&2; false; }
}

# The same failure through apt: on 2026-10-14 the integration container could
# not resolve archive.ubuntu.com, and #1498 measured a mirror mid-sync. A bare
# `apt-get update` in the Dockerfile fails the job on either.
@test "every apt-get update in the integration image retries" {
    df="$REPO/tests/Dockerfile.integration"
    bare=$(grep -n 'apt-get update' "$df" | grep -v -e 'until apt-get update' -e '^[0-9]*:#' -e 'apt-get update failed' || true)
    [ -z "$bare" ] || { printf 'an apt-get update without apt-update-retry:\n%s\n' "$bare" >&2; false; }
    n=$(grep -c 'RUN apt-update-retry && apt-get install' "$df" | tr -d ' ')
    [ "$n" -ge 2 ] || { echo "found $n retried apt installs, want at least 2" >&2; false; }
    grep -q 'APT::Update::Error-Mode "any";' "$df" || { echo 'apt warnings would not fail apt-get update' >&2; false; }
    grep -q "apt_opts=.*-o APT::Update::Error-Mode=any'" "$REPO/.github/workflows/ci.yml" \
        || { echo "ci.yml's apt retry loop would not retry a mirror that does not resolve" >&2; false; }
}
