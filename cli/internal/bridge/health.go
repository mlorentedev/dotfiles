package bridge

import (
	"encoding/json"
	"strings"
	"time"
)

// CommandRunner runs one bounded command and keeps stdout and stderr separate.
type CommandRunner func(timeout time.Duration, name string, args ...string) (stdout, stderr string, err error)

// Health is the two network layers the bridge must prove.
type Health struct {
	ListenerReady    bool
	DestinationReady bool
}

// ProbeHealth checks listener readiness, then the TLS-valid private destination.
func ProbeHealth(run CommandRunner, launch LaunchDescriptor) Health {
	if run == nil {
		return Health{}
	}
	stdout, _, err := run(7*time.Second, "curl",
		"--noproxy", "*", "--fail", "--silent", "--show-error", "--max-time", "5",
		"http://"+launch.HealthAddr+"/health/ready")
	if err != nil || healthStatus(stdout) != "ok" {
		return Health{}
	}
	health := Health{ListenerReady: true}
	stdout, _, err = run(20*time.Second, "curl",
		"--proxy", "socks5h://"+launch.Socks5, "--noproxy", "",
		"--fail", "--silent", "--show-error", "--max-time", "15",
		DestinationHealthURL)
	health.DestinationReady = err == nil && healthStatus(stdout) == "pass"
	return health
}

func healthStatus(body string) string {
	var doc struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		return ""
	}
	return strings.ToLower(doc.Status)
}
