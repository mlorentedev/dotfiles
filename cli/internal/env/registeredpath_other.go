//go:build !windows

package env

// RegisteredPath has nothing to read off Windows: an installer here places
// its binary in a directory the caller's PATH or rc files already name, and
// no store holds a PATH for the next process.
func RegisteredPath() ([]string, error) { return nil, nil }
