package update

import (
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
)

// NativeBackend is goos's real scheduler: launchctl, systemctl --user or
// schtasks, addressed for the user whose home is home and whose environment
// getenv reads. uid names the launchd domain; elevated decides a task's logon
// type. The converge step installs the trigger through it and doctor inspects
// the trigger through it, so the two cannot address different schedulers.
func NativeBackend(goos, home string, getenv func(string) string, uid int, elevated func() bool) (schedule.Backend, error) {
	config := getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	}
	user := getenv("USERNAME")
	if d := getenv("USERDOMAIN"); d != "" && user != "" {
		user = d + `\` + user
	}
	return schedule.For(goos, schedule.Runners{
		Launchctl: schedule.Runner(env.ExecLaunchctl),
		UID:       uid,
		AgentsDir: filepath.Join(home, "Library", "LaunchAgents"),
		Systemctl: schedule.Exec("systemctl", "--user"),
		UnitDir:   filepath.Join(config, "systemd", "user"),
		Schtasks:  schedule.Exec("schtasks"),
		TaskUser:  user,
		Elevated:  elevated,
		TempDir:   os.TempDir(),
	})
}
