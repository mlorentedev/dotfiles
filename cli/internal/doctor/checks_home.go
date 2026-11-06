package doctor

import (
	"regexp"
	"strings"
)

// checkForeignHome (#2432) FAILs on Windows when the user-scope persistent HOME
// names a directory other than USERPROFILE. git, ssh and env.Home read HOME
// first while Go's os.UserHomeDir reads USERPROFILE, so a HOME another
// installer persisted (a Cadence install wrote C:\SPB_Data) splits the machine
// into two homes with no other signal. Only the persisted scope is read: a
// process HOME, such as the one Git Bash exports, is not what a new process
// inherits.
func checkForeignHome(sys *System, rep *Report) {
	if sys.GOOS != "windows" || sys.UserEnv == nil {
		return
	}
	rep.Section("Home directory (Windows)")
	home, ok, err := sys.UserEnv("HOME")
	if err != nil {
		rep.Warn("user-scope environment unreadable (" + err.Error() + ")")
		return
	}
	if !ok || home == "" {
		rep.Pass("no persisted user-scope HOME (a shell may still export its own)")
		return
	}
	profile := sys.Getenv("USERPROFILE")
	if sameWindowsPath(expandPercent(home, sys.Getenv), profile) {
		rep.Pass("user-scope HOME agrees with USERPROFILE (" + profile + ")")
		return
	}
	rep.Fail("user-scope HOME is " + home + " but USERPROFILE is " + profile +
		": git and ssh read one, part of dotf the other. If dotf did not set it, an installer did; remove it with " +
		"[Environment]::SetEnvironmentVariable('HOME', $null, 'User') and open a new shell")
}

var percentVar = regexp.MustCompile(`%([^%]+)%`)

// expandPercent expands %NAME% the way Windows expands a REG_EXPAND_SZ value,
// leaving an unknown name as written.
func expandPercent(s string, getenv func(string) string) string {
	return percentVar.ReplaceAllStringFunc(s, func(m string) string {
		if v := getenv(m[1 : len(m)-1]); v != "" {
			return v
		}
		return m
	})
}

// sameWindowsPath compares two Windows paths as the filesystem does: case-insensitive,
// either separator, no trailing separator.
func sameWindowsPath(a, b string) bool {
	norm := func(p string) string { return strings.TrimRight(strings.ReplaceAll(p, "/", `\`), `\`) }
	return strings.EqualFold(norm(a), norm(b))
}
