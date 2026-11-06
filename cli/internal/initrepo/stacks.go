package initrepo

// codeDirs are the code directories a stack gets unless it declares its own.
var codeDirs = []string{"src", "tests", "scripts"}

// stackProfile is what one --stack value scaffolds beyond the shared structure.
type stackProfile struct {
	ci   string   // embedded CI workflow template; "" writes no ci.yml
	dirs []string // code directories
}

// stacks is the one place a stack is defined (#3093). A stack missing from it,
// unknown values included, gets codeDirs and no ci.yml. The Spec-Driven
// Development convention is documented in the generated AGENTS.md, not enforced
// by the ci templates: a portable spec-gate is deferred work (ADR-022).
var stacks = map[string]stackProfile{
	"go":     {ci: "ci-go.yml", dirs: codeDirs},
	"python": {ci: "ci-python.yml", dirs: codeDirs},
	"node":   {ci: "ci-node.yml", dirs: codeDirs},
	"ts":     {ci: "ci-node.yml", dirs: codeDirs},
	"none":   {dirs: codeDirs},
	// A repository of reusable workflows and composite actions: bats shape tests
	// live in tests/, logic lives in dotf, and lint.yml is its only CI.
	"workflows": {dirs: []string{"tests"}},
}

func stackOf(stack string) stackProfile {
	if p, ok := stacks[stack]; ok {
		return p
	}
	return stackProfile{dirs: codeDirs}
}
