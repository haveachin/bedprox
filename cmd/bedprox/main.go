package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/haveachin/bedprox"
	"github.com/haveachin/bedprox/bedrock"
	"go.uber.org/zap"
)

const configPathEnv = "BEDPROX_CONFIG_PATH"

var configPath = "config.yml"

func envString(name string, value string) string {
	envString := os.Getenv(name)
	if envString == "" {
		return value
	}

	return envString
}

var logger logr.Logger

func init() {
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to init logger; err: %s", err)
	}
	logger = zapr.NewLogger(zapLog)
}

func main() {
	cpnChan := make(chan net.Conn)
	srvChan := make(chan bedprox.ProcessedConn)
	poolChan := make(chan bedprox.ConnTunnel)

	logger.Info("starting system")

	startGateways(cpnChan)
	startCPNs(cpnChan, srvChan)
	go func() {
		if err := startServers(srvChan, poolChan); err != nil {
			logger.Error(err, "failed to start servers")
		}
	}()
	startConnPool(poolChan)

	logger.Info("system ready")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func startGateways(cpnChan chan<- net.Conn) {
	gateways, err := loadGateways()
	if err != nil {
		logger.Error(err, "loading gateways")
		return
	}

	for _, gw := range gateways {
		gw.SetLogger(logger)
		go gw.ListenAndServe(cpnChan)
	}
}

func startCPNs(cpnChan <-chan net.Conn, srvChan chan<- bedprox.ProcessedConn) {
	cpns, err := loadCPNs()
	if err != nil {
		logger.Error(err, "loading conn processors")
		return
	}

	for _, cpn := range cpns {
		cpn.Log = logger
		cpn.ConnProcessor = &bedrock.ConnProcessor{
			Log: logger,
		}
		go cpn.Start(cpnChan, srvChan)
	}
}

func startServers(srvChan <-chan bedprox.ProcessedConn, poolChan chan<- bedprox.ConnTunnel) error {
	servers, err := loadServers()
	if err != nil {
		return err
	}

	for _, srv := range servers {
		srv.SetLogger(logger)
	}

	srvGw := bedprox.ServerGateway{
		Servers: servers,
		Log:     logger,
	}

	if err := srvGw.Start(srvChan, poolChan); err != nil {
		return err
	}

	return nil
}

func startConnPool(poolChan <-chan bedprox.ConnTunnel) {
	pool := bedprox.ConnPool{
		Log: logger,
	}
	go pool.Start(poolChan)
}
