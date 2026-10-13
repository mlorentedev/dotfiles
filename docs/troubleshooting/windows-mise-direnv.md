# direnv installed by mise but not found on Windows

`mise install direnv@2.37.1` can report success while `mise which direnv`
reports no executable. On Windows, the aqua recipe used by the short name
downloads `direnv.windows-amd64` as a PE binary without the `.exe` suffix.
Both mise 2026.9.13 and 2026.10.8 leave it undiscoverable.

The `dotf tools sync` Windows renderer selects `github:direnv/direnv` with
`bin = "direnv.exe"` instead. The GitHub backend chooses the release asset
for the current OS/architecture, verifies the download, and gives the
single-file binary an executable name. With the pinned mise, an isolated
install resolved through `mise which direnv`, ran `direnv --version` at
2.37.1, and installed no tools on a second pass. Do not copy or rename
files in a setup script: that would conceal a broken declarative install.

Windows mise remains excluded from `packages.json` until a `DOTF_VERSION`
release includes this Windows renderer and the scoped npm-backed bats
installer. An isolated Windows run proved all 28 pins and a zero-work
second sync; a smaller catalog fixture proved `dotf converge --only tools`
installs mise and four pins from zero and reports `0 changed` on its
second pass. The full production catalog still needs a from-zero rollout
after the reader is released.
