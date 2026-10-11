//go:build darwin

package env

// NewUserEnvStore is the user's launchd session on macOS (LaunchdUserEnv).
func NewUserEnvStore() (UserEnvStore, error) { return LaunchdUserEnv{Run: ExecLaunchctl}, nil }
