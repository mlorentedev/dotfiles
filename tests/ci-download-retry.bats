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

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

@test "every CI download retries and writes a file" {
    files=("$REPO"/.github/workflows/*.yml "$REPO/tests/Dockerfile.integration")
    [ "${#files[@]}" -gt 1 ] || false
    no_retry=$(grep -nH 'curl -fsSL' "${files[@]}" | grep -v -e '--retry' || true)
    [ -z "$no_retry" ] || { printf 'a download without --retry:\n%s\n' "$no_retry" >&2; false; }
    piped=$(grep -nH 'curl -fsSL' "${files[@]}" | grep -v -e ' -o ' || true)
    [ -z "$piped" ] || { printf 'a download piped instead of written to a file:\n%s\n' "$piped" >&2; false; }
}

@test "the guard sees the downloads it is about" {
    n=$(grep -c 'curl -fsSL --retry 3 --retry-all-errors' "$REPO/.github/workflows/ci.yml" | tr -d ' ')
    [ "$n" -ge 6 ] || { echo "found $n retried downloads in ci.yml, want at least 6" >&2; false; }
}
