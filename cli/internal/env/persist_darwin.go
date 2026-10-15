//go:build darwin

package env

import "os"

// NewUserEnvStore is the user's launchd session on macOS (LaunchdUserEnv).
func NewUserEnvStore() (UserEnvStore, error) {
	return LaunchdUserEnv{Run: ExecLaunchctl, UID: os.Getuid()}, nil
}
