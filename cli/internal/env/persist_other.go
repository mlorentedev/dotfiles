//go:build !windows

package env

// NewUserEnvStore has no per-user persistent scope to offer on this OS: the
// rc files source paths.sh, and a profile-less process (systemd unit, cron)
// takes its environment from its own unit file, not from a user registry.
func NewUserEnvStore() (UserEnvStore, error) { return nil, ErrUserEnvUnsupported }
