package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
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
	logger.Info("loading proxy")

	p, err := loadProxy()
	if err != nil {
		logger.Error(err, "failed to load proxy")
		return
	}

	logger.Info("starting proxy")

	go func() {
		if err := p.start(logger); err != nil {
			logger.Error(err, "failed to start the proxy")
			os.Exit(1)
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
