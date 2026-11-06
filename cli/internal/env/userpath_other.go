//go:build !windows

package env

// NewUserPathStore has no user Path store to offer off Windows: the rc files
// put the tool directories on PATH (.zsh/path.sh), and macOS GUI apps get
// theirs from the launchd session, not from a persisted Path value.
func NewUserPathStore() (UserPathStore, error) { return nil, ErrUserEnvUnsupported }

// ExpandUserPath is the identity off Windows, where no stored Path holds
// `%VAR%` references.
func ExpandUserPath(s string) string { return s }
