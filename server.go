package bedprox

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-logr/logr"
)

type Server interface {
	GetID() string
	GetDomains() []string
	ProcessConn(c net.Conn) (ConnTunnel, error)
	SetLogger(log logr.Logger)
}

type ServerGateway struct {
	GatewayIDServerIDs map[string][]string
	Servers            []Server
	Log                logr.Logger
	Plugins            []Plugin

	// "GatewayID@Domain" mapped to server
	srvs map[string]Server
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

		if pc.IsJoining() {
			for _, p := range sg.Plugins {
				p.OnPlayerJoin()
			}
		}

		ct, err := srv.ProcessConn(pc)
		if err != nil {
			ct.Close()
			continue
		}
		poolChan <- ct
	}

	return nil
}
