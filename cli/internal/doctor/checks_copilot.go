package doctor

// checkCopilot reports the GitHub Copilot CLI against its packages.json pin
// (AI-038, #1321; ADR-036 amendment): copilot is an npm catalog tool on every
// OS, so its version is probed once in Go (tools.ProbeVersion via semverOf)
// and compared with the catalog pin on a floor basis — `dotf tools install`'s
// decideAction never downgrades a newer install, so doctor must agree that
// ahead-of-pin is healthy, not drift (see matchPinFloorFrom). Absent is a
// SKIP: a box may deliberately not carry Copilot, and setup only deploys its
// config when the binary is on PATH. A leftover winget or scoop copy beside
// the npm one is reported by checkShadowedCatalogTools, which reads the same
// catalog.
func checkCopilot(sys *System, cfg *Config, rep *Report) {
	rep.Section("GitHub Copilot CLI")
	if !sys.has("copilot") {
		rep.Skip("copilot not installed (run `dotf tools install copilot`; needs Node.js on PATH)")
		return
	}
	ver := semverOf(sys, "copilot")
	if ver == "" || ver == "unknown" {
		rep.Warn("copilot in PATH but `copilot --version` yields no semver — pin match not verified")
		return
	}
	rep.Pass("copilot in PATH: " + ver)
	matchPinFloorFrom(rep, "copilot", ver, catalogPin(sys, cfg, "copilot"), "packages.json")
}
