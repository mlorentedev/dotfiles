package sshtarget

import (
	"bytes"
	"fmt"
	"strings"
)

// tailscaledLabel is the launchd job of the open-source tailscaled, the one
// variant of Tailscale on macOS that runs before anyone logs in. The App
// Store and Standalone apps run in the user's session.
const tailscaledLabel = "system/com.tailscale.tailscaled"

// MeshAtBootFor reads, without root, whether the mesh comes back after a
// reboot nobody attends, and returns the reason when it does not, "" when it
// does. It reports and never converges: on macOS the owner keeps the
// Standalone app and FileVault on (#2356, 2026-10-13), so a reboot is
// attended by decision, and the report keeps that decision visible. Nil on an
// OS the target does not read yet.
func MeshAtBootFor(goos string, run Runner) func() string {
	switch goos {
	case "darwin":
		return func() string {
			if _, err := run("launchctl", "print", tailscaledLabel); err == nil {
				return ""
			}
			return "tailscale runs as the app, not the tailscaled system daemon: after a reboot the node rejoins the mesh only once someone logs in (and unlocks FileVault)"
		}
	case "linux":
		return func() string {
			out, err := run("systemctl", "is-enabled", "tailscaled")
			state := strings.TrimSpace(string(bytes.SplitN(out, []byte("\n"), 2)[0]))
			if err == nil && state == "enabled" {
				return ""
			}
			if state == "" || strings.ContainsAny(state, " :") {
				state = "not installed" // systemctl's own error, not a unit state
			}
			return fmt.Sprintf("tailscaled is %s under systemd: after a reboot the node stays off the mesh until someone starts it (`sudo systemctl enable --now tailscaled`)", state)
		}
	}
	return nil
}
