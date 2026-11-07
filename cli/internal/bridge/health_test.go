package bridge

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProbeHealthSeparatesListenerAndDestination(t *testing.T) {
	tests := []struct {
		name        string
		listenerOut string
		listenerErr error
		destOut     string
		destErr     error
		want        Health
		wantCalls   int
	}{
		{"listener down", "", errors.New("refused"), "", nil, Health{}, 1},
		{"destination down", `{"status":"ok"}`, nil, "", errors.New("timeout"), Health{ListenerReady: true}, 2},
		{"healthy", `{"status":"ok","secret":"LISTENER"}`, nil, `{"status":"pass","token":"DEST"}`, nil,
			Health{ListenerReady: true, DestinationReady: true}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			run := func(_ time.Duration, _ string, args ...string) (string, string, error) {
				calls++
				if strings.Contains(strings.Join(args, " "), "/health/ready") {
					return tt.listenerOut, "", tt.listenerErr
				}
				return tt.destOut, "", tt.destErr
			}
			got := ProbeHealth(run, testDescriptor("office"))
			if got != tt.want || calls != tt.wantCalls {
				t.Fatalf("ProbeHealth() = %+v, calls %d; want %+v, calls %d", got, calls, tt.want, tt.wantCalls)
			}
		})
	}
}

func TestProbeHealthUsesTLSAndRemoteDNSWithoutWeakening(t *testing.T) {
	var commands []string
	run := func(_ time.Duration, name string, args ...string) (string, string, error) {
		command := name + " " + strings.Join(args, " ")
		commands = append(commands, command)
		if strings.Contains(command, "/health/ready") {
			return `{"status":"ok"}`, "", nil
		}
		return `{"status":"pass"}`, "", nil
	}

	if got := ProbeHealth(run, testDescriptor("office")); !got.DestinationReady {
		t.Fatalf("ProbeHealth() = %+v", got)
	}
	joined := strings.Join(commands, "\n")
	for _, want := range []string{"--proxy socks5h://127.0.0.1:1081", "https://gitea.kubelab.live/api/healthz"} {
		if !strings.Contains(joined, want) {
			t.Errorf("commands missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "--insecure") {
		t.Fatalf("probe weakened TLS:\n%s", joined)
	}
}
