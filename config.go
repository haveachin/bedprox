package bedprox

import "github.com/haveachin/bedprox/webhook"

type ProxyConfig interface {
	LoadGateways() ([]Gateway, error)
	LoadServers() ([]Server, error)
	LoadCPNs() ([]CPN, error)
	LoadWebhooks() ([]webhook.Webhook, error)
}
