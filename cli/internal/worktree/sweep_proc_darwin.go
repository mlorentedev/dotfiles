//go:build darwin

package worktree

import "path/filepath"

const processDiscoverySupported = true

// isHostProcessInside is Gate f on darwin, from one ps and lsof listing. Its
// answers follow the Linux gate: a failure that prevents the scan answers
// Inside (the caller deletes on a false), and a process whose cwd lsof could
// not read counts as Uninspectable, never as outside. The target is resolved
// first, because lsof reports the physical path: $TMPDIR's /var/folders is
// /private/var/folders to every process.
func isHostProcessInside(targetPath string) GateFReading {
	abs, err := filepath.Abs(targetPath)
	if err != nil {
		return GateFReading{Inside: true}
	}
	target, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return GateFReading{Inside: true}
	}
	procs, err := readProcTable()
	if err != nil {
		return GateFReading{Inside: true}
	}
	reading := GateFReading{}
	for _, p := range procs {
		if p.Cwd == "" {
			reading.Uninspectable++
			continue
		}
		if pathWithin(p.Cwd, target) {
			return GateFReading{Inside: true}
		}
	}
	return reading
}
