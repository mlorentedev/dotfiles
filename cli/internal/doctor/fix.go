package doctor

import (
	"path/filepath"
	"strings"
)

// runHeals is the --fix tail: it invokes the known heal scripts that still live
// in shell (claude-mem-heal.sh — itself out of scope for this port). A subprocess
// cannot export env into the parent shell, so the env-default "fixes" are
// reported as profile lines by checkContractEnvVars; the only durable action
// here is running the heal. Faithful to doctor.sh: a heal that errors or emits
// nothing is reported as "nothing to heal" rather than a failure.
func runHeals(sys *System, cfg *Config, rep *Report) {
	rep.Section("Heal scripts (--fix)")
	heal := filepath.Join(cfg.DotfilesDir, "scripts", "claude-mem-heal.sh")
	if !isExecFile(heal) {
		rep.Info("claude-mem-heal.sh not deployed (skipping)")
		return
	}
	out, err := sys.CommandOutput("bash", heal)
	out = strings.TrimSpace(out)
	if err == nil && out != "" {
		rep.Fix("claude-mem-heal:\n" + indentLines(out, "         "))
	} else {
		rep.Pass("claude-mem-heal: nothing to heal")
	}
}

// indentLines prefixes every line of s with prefix.
func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
