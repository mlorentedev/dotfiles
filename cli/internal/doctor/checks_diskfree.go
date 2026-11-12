package doctor

import (
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/diskfree"
)

// checkDiskFree warns when the home volume is short of space and names the
// remedy. Under diskfree.Low the cache-gc converge step is due whatever its
// weekly stamp says, so the remedy works on any machine, with or without the
// self-deploy role that runs it unattended (#3367). A WARN, never a FAIL: a
// full disk is a state to act on, not a broken install.
func checkDiskFree(sys *System, rep *Report) {
	if sys.DiskFree == nil {
		return
	}
	rep.Section("Disk space")
	home := sys.home()
	if home == "" {
		rep.Skip("no HOME or USERPROFILE to measure")
		return
	}
	free, err := sys.DiskFree(home)
	if err != nil {
		rep.Warn("cannot measure free space on " + home + ": " + err.Error())
		return
	}
	if free < diskfree.Low {
		rep.Warn(fmt.Sprintf("%s free on the home volume, under %s: run `dotf converge --only cache-gc` to collect the tool caches (docs/runbooks/guide-self-deploy-timer.md)",
			gibString(free), gibString(diskfree.Low)))
		return
	}
	rep.Pass(gibString(free) + " free on the home volume")
}

func gibString(b uint64) string { return fmt.Sprintf("%.1f GiB", float64(b)/(1<<30)) }
