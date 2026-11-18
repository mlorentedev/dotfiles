package doctor

import (
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// checkAgentLaunchers says, on Windows, which file a launch of each agent
// starts, before the first launch finds out (WIN-015, #3572).
//
// dotf refuses to start an agent through a batch file: cmd.exe would re-parse
// the prompt (#3429). npm installs an agent as a .cmd, and since #3569 a launch
// starts the native executable the tool manager holds behind such a shim, and
// is refused only when there is none. Which of the two a machine is in was
// learned when a lane or a `dotf spec review` failed to start.
//
// The answer is lane.NativeTarget's, the function the launch itself asks, so
// the check cannot disagree with the launch about the rule. It resolves and
// reads the file system; it starts no agent. The agents are the lane launch
// table's, which holds every runner the review launcher knows (pi, agy).
//
// A batch-only launcher is a warning, not a failure: the agent works from the
// shell, which does not go through dotf, and only dotf's own launches refuse it.
func checkAgentLaunchers(sys *System, rep *Report) {
	rep.Section("Agent launchers (what dotf starts)")
	if sys.GOOS != "windows" {
		rep.Skip("only Windows starts a batch file through cmd.exe")
		return
	}
	behind := sys.NativeBehind
	if behind == nil {
		behind = func(string) string { return "" }
	}
	for _, agent := range lane.Agents() {
		onPath, err := sys.LookPath(agent)
		if err != nil {
			rep.Skip(agent + ": not installed")
			continue
		}
		target, err := lane.NativeTarget(agent, sys.LookPath, behind)
		switch {
		case err != nil:
			rep.Warn(fmt.Sprintf("%s: the only launcher is the batch file %s, which dotf lane and dotf spec review refuse to start. "+
				"Install a native build (`mise use -g %s` where mise has one)", agent, onPath, agent))
		case target == onPath:
			rep.Pass(fmt.Sprintf("%s: starts %s", agent, target))
		default:
			rep.Pass(fmt.Sprintf("%s: starts %s, the native executable behind the batch file %s", agent, target, onPath))
		}
	}
}
