package bedprox

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/haveachin/bedprox/protocol"
	"github.com/haveachin/bedprox/webhook"
	"github.com/sandertv/go-raknet"
)

type Server struct {
	ID                string
	Domains           []string
	Dialer            raknet.Dialer
	Address           string
	SendProxyProtocol bool
	DisconnectMessage string
	WebhookIDs        []string
	Log               logr.Logger
}

func (s Server) Dial() (*raknet.Conn, error) {
	c, err := s.Dialer.Dial(s.Address)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (s Server) replaceTemplates(c ProcessingConn, msg string) string {
	tmpls := map[string]string{
		"username":      c.username,
		"now":           time.Now().Format(time.RFC822),
		"remoteAddress": c.RemoteAddr().String(),
		"localAddress":  c.LocalAddr().String(),
		"domain":        c.srvHost,
		"serverAddress": s.Address,
	}

	for k, v := range tmpls {
		msg = strings.Replace(msg, fmt.Sprintf("{{%s}}", k), v, -1)
	}

	return msg
}

func (s Server) handleOffline(c ProcessingConn) error {
	msg := s.replaceTemplates(c, s.DisconnectMessage)

	pk := protocol.Disconnect{
		HideDisconnectionScreen: false,
		Message:                 msg,
	}

	buf := protocol.BufferPool.Get().(*bytes.Buffer)
	defer func() {
		// Reset the buffer so we can return it to the buffer pool safely.
		buf.Reset()
		protocol.BufferPool.Put(buf)
	}()

	pk.Marshal(protocol.NewWriter(buf))
	encoder := protocol.NewEncoder(buf)
	b := make([]byte, buf.Len())
	if err := encoder.Encode(b); err != nil {
		return err
	}
	if _, err := c.Write(b); err != nil {
		return err
	}

	return nil
}

func (s Server) ProcessConnection(c ProcessingConn) (ProcessedConn, error) {
	rc, err := s.Dial()
	if err != nil {
		log.Println("no server conn", err)
		if err := s.handleOffline(c); err != nil {
			return ProcessedConn{}, err
		}
		return ProcessedConn{}, err
	}

	if _, err := rc.Write(c.readBytes); err != nil {
		log.Println("woops")
		rc.Close()
		return ProcessedConn{}, err
	}

	return ProcessedConn{
		ProcessingConn: c,
		ServerConn:     rc,
		ServerID:       s.ID,
	}, nil
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
		for _, host := range server.Domains {
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
		ww := make([]webhook.Webhook, len(s.WebhookIDs))
		for n, id := range s.WebhookIDs {
			w, ok := whks[id]
			if !ok {
				return fmt.Errorf("no webhook with id %q", id)
			}
			ww[n] = w
		}
		sg.srvWhks[s.ID] = ww
	}
	return nil
}

func (sg ServerGateway) Start(srvChan <-chan ProcessingConn, poolChan chan<- ProcessedConn) error {
	if err := sg.indexServers(); err != nil {
		return err
	}

	if err := sg.indexWebhooks(); err != nil {
		return err
	}

	for {
		c, ok := <-srvChan
		if !ok {
			break
		}

		hostLower := strings.ToLower(c.srvHost)
		srv, ok := sg.srvs[hostLower]
		if !ok {
			sg.Log.Info("invlaid server host",
				"serverId", hostLower,
				"remoteAddress", c.RemoteAddr(),
			)
			continue
		}

		sg.Log.Info("connecting client",
			"serverId", hostLower,
			"remoteAddress", c.RemoteAddr(),
		)
		pc, err := srv.ProcessConnection(c)
		if err != nil {
			c.Close()
			continue
		}
		poolChan <- pc
	}

	return nil
}
