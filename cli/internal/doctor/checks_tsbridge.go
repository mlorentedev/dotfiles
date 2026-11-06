package doctor

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

const (
	tsBridgeHealthURL = "http://127.0.0.1:9081/health/ready"
	tsBridgeSocks     = "socks5h://127.0.0.1:1081"
	tsBridgeGiteaURL  = "https://gitea.kubelab.live/api/healthz"
)

func checkTsBridge(sys *System, cfg *Config, rep *Report) {
	rep.Section("ts-bridge client")
	roles, err := envpkg.LoadMachineRoles(envpkg.MachinePath(sys.home()))
	if err != nil {
		rep.Fail(err.Error())
		return
	}
	if !slices.Contains(roles, envpkg.RoleTSBridgeClient) {
		rep.Skip("this machine does not declare the ts-bridge-client role")
		return
	}
	if !sys.has("ts-bridge") {
		rep.Fail("ts-bridge not installed (run: dotf tools install ts-bridge)")
		return
	}
	version := semverOf(sys, "ts-bridge")
	if version == "" || version == "unknown" {
		rep.Fail("ts-bridge runs but reports no semver")
		return
	}
	matchPinFloorFrom(rep, "ts-bridge", version, catalogPin(sys, cfg, "ts-bridge"), "packages.json")
	if !sys.has("curl") || sys.CommandOutputBounded == nil {
		rep.Fail("curl is unavailable, so bridge and destination health cannot be proved")
		return
	}
	if !tsBridgeListenerReady(sys) {
		rep.Warn("bridge listener not ready at " + tsBridgeHealthURL + " (start the configured ts-bridge profile)")
		return
	}
	rep.Pass("bridge listener ready")
	if !tsBridgeDestinationReady(sys) {
		rep.Fail("Gitea destination through ts-bridge failed with verified TLS")
		return
	}
	rep.Pass("Gitea destination reachable with verified TLS")
}

func tsBridgeListenerReady(sys *System) bool {
	body, err := tsBridgeCurl(sys, 7*time.Second,
		"--fail", "--silent", "--show-error", "--max-time", "5", tsBridgeHealthURL)
	return err == nil && jsonStatus(body) == "ok"
}

func tsBridgeDestinationReady(sys *System) bool {
	body, err := tsBridgeCurl(sys, 20*time.Second,
		"--proxy", tsBridgeSocks, "--noproxy", "",
		"--fail", "--silent", "--show-error", "--max-time", "15", tsBridgeGiteaURL)
	return err == nil && jsonStatus(body) == "pass"
}

func tsBridgeCurl(sys *System, timeout time.Duration, args ...string) (string, error) {
	stdout, _, err := sys.CommandOutputBounded(timeout, "curl", args...)
	if err != nil {
		return "", fmt.Errorf("curl: %w", err)
	}
	return stdout, nil
}

func jsonStatus(body string) string {
	var doc struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		return ""
	}
	return strings.ToLower(doc.Status)
}
