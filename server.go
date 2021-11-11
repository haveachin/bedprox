package bedprox

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/haveachin/bedprox/webhook"
)

type Server interface {
	ID() string
	Domains() []string
	WebhookIDs() []string
	ProcessConn(c Conn) (ConnTunnel, error)
}

type ServerGateway struct {
	Servers  []Server
	Webhooks []webhook.Webhook
	Log      logr.Logger

	// Domain mapped to server
	srvs map[string]Server
	// Server ID mapped to webhooks
	srvWhks map[string][]webhook.Webhook
}

func (sg *ServerGateway) indexServers() error {
	sg.srvs = map[string]Server{}
	for _, server := range sg.Servers {
		for _, host := range server.Domains() {
			hostLower := strings.ToLower(host)
			if _, exits := sg.srvs[hostLower]; exits {
				return fmt.Errorf("duplicate server domain %q", hostLower)
			}
			sg.srvs[hostLower] = server
		}
	}
	return nil
}

// indexWebhooks indexes the webhooks that servers use.
// This creates a map
func (sg *ServerGateway) indexWebhooks() error {
	whks := map[string]webhook.Webhook{}
	for _, w := range sg.Webhooks {
		whks[w.ID] = w
	}

	sg.srvWhks = map[string][]webhook.Webhook{}
	for _, s := range sg.Servers {
		ww := make([]webhook.Webhook, len(s.WebhookIDs()))
		for n, id := range s.WebhookIDs() {
			w, ok := whks[id]
			if !ok {
				return fmt.Errorf("no webhook with id %q", id)
			}
			ww[n] = w
		}
		sg.srvWhks[s.ID()] = ww
	}
	return nil
}

func (sg ServerGateway) Start(srvChan <-chan ProcessedConn, poolChan chan<- ConnTunnel) error {
	if err := sg.indexServers(); err != nil {
		return err
	}

	if err := sg.indexWebhooks(); err != nil {
		return err
	}

	for {
		pc, ok := <-srvChan
		if !ok {
			break
		}

		hostLower := strings.ToLower(pc.ServerAddr())
		srv, ok := sg.srvs[hostLower]
		if !ok {
			sg.Log.Info("invlaid server host",
				"serverId", hostLower,
				"remoteAddress", pc.RemoteAddr(),
			)
			continue
		}

		sg.Log.Info("connecting client",
			"serverId", hostLower,
			"remoteAddress", pc.RemoteAddr(),
		)
		ct, err := srv.ProcessConn(pc)
		if err != nil {
			ct.Close()
			continue
		}
		poolChan <- ct
	}

	return nil
}
