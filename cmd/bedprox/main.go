package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/haveachin/bedprox"
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
	cpnChan := make(chan bedprox.ProcessingConn)
	srvChan := make(chan bedprox.ProcessingConn)
	poolChan := make(chan bedprox.ProcessedConn)

	logger.Info("starting")

	startGateways(cpnChan)
	startCPNs(cpnChan, srvChan)
	go func() {
		if err := startServers(srvChan, poolChan); err != nil {
			logger.Error(err, "Failed to start servers")
		}
	}()
	startConnPool(poolChan)

	logger.Info("ready")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func startGateways(cpnChan chan<- bedprox.ProcessingConn) {
	gateways, err := loadGateways()
	if err != nil {
		logger.Error(err, "loading gateways")
		return
	}

	for _, gw := range gateways {
		gw.Log = logger
		go gw.Start(cpnChan)
	}
}

func startCPNs(cpnChan <-chan bedprox.ProcessingConn, srvChan chan<- bedprox.ProcessingConn) {
	cpns, err := loadCPNs()
	if err != nil {
		logger.Error(err, "loading conn processors")
		return
	}

	for _, cpn := range cpns {
		cpn.Log = logger
		go cpn.Start(cpnChan, srvChan)
	}
}

func startServers(srvChan <-chan bedprox.ProcessingConn, poolChan chan<- bedprox.ProcessedConn) error {
	servers, err := loadServers()
	if err != nil {
		return err
	}

	for _, srv := range servers {
		srv.Log = logger
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

func startConnPool(poolChan <-chan bedprox.ProcessedConn) {
	pool := bedprox.ConnPool{
		Log: logger,
	}
	go pool.Start(poolChan)
}
