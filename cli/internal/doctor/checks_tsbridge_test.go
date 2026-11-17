package doctor

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

func TestCheckTsBridgeWithoutRoleSkipsBeforeProbes(t *testing.T) {
	sys, cfg := tsBridgeDoctorFixture(t, false, nil)
	sys.CommandOutputBounded = func(time.Duration, string, ...string) (string, string, error) {
		t.Fatal("a non-client machine must not probe bridge endpoints")
		return "", "", nil
	}
	var buf bytes.Buffer

	checkTsBridge(sys, cfg, capture(&buf))

	if out := buf.String(); !strings.Contains(out, "[SKIP] this machine does not declare the ts-bridge-client role") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestCheckTsBridgeMissingBinaryFails(t *testing.T) {
	sys, cfg := tsBridgeDoctorFixture(t, true, nil)
	var buf bytes.Buffer

	checkTsBridge(sys, cfg, capture(&buf))

	if out := buf.String(); !strings.Contains(out, "[FAIL] ts-bridge not installed") ||
		!strings.Contains(out, "dotf tools install ts-bridge") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestCheckTsBridgeListenerDownIsExplicit(t *testing.T) {
	sys, cfg := tsBridgeDoctorFixture(t, true, []string{"ts-bridge", "curl"})
	sys.CommandOutputBounded = func(_ time.Duration, name string, args ...string) (string, string, error) {
		if strings.Contains(strings.Join(args, " "), "gitea.kubelab.live") {
			t.Fatal("destination probe ran although listener readiness failed")
		}
		return "", "connection refused", errors.New("exit status 7")
	}
	var buf bytes.Buffer

	checkTsBridge(sys, cfg, capture(&buf))

	if out := buf.String(); !strings.Contains(out, "[WARN] bridge listener not ready") ||
		strings.Contains(out, "[ OK ] Gitea destination") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestCheckTsBridgeReportsBinaryDrift(t *testing.T) {
	sys, cfg := tsBridgeDoctorFixture(t, true, []string{"ts-bridge", "curl"})
	sys.VersionOutput = func(name string, args ...string) ([]byte, error) {
		if name == "ts-bridge" {
			return []byte("ts-bridge v1.19.0"), nil
		}
		return nil, errors.New("unexpected command")
	}
	sys.CommandOutputBounded = func(time.Duration, string, ...string) (string, string, error) {
		return "", "", errors.New("listener down")
	}
	var buf bytes.Buffer

	checkTsBridge(sys, cfg, capture(&buf))

	if out := buf.String(); !strings.Contains(out, "[WARN] ts-bridge version drift: installed=1.19.0 pinned=1.20.0 (packages.json)") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestCheckTsBridgeReadyButDestinationFailedIsFailure(t *testing.T) {
	sys, cfg := tsBridgeDoctorFixture(t, true, []string{"ts-bridge", "curl"})
	sys.CommandOutputBounded = func(_ time.Duration, _ string, args ...string) (string, string, error) {
		if strings.Contains(strings.Join(args, " "), "9081/health/ready") {
			return `{"status":"ok"}`, "", nil
		}
		return "", "SOCKS5 target timeout", errors.New("exit status 97")
	}
	var buf bytes.Buffer

	checkTsBridge(sys, cfg, capture(&buf))

	out := buf.String()
	if !strings.Contains(out, "[ OK ] bridge listener ready") ||
		!strings.Contains(out, "[FAIL] Gitea destination through ts-bridge failed") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestCheckTsBridgeProvesTlsDestinationWithoutInsecureOrCredentials(t *testing.T) {
	sys, cfg := tsBridgeDoctorFixture(t, true, []string{"ts-bridge", "curl"})
	var commands []string
	sys.CommandOutputBounded = func(_ time.Duration, name string, args ...string) (string, string, error) {
		cmd := name + " " + strings.Join(args, " ")
		commands = append(commands, cmd)
		if strings.Contains(cmd, "9081/health/ready") {
			return `{"status":"ok","profile_secret":"LISTENER-SENTINEL"}`, "", nil
		}
		return `{"status":"pass","description":"Gitea","token":"DESTINATION-SENTINEL"}`, "", nil
	}
	var buf bytes.Buffer

	checkTsBridge(sys, cfg, capture(&buf))

	out := buf.String()
	if !strings.Contains(out, "[ OK ] bridge listener ready") ||
		!strings.Contains(out, "[ OK ] Gitea destination reachable with verified TLS") {
		t.Fatalf("output:\n%s", out)
	}
	joined := strings.Join(commands, "\n")
	for _, want := range []string{"--proxy socks5h://127.0.0.1:1081", "https://gitea.kubelab.live/api/healthz"} {
		if !strings.Contains(joined, want) {
			t.Errorf("commands missing %q:\n%s", want, joined)
		}
	}
	for _, forbidden := range []string{"--insecure", "-k", "authkey", "credential", "listener-sentinel", "destination-sentinel"} {
		if strings.Contains(strings.ToLower(joined), forbidden) || strings.Contains(strings.ToLower(out), forbidden) {
			t.Errorf("secret/TLS weakening token %q leaked:\n%s\n%s", forbidden, joined, out)
		}
	}
}

func tsBridgeDoctorFixture(t *testing.T, role bool, onPath []string) (*System, *Config) {
	t.Helper()
	repo, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	writeFile(t, filepath.Join(repo, "packages.json"), `{"tools":[{
		"name":"ts-bridge","version":"1.20.0","profile":"full","roles":["ts-bridge-client"],
		"source":{"type":"github-archive","repo":"o/r","asset":{"windows":"tool.zip"},
		"checksums":"checksums.txt","member":{"windows":"tool.exe"}}
	}]}`)
	if role {
		if _, err := envpkg.SetMachineRole(envpkg.MachinePath(home), envpkg.RoleTSBridgeClient, true); err != nil {
			t.Fatal(err)
		}
	}
	sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, onPath,
		map[string]string{"ts-bridge --version": "ts-bridge v1.20.0"})
	sys.GOOS = "windows"
	return sys, &Config{DotfilesDir: repo}
}

func TestCheckTsBridgeMalformedHealthBodiesAreNotHealthy(t *testing.T) {
	sys, cfg := tsBridgeDoctorFixture(t, true, []string{"ts-bridge", "curl"})
	sys.CommandOutputBounded = func(_ time.Duration, _ string, args ...string) (string, string, error) {
		if strings.Contains(strings.Join(args, " "), "9081/health/ready") {
			return `{"status":"starting"}`, "", nil
		}
		return `{"status":"fail"}`, "", nil
	}
	var buf bytes.Buffer
	checkTsBridge(sys, cfg, capture(&buf))
	if out := buf.String(); !strings.Contains(out, "[WARN] bridge listener not ready") {
		t.Fatalf("output:\n%s", out)
	}
}
