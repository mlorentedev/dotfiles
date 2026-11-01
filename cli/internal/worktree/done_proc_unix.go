//go:build linux || darwin

package worktree

// maxAncestorDepth bounds the ancestor walk. A real chain (dotf, a shell, an
// agent, a terminal, a session manager, init) is under a dozen; the bound only
// keeps a malformed process table from looping. Only the Linux and macOS walks
// read it: declared beside them, a Windows build has no unused constant (#3014).
const maxAncestorDepth = 64
