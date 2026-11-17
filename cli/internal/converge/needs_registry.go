package converge

// What each reconciler in Registry needs (BUG-173, ADR-064). A need is an
// earlier step whose failure makes running this one unsafe: it reads what the
// earlier step writes, or a decision says it must not run without it. A step
// that only happens to come later needs nothing from the one before it.
//
// An edge that is missing runs a step against a prerequisite that failed; an
// edge too many only reports a step as not reached. So an uncertain edge is
// kept, and one is removed only with the code that shows it is not read. The
// evidence for each list is in specs/BUG-173-converge-step-dependencies.
//
// legacy-setup declares nothing on purpose: the setup script re-runs native
// work and runs after every native step (ADR-045 decision 5), so it needs
// them all, which is what a reconciler that is not a Needer gets.

const (
	nCheckout       = "checkout"
	nRecordsMirror  = "records-mirror"
	nEnvGenerate    = "env-generate"
	nRecordsHarness = "records-harness"
	nTools          = "tools"
	nPiSettings     = "pi-settings"
	nConfigsDeploy  = "configs-deploy"
	nRecordsBind    = "records-bind"
	nGitHooks       = "git-hooks"
)

// checkout is the root: every other step reads the checkout.
func (checkout) Needs() []string { return nil }

// The records (ADR-041 decision 4, ADR-045 decision 4): the mirror reads the
// checkout, and the instruction files follow the mirror.
func (recordsMirror) Needs() []string  { return []string{nCheckout} }
func (envGenerate) Needs() []string    { return []string{nCheckout, nRecordsMirror} }
func (recordsHarness) Needs() []string { return []string{nCheckout, nRecordsMirror} }

// No agent is installed before its instructions exist, nor before the paths
// env-generate writes.
func (toolsSync) Needs() []string {
	return []string{nCheckout, nRecordsMirror, nEnvGenerate, nRecordsHarness}
}

// These read the catalog or the pins in the checkout and use what tools
// installs. Nothing needs them.
func (systemUpgrade) Needs() []string    { return []string{nCheckout, nTools} }
func (cacheGC) Needs() []string          { return []string{nCheckout, nTools} }
func (pwshModules) Needs() []string      { return []string{nCheckout, nTools} }
func (shellCompletions) Needs() []string { return []string{nCheckout, nTools} }
func (gitLFS) Needs() []string           { return []string{nCheckout, nTools} }
func (envPersist) Needs() []string       { return []string{nCheckout, nTools} }

// pi-settings seeds the file configs-deploy then merges into (#1948).
func (piSettings) Needs() []string { return []string{nCheckout, nTools} }
func (configsDeploy) Needs() []string {
	return []string{nCheckout, nRecordsMirror, nTools, nPiSettings}
}

// The bindings are merged into the settings configs-deploy seeds, for the
// agents whose instructions records-harness wrote.
func (recordsBind) Needs() []string {
	return []string{nCheckout, nRecordsMirror, nRecordsHarness, nTools, nConfigsDeploy}
}

// git-config's post-condition is the gitconfig configs-deploy writes.
func (gitConfig) Needs() []string { return []string{nCheckout, nTools, nConfigsDeploy} }
func (gitHooks) Needs() []string  { return []string{nCheckout, nTools} }

// Activating a lane creates worktrees, which fires the hooks git-hooks set.
func (lanes) Needs() []string {
	return []string{nCheckout, nRecordsMirror, nTools, nGitHooks}
}

// These read the machine's roles, the scheduler or the checkout alone.
func (orcaHooks) Needs() []string         { return []string{nCheckout} }
func (tsBridgeClient) Needs() []string    { return []string{nCheckout} }
func (selfDeploy) Needs() []string        { return []string{nCheckout} }
func (hiveUpgradeRetire) Needs() []string { return []string{nCheckout} }
func (workstationTarget) Needs() []string { return []string{nCheckout} }

// The extensions go into the settings configs-deploy and records-bind write.
func (agentExtensions) Needs() []string {
	return []string{nCheckout, nRecordsMirror, nRecordsHarness, nTools, nConfigsDeploy, nRecordsBind}
}
