package update

import (
	"strings"
	"testing"
)

// The trigger names the installed dotf in each OS's own spelling, whatever OS
// renders it, and only launchd gets an environment and a log.
func TestTrigger_NamesTheInstalledDotfPerOS(t *testing.T) {
	tests := []struct {
		goos, home, program string
		hasEnv              bool
	}{
		{"darwin", "/Users/me", "/Users/me/.local/bin/dotf", true},
		{"linux", "/home/me", "/home/me/.local/bin/dotf", false},
		{"windows", `C:\Users\me`, `C:\Users\me\.local\bin\dotf.exe`, false},
	}
	for _, tt := range tests {
		job := Trigger(tt.goos, tt.home, "")
		if job.Program != tt.program || strings.Join(job.Args, " ") != "update" {
			t.Errorf("%s: runs %q %q, want %q update", tt.goos, job.Program, job.Args, tt.program)
		}
		if (len(job.Env) > 0) != tt.hasEnv || (job.Log != "") != tt.hasEnv {
			t.Errorf("%s: env %v log %q", tt.goos, job.Env, job.Log)
		}
		if job.Daily == nil || job.RunAtLoad {
			t.Errorf("%s: a daily job that never runs at load, got daily=%v runAtLoad=%v", tt.goos, job.Daily, job.RunAtLoad)
		}
	}
	if p := Trigger("darwin", "/Users/me", "").Env[0].Value; !strings.HasPrefix(p, "/Users/me/.local/bin:") || !strings.Contains(p, "/opt/homebrew/bin") {
		t.Errorf("launchd PATH = %q: it must reach dotf and Homebrew", p)
	}
}

// The job reads the machine.json the installing run read: a scheduler does not
// inherit the shell's XDG_CONFIG_HOME, and a job that finds no role removes
// itself.
func TestTrigger_CarriesTheConfigHomeOfTheInstallingRun(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		env := Trigger(goos, "/home/me", "/srv/cfg").Env
		if last := env[len(env)-1]; last.Name != "XDG_CONFIG_HOME" || last.Value != "/srv/cfg" {
			t.Errorf("%s: env %v, want XDG_CONFIG_HOME=/srv/cfg", goos, env)
		}
	}
}
