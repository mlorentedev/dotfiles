package lane

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// miseWhichLimit bounds the one question NativeTarget asks mise. A launch that
// waits on it is a launch that has not started.
const miseWhichLimit = 15 * time.Second

// NativeTarget is the file a launch of name starts on Windows: what PATH
// resolves, unless that is a batch shim and the tool manager holds a native
// executable of the same name (#3569). npm installs an agent as a .cmd, and
// mise's own shim is a .cmd too when mise-shim.exe is not beside mise, so the
// first `pi` on PATH is a batch file on a machine whose mise has pi.exe
// installed. Starting that executable is the native launch RefuseBatchShim
// asks for: cmd.exe never sees the prompt.
//
// look is exec.LookPath and behind is MiseNative; both are parameters so the
// decision is tested without a PATH or a mise. An error from look is returned
// as it is. Every other failure is the batch-shim refusal for what PATH
// resolved, so a machine with no native copy reads the error it read before.
//
// Only a bare command name is looked up behind its shim. A caller that names a
// file (`C:\npm\pi.cmd`, `pi.cmd`) asked for that file, and it is refused.
// What behind answers is not trusted. It must be an absolute path to a regular
// file whose name ends in .exe. mise answers for an npm tool with the package's
// extensionless shell stub (`installs\opencode\1.16.2\opencode`), which has an
// opencode.cmd beside it: started by that name, Windows appends PATHEXT and
// runs the batch file after all. Those tools stay refused.
func NativeTarget(name string, look func(string) (string, error), behind func(string) string) (string, error) {
	bin, err := look(name)
	if err != nil {
		return "", err
	}
	refusal := RefuseBatchShim(name, bin)
	if refusal == nil {
		return bin, nil
	}
	if !bareCommand(name) {
		return "", refusal
	}
	native := behind(name)
	if !filepath.IsAbs(native) || !strings.EqualFold(filepath.Ext(native), ".exe") {
		return "", refusal
	}
	if fi, err := os.Stat(native); err != nil || !fi.Mode().IsRegular() {
		return "", refusal
	}
	return native, nil
}

// bareCommand reports whether name is a command name alone: no directory, no
// drive, no extension, and nothing a command line would read as a flag.
func bareCommand(name string) bool {
	return name != "" && !strings.HasPrefix(name, "-") &&
		!strings.ContainsAny(name, `/\:`) && filepath.Ext(name) == ""
}

// MiseNative is the path mise resolves for the executable name, or "" when
// mise is absent, does not know the name, or does not answer in time. mise is
// started only where it is a native executable itself: a mise that is a batch
// file would hand name to cmd.exe. The answer is not trusted here; NativeTarget
// checks it.
func MiseNative(name string) string {
	mise, err := exec.LookPath("mise")
	if err != nil || RefuseBatchShim("mise", mise) != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), miseWhichLimit)
	defer cancel()
	out, err := exec.CommandContext(ctx, mise, "which", "--", name).Output() //nolint:gosec // mise from PATH, name a bare command name
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
