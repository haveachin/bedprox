package bedprox

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/haveachin/bedprox/webhook"
)

type Server interface {
	GetID() string
	GetDomains() []string
	GetWebhookIDs() []string
	ProcessConn(c net.Conn, webhooks []webhook.Webhook) (ConnTunnel, error)
	SetLogger(log logr.Logger)
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
		for _, host := range server.GetDomains() {
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
func (sg *ServerGateway) indexWebhooks() error {
	whks := map[string]webhook.Webhook{}
	for _, w := range sg.Webhooks {
		whks[w.ID] = w
	}

	sg.srvWhks = map[string][]webhook.Webhook{}
	for _, s := range sg.Servers {
		ww := make([]webhook.Webhook, len(s.GetWebhookIDs()))
		for n, id := range s.GetWebhookIDs() {
			w, ok := whks[id]
			if !ok {
				return fmt.Errorf("no webhook with id %q", id)
			}
			ww[n] = w
		}
		sg.srvWhks[s.GetID()] = ww
	}
	return nil
}

func (sg ServerGateway) executeTemplate(msg string, c ProcessedConn) string {
	tmpls := map[string]string{
		"username":      c.Username(),
		"now":           time.Now().Format(time.RFC822),
		"remoteAddress": c.RemoteAddr().String(),
		"localAddress":  c.LocalAddr().String(),
		"serverAddress": c.ServerAddr(),
	}

	for k, v := range tmpls {
		msg = strings.Replace(msg, fmt.Sprintf("{{%s}}", k), v, -1)
	}

	return msg
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
				"serverAddress", hostLower,
				"remoteAddress", pc.RemoteAddr(),
			)
			_ = pc.Disconnect("Server not found")
			continue
		}

		if !pc.CanJoinServerWithID(srv.GetID()) {
			sg.Log.Info("server not in gateway",
				"serverId", srv.GetID(),
				"remoteAddress", pc.RemoteAddr(),
			)
			_ = pc.Disconnect("This is not the server you are looking for")
			continue
		}

		sg.Log.Info("connecting client",
			"serverId", hostLower,
			"remoteAddress", pc.RemoteAddr(),
		)

		whks := sg.srvWhks[srv.GetID()]
		ct, err := srv.ProcessConn(pc, whks)
		if err != nil {
			ct.Close()
			continue
		}
		poolChan <- ct
	}

	return nil
}
