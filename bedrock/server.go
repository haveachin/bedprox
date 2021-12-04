package bedrock

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/haveachin/bedprox"
	"github.com/haveachin/bedprox/bedrock/protocol"
	"github.com/haveachin/bedprox/webhook"
	"github.com/sandertv/go-raknet"
)

type Server struct {
	ID                string
	Domains           []string
	Dialer            raknet.Dialer
	DialTimeout       time.Duration
	Address           string
	SendProxyProtocol bool
	DisconnectMessage string
	WebhookIDs        []string
	Log               logr.Logger
}

func (s Server) GetID() string {
	return s.ID
}

func (s Server) GetDomains() []string {
	return s.Domains
}

func (s Server) GetWebhookIDs() []string {
	return s.WebhookIDs
}

func (s *Server) SetLogger(log logr.Logger) {
	s.Log = log
}

func (s Server) Dial() (*raknet.Conn, error) {
	c, err := s.Dialer.DialTimeout(s.Address, s.DialTimeout)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (s Server) replaceTemplates(c ProcessedConn, msg string) string {
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

func (s Server) handleOffline(c ProcessedConn) error {
	msg := s.replaceTemplates(c, s.DisconnectMessage)

	pk := protocol.Disconnect{
		HideDisconnectionScreen: false,
		Message:                 msg,
	}

	b, err := protocol.MarshalPacket(&pk)
	if err != nil {
		return err
	}

	if _, err := c.Write(b); err != nil {
		return err
	}

	return nil
}

func (s Server) ProcessConn(c net.Conn, webhooks []webhook.Webhook) (bedprox.ConnTunnel, error) {
	pc := c.(*ProcessedConn)
	rc, err := s.Dial()
	if err != nil {
		if err := s.handleOffline(*pc); err != nil {
			s.Log.Error(err, "failed to handle offline")
			return bedprox.ConnTunnel{}, err
		}
		s.Log.Info("disconnected client")
		return bedprox.ConnTunnel{}, err
	}

	if _, err := rc.Write(pc.readBytes); err != nil {
		s.Log.Error(err, "failed to write to server")
		rc.Close()
		return bedprox.ConnTunnel{}, err
	}

	return bedprox.ConnTunnel{
		Conn:       c,
		RemoteConn: rc,
	}, nil
}
