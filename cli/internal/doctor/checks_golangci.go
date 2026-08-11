package doctor

import (
	"fmt"
	"regexp"
)

// golangciVersionRe pulls the semver out of `golangci-lint --version`, which
// reads e.g. "golangci-lint has version 2.12.2 built with go1.26.0 from …".
// The `v` is optional on purpose: v1 printed "has version v1.62.2" and v2
// prints "has version 2.12.2", and a check that only understood one of them
// would report the other as unknown — turning a drift into a mystery.
var golangciVersionRe = regexp.MustCompile(`version\s+v?(\d+\.\d+\.\d+)`)

// golangciVersion extracts the installed semver, or "" when golangci-lint is
// absent or answers in a shape we do not recognise. trailingVersion is not
// usable here: the version is mid-line, and the last field is "(unknown)".
func golangciVersion(sys *System) string {
	out, err := sys.CommandOutput("golangci-lint", "--version")
	if err != nil {
		return ""
	}
	m := golangciVersionRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// checkGolangciLint compares the locally installed golangci-lint against the
// versions.conf pin that CI consumes (BUG-071).
//
// This exists because the absence of a linter is loud and a stale one is not.
// Before the pin, CI resolved the action's `latest` while this machine had a
// binary two majors behind; `golangci-lint run` reported "0 issues" locally and
// CI failed on the same commit, because v2 checks deferred calls in test files
// that v1 ignores. A local pass was not merely uninformative — it manufactured
// the confidence to push. Drift is a WARN rather than a FAIL, matching every
// other pinned-tool check here: the tool still works, it just cannot speak for
// CI.
func checkGolangciLint(sys *System, cfg *Config, rep *Report) {
	rep.Section("Go lint toolchain")

	pin := cfg.Versions["GOLANGCI_LINT_VERSION"]
	if !sys.has("golangci-lint") {
		// Optional tooling: absent is a SKIP, not a failure. The message names
		// the pin so the fix is a copy-paste rather than a lookup.
		if pin == "" {
			rep.Skip("golangci-lint not installed (CI lints the Go layer regardless)")
			return
		}
		rep.Skip(fmt.Sprintf(
			"golangci-lint not installed — CI gates the Go layer at v%s "+
				"(go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v%s)", pin, pin))
		return
	}

	installed := golangciVersion(sys)
	if installed == "" {
		rep.Warn("golangci-lint is on PATH but its --version output was not recognised — cannot verify it matches CI")
		return
	}
	matchPin(rep, "golangci-lint", installed, pin)
}
