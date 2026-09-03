package doctor

import (
	"path/filepath"
	"testing"
)

// setupShellCoverage is what setup-linux.sh's check_deployed and
// check_dependencies asserted immediately before OPS-043 deleted them. It is a
// deliberately FROZEN historical snapshot, not a parse: after the deletion there
// is no line left to read, so this table IS the record of what was removed.
//
// blocking says whether the shell call could fail the item. check_deployed
// log_error'd (blocking). check_dependencies only ever log_warning'd
// (scripts/utils.sh:359-372) — it never failed, not even under set -e — so its
// fourteen tools were advisory, and doctor need only be advisory to match.
var setupShellCoverage = []struct {
	name     string
	blocking bool
}{
	// check_deployed — setup-linux.sh:1583-1585
	{".zsh/aliases.zsh", true},
	{".zsh/functions.zsh", true},
	{".zsh/functions.sh", true},
	// check_dependencies — setup-linux.sh:1597
	{"git", false}, {"zsh", false}, {"eza", false}, {"direnv", false},
	{"node", false}, {"npm", false}, {"zoxide", false}, {"docker", false},
	{"docker-compose", false}, {"kubectl", false}, {"helm", false},
	{"terraform", false}, {"ansible", false}, {"pip", false},
}

// TestSetupShellParity is the evidence for the deletion in setup-linux.sh, and
// the reason this spec did not simply do what #1337 asked. Every item the two
// shell calls covered must have an equal-or-stronger verdict in doctor.
//
// Measured before the port, that was false in three places, which is why the
// port had to precede the delete: docker-compose was covered nowhere at all,
// .zsh/functions.sh was in no doctor list, and the deploy-dir→$HOME leg had no
// content comparison for any of the three files (checkSymlinks PASSes a drifted
// real file).
func TestSetupShellParity(t *testing.T) {
	c, err := loadContract(filepath.Join("..", "..", "..", "env-contract.json"))
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}

	core := map[string]bool{}
	for _, tool := range coreTools {
		core[tool] = true
	}
	optional := map[string]bool{}
	for _, tool := range optionalTools {
		optional[tool] = true
	}
	contractBins := contractBinaryNames(c)
	for _, o := range c.OptionalBinaries {
		optional[o.Name] = true
	}

	contentChecked := map[string]bool{}
	for _, e := range homeDeployMap {
		if e.contentChecked {
			contentChecked[e.dst] = true
		}
	}

	for _, item := range setupShellCoverage {
		t.Run(item.name, func(t *testing.T) {
			switch {
			// The three check_deployed files: only a byte comparison is
			// equal-or-stronger. Existence (checkSymlinks) is weaker — it
			// PASSes exactly the drift the shell call FAILed on.
			case item.blocking:
				if !contentChecked[item.name] {
					t.Fatalf("%s was byte-compared by check_deployed but has no content-checked homeDeployMap entry — deleting the shell call would lose the assertion", item.name)
				}

			// docker-compose has no PATH entry to find under compose v2; its
			// verdict comes from checkDockerCompose probing the plugin.
			case item.name == "docker-compose":
				if core[item.name] || optional[item.name] {
					t.Fatalf("docker-compose is listed as a PATH tool; compose v2 is a `docker` CLI plugin with no binary on PATH, so that answers wrongly on a current install")
				}

			// Everything else only needs to be *mentioned* somewhere doctor
			// reports, since the shell call was advisory. coreTools is strictly
			// stronger (FAIL); optionalTools is a SKIP, equal to the WARN it
			// replaces — both surface the tool without failing the run.
			default:
				if !core[item.name] && !optional[item.name] && !contractBins[item.name] {
					t.Fatalf("%s was reported by check_dependencies and appears in no doctor list (coreTools, optionalTools, contract binaries) — deleting the shell call would drop it entirely", item.name)
				}
			}
		})
	}
}

// TestSetupParityTableIsComplete guards the frozen table against being quietly
// trimmed: seventeen items were deleted from setup-linux.sh, and a parity claim
// over sixteen of them is not the claim this spec made.
func TestSetupParityTableIsComplete(t *testing.T) {
	const want = 17 // 3 check_deployed files + 14 check_dependencies tools
	if got := len(setupShellCoverage); got != want {
		t.Fatalf("setupShellCoverage has %d items, want %d — the table records what setup-linux.sh asserted at deletion time and is not editable after the fact", got, want)
	}
}
