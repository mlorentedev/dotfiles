#!/usr/bin/env bats
# The integration container must test the dotf built from the tree under
# test, not the last release. setup-linux.sh's install_dotf downloaded 0.51.0
# into the container, which had no `harness mirror`, so the job failed on the
# very PR that added the command (#1305) -- and would have kept certifying the
# previous release's behaviour on every PR after it. Same principle as the
# Windows job (TEST-003, #1298).
#
# Asserted on the Dockerfile / workflow / installer text, because nothing here
# can run Docker or GitHub's runner.

setup() {
    export REPO="$BATS_TEST_DIRNAME/.."
    export DOCKERFILE="$REPO/tests/Dockerfile.integration"
    export CI="$REPO/.github/workflows/ci.yml"
}

@test "integration: the container builds dotf from this tree into ~/.local/bin" {
    grep -qF 'go build -o /home/testuser/.local/bin/dotf ./cmd/dotf' "$DOCKERFILE"
    # The build must follow the COPY of the repo (it compiles what was copied).
    copy_line=$(grep -n '^COPY --chown=testuser:testuser \. /home/testuser/dotfiles-repo' "$DOCKERFILE" | head -1 | cut -d: -f1)
    build_line=$(grep -n 'go build -o /home/testuser/.local/bin/dotf' "$DOCKERFILE" | head -1 | cut -d: -f1)
    [ -n "$copy_line" ] && [ -n "$build_line" ]
    [ "$build_line" -gt "$copy_line" ]
}

@test "integration: DOTF_VERSION=dev keeps the source build through install_dotf" {
    grep -qF 'ENV DOTF_VERSION=dev' "$DOCKERFILE"
    setup_line=$(grep -n 'bash setup-linux.sh' "$DOCKERFILE" | head -1 | cut -d: -f1)
    env_line=$(grep -n '^ENV DOTF_VERSION=dev' "$DOCKERFILE" | head -1 | cut -d: -f1)
    [ "$env_line" -lt "$setup_line" ]
}

@test "integration: GO_VERSION reaches the container from cli/go.mod, not a second pin" {
    grep -qF "GO_VERSION=\$(sed -n 's/^go //p' cli/go.mod)" "$CI"
    grep -qF -- '--build-arg GO_VERSION="$GO_VERSION"' "$CI"
    grep -qF 'ARG GO_VERSION=' "$DOCKERFILE"
}

@test "both installers treat a 'dev' build as already installed when the pin is dev" {
    grep -qF "grep -oE '[0-9]+\\.[0-9]+\\.[0-9]+|dev'" "$REPO/scripts/install-dotf.sh"
    grep -qF "(\\d+\\.\\d+\\.\\d+|dev)" "$REPO/scripts/install-dotf.ps1"
}
