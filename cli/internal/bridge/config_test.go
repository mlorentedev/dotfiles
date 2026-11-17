package bridge

import (
	"strings"
	"testing"
)

func TestRouteDefaultsShareOneHostAndSocksEndpoint(t *testing.T) {
	if SocksURL != "socks5h://"+SocksAddress {
		t.Fatalf("SocksURL = %q, address = %q", SocksURL, SocksAddress)
	}
	for _, want := range []string{GiteaHost, SocksURL} {
		if !strings.Contains(GitConfigBlock, want) {
			t.Errorf("GitConfigBlock lacks %q:\n%s", want, GitConfigBlock)
		}
	}
	if !strings.Contains(DestinationHealthURL, GiteaHost) ||
		!strings.Contains(ListenerHealthURL, HealthAddress) {
		t.Fatalf("health URLs drifted: listener=%q destination=%q", ListenerHealthURL, DestinationHealthURL)
	}
}
