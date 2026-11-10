// Package bridge declares the shared, non-secret routing contract for the
// role-gated ts-bridge client.
package bridge

const (
	GiteaHost            = "gitea.kubelab.live"
	SocksAddress         = "127.0.0.1:1081"
	SocksURL             = "socks5h://" + SocksAddress
	HealthAddress        = "127.0.0.1:9081"
	ListenerHealthURL    = "http://" + HealthAddress + "/health/ready"
	DestinationHealthURL = "https://" + GiteaHost + "/api/healthz"
	GitConfigBlock       = `[http "https://` + GiteaHost + `"]
	proxy = ` + SocksURL + `
`
)
