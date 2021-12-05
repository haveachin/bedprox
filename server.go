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
	GatewayIDServerIDs map[string][]string
	Servers            []Server
	Webhooks           []webhook.Webhook
	Log                logr.Logger

	// "GatewayID@Domain" mapped to server
	srvs map[string]Server
	// Server ID mapped to webhooks
	srvWhks map[string][]webhook.Webhook
}

func (sg *ServerGateway) indexServers() error {
	srvs := map[string]Server{}
	for _, srv := range sg.Servers {
		srvs[srv.GetID()] = srv
	}

	sg.srvs = map[string]Server{}
	for gID, sIDs := range sg.GatewayIDServerIDs {
		for _, sID := range sIDs {
			srv, ok := srvs[sID]
			if !ok {
				return fmt.Errorf("server with ID %q doesn't exist", sID)
			}

			for _, domain := range srv.GetDomains() {
				lowerDomain := strings.ToLower(domain)
				sgID := fmt.Sprintf("%s@%s", gID, lowerDomain)
				if _, exits := sg.srvs[sgID]; exits {
					return fmt.Errorf("duplicate server gateway ID %q", sgID)
				}
				sg.srvs[sgID] = srv
			}
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
	for _, srv := range sg.Servers {
		ww := make([]webhook.Webhook, len(srv.GetWebhookIDs()))
		for n, id := range srv.GetWebhookIDs() {
			w, ok := whks[id]
			if !ok {
				return fmt.Errorf("webhook with ID %q doesn't exist", id)
			}
			ww[n] = w
		}
		sg.srvWhks[srv.GetID()] = ww
	}
	return nil
}

func (sg ServerGateway) executeTemplate(msg string, pc ProcessedConn) string {
	tmpls := map[string]string{
		"username":      pc.Username(),
		"now":           time.Now().Format(time.RFC822),
		"remoteAddress": pc.RemoteAddr().String(),
		"localAddress":  pc.LocalAddr().String(),
		"serverAddress": pc.ServerAddr(),
		"gatewayID":     pc.GatewayID(),
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

		srvAddrLower := strings.ToLower(pc.ServerAddr())
		sgID := fmt.Sprintf("%s@%s", pc.GatewayID(), srvAddrLower)
		srv, ok := sg.srvs[sgID]
		if !ok {
			sg.Log.Info("invalid server",
				"serverAddress", pc.ServerAddr(),
				"remoteAddress", pc.RemoteAddr(),
			)
			msg := pc.ServerNotFoundMessage()
			msg = sg.executeTemplate(msg, pc)
			_ = pc.Disconnect(msg)
			continue
		}

		sg.Log.Info("connecting client",
			"serverId", sgID,
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
