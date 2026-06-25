package doctor

import "path/filepath"

// checkSecretsTooling verifies the two binaries the ADR-028 secrets-governance
// model depends on are present, plus the age identity key. It is the governance
// hook the ADR (§Mitigations) and the secrets runbook both name.
//
// Absent bw or age is a FAIL: neither the live SSOT (Bitwarden) nor the DR /
// bootstrap floor (age) can run without them. A missing age key is a WARN, not a
// FAIL — a freshly provisioned machine legitimately has the tools before its key
// is restored from the offline backup (the recover runbook), and FAILing there
// would block an otherwise-healthy setup.
func checkSecretsTooling(sys *System, rep *Report) {
	rep.Section("Secrets tooling")

	if sys.has("bw") {
		rep.Pass("bw (Bitwarden CLI — live secrets SSOT) found")
	} else {
		rep.Fail("bw not in PATH — run 'dotf tools install' (ADR-028 live SSOT)")
	}

	if sys.has("age") {
		rep.Pass("age (DR escrow + bootstrap floor) found")
	} else {
		rep.Fail("age not in PATH — re-run setup (ADR-028 DR floor)")
	}

	keyPath := sys.Getenv("AGE_KEY_PATH")
	if keyPath == "" {
		keyPath = filepath.Join(sys.home(), ".config", "age", "key.txt")
	}
	if pathExists(keyPath) {
		rep.Pass("age identity key present (" + keyPath + ")")
	} else {
		rep.Warn("age identity key missing at " + keyPath + " — restore from offline backup (recover runbook)")
	}
}
