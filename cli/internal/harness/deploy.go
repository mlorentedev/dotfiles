package harness

import "errors"

// Deployment is what Deploy did on each surface, in the order it ran them.
type Deployment struct {
	Instructions []InstructionOutcome
	Skills       []SkillOutcome
	Agents       []SkillOutcome
	Presence     []PresenceOutcome
	Doctrine     []DoctrineOutcome
	// AgentsErr is the agents surface's failure. When it is set, presence and
	// doctrine were not run, since both carry the roster those records feed.
	AgentsErr error
}

// Deploy runs every harness surface in the order compile-harness.sh --deploy
// runs them: instructions first, because the skill catalog and the presence
// roster are regions inside those files; then skills, agents, presence and
// doctrine. A failed surface does not stop the ones after it, except agents.
// Presence has no preview, so dryRun skips it. The returned error joins every
// surface's failure.
func Deploy(repoRoot, home string, has func(string) bool, dryRun bool) (Deployment, error) {
	var d Deployment
	var errs []error

	var err error
	d.Instructions, err = DeployInstructions(repoRoot, home, has, dryRun)
	errs = append(errs, err)

	d.Skills, err = DeploySkills(repoRoot, home, has, dryRun)
	errs = append(errs, err)

	d.Agents, d.AgentsErr = DeployAgents(repoRoot, home, dryRun)
	if d.AgentsErr != nil {
		return d, errors.Join(append(errs, d.AgentsErr)...)
	}

	if !dryRun {
		d.Presence, err = DeployPresence(repoRoot, home)
		errs = append(errs, err)
	}

	d.Doctrine, err = DeployDoctrine(repoRoot, home, dryRun)
	errs = append(errs, err)

	return d, errors.Join(errs...)
}
