package doctor

import (
	"encoding/json"
	"fmt"
	"os"
)

// Contract is the native Go model of env-contract.json. The shell twin shelled
// out to jq to read this; parsing it with encoding/json removes that runtime
// dependency (doctor.sh hard-failed with exit 2 when jq was absent).
type Contract struct {
	EnvVars             []ContractEnvVar    `json:"env_vars"`
	RequiredPathEntries map[string][]string `json:"required_path_entries"`
	RequiredBinaries    []ContractBinary    `json:"required_binaries"`
	OptionalBinaries    []ContractOptional  `json:"optional_binaries"`
}

// ContractEnvVar declares a structural environment variable: whether it is
// required (globally or per-OS), an optional per-OS default to apply, and an
// optional validation (only "path_exists" today).
type ContractEnvVar struct {
	Name       string            `json:"name"`
	Required   bool              `json:"required"`
	RequiredOn string            `json:"required_on"`
	Default    map[string]string `json:"default"`
	Validation string            `json:"validation"`
}

// requiredOnLinux reports whether this var must be present on Linux: either
// globally required, or scoped required_on: linux.
func (e ContractEnvVar) requiredOnLinux() bool {
	return e.Required || e.RequiredOn == "linux"
}

// ContractBinary is a required binary with an optional pinned minimum version
// and the regex used to extract the installed version from `--version` output.
type ContractBinary struct {
	Name           string `json:"name"`
	Required       bool   `json:"required"`
	MinVersion     string `json:"min_version"`
	VersionPattern string `json:"version_pattern"`
}

// ContractOptional is an optional binary reported as INFO when absent.
type ContractOptional struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
}

// loadContract reads and parses the env-contract.json at path. A missing or
// malformed contract is a hard error (the env-contract sweep cannot run without
// it), surfaced to the caller rather than silently skipped.
func loadContract(path string) (*Contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env-contract.json: %w", err)
	}
	var c Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse env-contract.json (%s): %w", path, err)
	}
	return &c, nil
}
