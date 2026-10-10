package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestCheckDockerCompose pins the one tool setup's check_dependencies named that
// doctor covered nowhere. Compose v2 ships as a `docker` CLI plugin, not a
// binary, so a PATH test alone answers wrongly on a current install — measured
// on msi 2026-09-02, where the v1 binary and plugin v2.39.1 are both present.
// Only darwin declares compose (packages.json, brew), so absence is a SKIP; a
// FAIL would red every box that never wanted it (the BUG-052 reasoning that put
// terraform in optionalTools).
func TestCheckDockerCompose(t *testing.T) {
	cases := []struct {
		name         string
		onPath       []string
		cmdOut       map[string]string
		wantFailures int
		wantSubstr   string
	}{
		{
			name:       "v2 plugin present → pass naming the version",
			onPath:     []string{"docker"},
			cmdOut:     map[string]string{"docker compose version": "Docker Compose version v2.39.1"},
			wantSubstr: "v2.39.1",
		},
		{
			// A box still on the standalone binary: compose works, so this is
			// not a failure, but the plugin is the supported form.
			name:       "only the v1 binary → pass, flagged legacy",
			onPath:     []string{"docker", "docker-compose"},
			wantSubstr: "legacy",
		},
		{
			// Homebrew's docker-compose is v2 but unregistered until
			// docker-config deploys: `docker compose` fails, so not a pass.
			name:       "v2 binary not registered as a plugin → warn naming the deploy",
			onPath:     []string{"docker", "docker-compose"},
			cmdOut:     map[string]string{"docker-compose version": "Docker Compose version v2.40.0"},
			wantSubstr: "dotf deploy docker-config",
		},
		{
			name:       "docker present, no compose either way → reported",
			onPath:     []string{"docker"},
			wantSubstr: "compose",
		},
		{
			// Without docker there is nothing for a plugin to hang off; the
			// core-tools section already reports docker itself.
			name:       "no docker → skip without duplicating the core-tools verdict",
			wantSubstr: "docker",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sys := newSys(nil, tc.onPath, tc.cmdOut)
			var buf bytes.Buffer
			rep := capture(&buf)
			checkDockerCompose(sys, rep)

			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}

// Docker without compose is a gap on every OS that provisions docker, and the
// remedy differs: darwin declares compose, so tools install fixes it. Linux has
// no compose row yet, and the plugin must come from docker's own source, since
// Ubuntu's docker-compose-v2 displaces Docker's docker-ce. A SKIP that called it
// "optional" hid a box where every `docker compose` failed.
func TestCheckDockerCompose_MissingIsAWarnNamingTheSource(t *testing.T) {
	for _, tc := range []struct {
		goos, want string
		warn       bool
	}{
		{"darwin", "dotf tools install", true},
		{"linux", "docker-compose-plugin", true},
		{"windows", "Docker Desktop", false},
	} {
		t.Run(tc.goos, func(t *testing.T) {
			sys := newSys(nil, []string{"docker"}, nil)
			sys.GOOS = tc.goos
			var buf bytes.Buffer
			rep := capture(&buf)
			checkDockerCompose(sys, rep)
			if got := rep.Warnings() == 1; got != tc.warn || rep.Failures() != 0 {
				t.Fatalf("warnings = %d, failures = %d; want a warn: %v\n%s", rep.Warnings(), rep.Failures(), tc.warn, buf.String())
			}
			if !strings.Contains(buf.String(), tc.want) {
				t.Fatalf("output missing %q\n%s", tc.want, buf.String())
			}
		})
	}
}

// TestCheckDockerEngine: a docker CLI with no engine behind it is the state the
// Mac lands in after `dotf tools install` until Colima runs, and every
// `docker compose` in kubelab fails there. It is a WARN with the start command
// for the OS, never a FAIL: the Windows gate fails on any unlisted [FAIL] line
// and its runner's engine is not ours to start.
func TestCheckDockerEngine(t *testing.T) {
	cases := []struct {
		name       string
		goos       string
		onPath     []string
		cmdOut     map[string]string
		wantWarn   int
		wantSubstr string
	}{
		{
			name:       "engine answers → pass naming the server version",
			goos:       "darwin",
			onPath:     []string{"docker"},
			cmdOut:     map[string]string{"docker info --format {{.ServerVersion}}": "28.5.1"},
			wantSubstr: "28.5.1",
		},
		{
			name:       "darwin, engine down → warn with the Colima service command",
			goos:       "darwin",
			onPath:     []string{"docker"},
			wantWarn:   1,
			wantSubstr: "dotf doctor --fix",
		},
		{
			name:       "linux, engine down → warn with the systemd unit",
			goos:       "linux",
			onPath:     []string{"docker"},
			wantWarn:   1,
			wantSubstr: "systemctl start docker",
		},
		{
			name:       "windows, engine down → warn naming Docker Desktop",
			goos:       "windows",
			onPath:     []string{"docker"},
			wantWarn:   1,
			wantSubstr: "Docker Desktop",
		},
		{
			name:       "no docker → skip",
			goos:       "darwin",
			wantSubstr: "not on PATH",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sys := newSys(nil, tc.onPath, tc.cmdOut)
			sys.GOOS = tc.goos
			var buf bytes.Buffer
			rep := capture(&buf)
			checkDockerEngine(sys, rep, false)

			if rep.Failures() != 0 {
				t.Fatalf("failures = %d, want 0\n%s", rep.Failures(), buf.String())
			}
			if rep.Warnings() != tc.wantWarn {
				t.Fatalf("warnings = %d, want %d\n%s", rep.Warnings(), tc.wantWarn, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}

// On Linux `docker info` also fails when the daemon runs and the socket refuses
// this user, the state apt's docker.io leaves until the user joins the docker
// group. Advising `systemctl start` there sends the owner after a daemon that is
// already up (pr-agent on #2237).
func TestCheckDockerEngine_SocketPermissionDenied(t *testing.T) {
	denied := "permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock"
	for _, tc := range []struct {
		name, goos, errOut, want, notWant string
	}{
		{"linux, socket refuses the user → the docker group", "linux", denied, "usermod -aG docker", "systemctl"},
		{"linux, daemon down → the systemd unit", "linux", "Cannot connect to the Docker daemon. Is the docker daemon running?", "systemctl start docker", "usermod"},
		{"darwin, denied → the Colima remedy, no Linux group", "darwin", denied, "dotf doctor --fix", "usermod"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sys := newSys(nil, []string{"docker"}, nil)
			sys.GOOS = tc.goos
			sys.CommandOutputBounded = func(time.Duration, string, ...string) (string, string, error) {
				return "", tc.errOut, errors.New("exit status 1")
			}
			var buf bytes.Buffer
			rep := capture(&buf)
			checkDockerEngine(sys, rep, false)

			if rep.Warnings() != 1 || rep.Failures() != 0 {
				t.Fatalf("want one WARN and no FAIL\n%s", buf.String())
			}
			if !strings.Contains(buf.String(), tc.want) || strings.Contains(buf.String(), tc.notWant) {
				t.Fatalf("want %q and not %q\n%s", tc.want, tc.notWant, buf.String())
			}
		})
	}
}

// With --fix on darwin doctor starts Colima itself, so the owner is handed no
// manual step (#2013 P5b). The fake engine answers only after the start ran.
func TestCheckDockerEngine_FixStartsColima(t *testing.T) {
	engineProbeInterval = 0
	t.Cleanup(func() { engineProbeInterval = 3 * time.Second })

	for _, tc := range []struct {
		name        string
		goos        string
		fix         bool
		onPath      []string
		wantStarted bool
		wantSubstr  string
	}{
		{"darwin --fix starts the service and passes", "darwin", true, []string{"docker", "brew", "colima"}, true, "engine reachable: server 28.5.1"},
		{"no --fix only advises", "darwin", false, []string{"docker", "brew", "colima"}, false, "dotf doctor --fix"},
		{"linux --fix leaves systemd alone", "linux", true, []string{"docker", "brew", "colima"}, false, "systemctl start docker"},
		// --fix has nothing to start without colima, so pointing back at it would
		// be circular: the remedy names the install that makes --fix work.
		{"colima absent: install it first", "darwin", true, []string{"docker", "brew"}, false, "colima is not installed (run: dotf tools install, then dotf doctor --fix)"},
		{"brew absent: install it first", "darwin", true, []string{"docker", "colima"}, false, "Homebrew is not installed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sys := newSys(nil, tc.onPath, nil)
			sys.GOOS = tc.goos
			started := false
			sys.CommandOutputBounded = func(_ time.Duration, name string, args ...string) (string, string, error) {
				switch strings.Join(append([]string{name}, args...), " ") {
				case "brew services start colima":
					started = true
					return "Successfully started `colima`", "", nil
				case "docker info --format {{.ServerVersion}}":
					if started {
						return "28.5.1", "", nil
					}
				}
				return "", "", errors.New("down")
			}
			var buf bytes.Buffer
			rep := capture(&buf)
			checkDockerEngine(sys, rep, tc.fix)

			if started != tc.wantStarted {
				t.Fatalf("started = %v, want %v\n%s", started, tc.wantStarted, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
			if rep.Failures() != 0 {
				t.Fatalf("an engine check never FAILs\n%s", buf.String())
			}
		})
	}
}
