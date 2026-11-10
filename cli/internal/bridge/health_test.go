package bridge

import (
	"errors"
	"reflect"
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
	type invocation struct {
		name string
		args []string
	}
	var commands []invocation
	run := func(_ time.Duration, name string, args ...string) (string, string, error) {
		commands = append(commands, invocation{name: name, args: append([]string(nil), args...)})
		if strings.Contains(strings.Join(args, " "), "/health/ready") {
			return `{"status":"ok"}`, "", nil
		}
		return `{"status":"pass"}`, "", nil
	}

	if got := ProbeHealth(run, testDescriptor("office")); !got.DestinationReady {
		t.Fatalf("ProbeHealth() = %+v", got)
	}
	want := []invocation{
		{name: "curl", args: []string{
			"--noproxy", "*", "--fail", "--silent", "--show-error", "--max-time", "5",
			"http://127.0.0.1:9081/health/ready",
		}},
		{name: "curl", args: []string{
			"--proxy", "socks5h://127.0.0.1:1081", "--noproxy", "",
			"--fail", "--silent", "--show-error", "--max-time", "15",
			"https://gitea.kubelab.live/api/healthz",
		}},
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("ProbeHealth commands = %#v; want %#v", commands, want)
	}
}
