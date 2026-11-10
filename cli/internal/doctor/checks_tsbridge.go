package doctor

import (
	"slices"

	"github.com/mlorentedev/dotfiles/cli/internal/bridge"
	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

const tsBridgeHealthURL = bridge.ListenerHealthURL

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
	health := bridge.ProbeHealth(sys.CommandOutputBounded, bridge.LaunchDescriptor{
		Profile:    "doctor",
		Socks5:     bridge.SocksAddress,
		HealthAddr: bridge.HealthAddress,
	})
	if !health.ListenerReady {
		rep.Warn("bridge listener not ready at " + tsBridgeHealthURL + " (start the configured ts-bridge profile)")
		return
	}
	rep.Pass("bridge listener ready")
	if !health.DestinationReady {
		rep.Fail("Gitea destination through ts-bridge failed with verified TLS")
		return
	}
	rep.Pass("Gitea destination reachable with verified TLS")
}
