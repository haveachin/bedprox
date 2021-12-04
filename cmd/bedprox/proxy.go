package main

import (
	"net"

	"github.com/go-logr/logr"
	"github.com/haveachin/bedprox"
	"github.com/haveachin/bedprox/bedrock"
)

type proxy struct {
	gateways      []bedprox.Gateway
	cpns          []bedprox.CPN
	serverGateway bedprox.ServerGateway
	connPool      bedprox.ConnPool
}

func loadProxy() (proxy, error) {
	gateways, err := loadGateways()
	if err != nil {
		return proxy{}, err
	}

	gwIDsIDs := map[string][]string{}
	for _, gw := range gateways {
		gwIDsIDs[gw.GetID()] = gw.GetServerIDs()
	}

	cpns, err := loadCPNs()
	if err != nil {
		return proxy{}, err
	}

	servers, err := loadServers()
	if err != nil {
		return proxy{}, err
	}

	return proxy{
		gateways: gateways,
		cpns:     cpns,
		serverGateway: bedprox.ServerGateway{
			GatewayIDServerIDs: gwIDsIDs,
			Servers:            servers,
			Log:                logger,
		},
		connPool: bedprox.ConnPool{
			Log: logger,
		},
	}, nil
}

func (p proxy) start(log logr.Logger) error {
	cpnChan := make(chan net.Conn)
	srvChan := make(chan bedprox.ProcessedConn)
	poolChan := make(chan bedprox.ConnTunnel)

	for _, gw := range p.gateways {
		gw.SetLogger(log)
		go gw.ListenAndServe(cpnChan)
	}

	for _, cpn := range p.cpns {
		cpn.Log = log
		cpn.ConnProcessor = &bedrock.ConnProcessor{
			Log: log,
		}
		go cpn.Start(cpnChan, srvChan)
	}

	go p.connPool.Start(poolChan)

	for _, srv := range p.serverGateway.Servers {
		srv.SetLogger(log)
	}

	if err := p.serverGateway.Start(srvChan, poolChan); err != nil {
		return err
	}

	return nil
}
