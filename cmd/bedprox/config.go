package main

import (
	"bytes"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	_ "embed"

	"github.com/haveachin/bedprox"
	"github.com/haveachin/bedprox/webhook"
	"github.com/sandertv/go-raknet"
	"github.com/spf13/viper"
)

//go:embed config.yml
var defaultConfig []byte

func init() {
	configPath = envString(configPathEnv, configPath)

	viper.SetConfigFile(configPath)
	viper.ReadConfig(bytes.NewBuffer(defaultConfig))
	if err := viper.MergeInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(configPath, defaultConfig, 0644); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}
}

type pingStatusConfig struct {
	Edition         string `mapstructure:"edition"`
	ProtocolVersion int    `mapstructure:"protocol_version,omitempty"`
	VersionName     string `mapstructure:"version_name,omitempty"`
	PlayerCount     int    `mapstructure:"player_count,omitempty"`
	MaxPlayerCount  int    `mapstructure:"max_player_count,omitempty"`
	GameMode        string `mapstructure:"game_mode"`
	GameModeNumeric int    `mapstructure:"game_mode_numeric"`
	MOTD            string `mapstructure:"motd,omitempty"`
}

func newPingStatus(cfg pingStatusConfig) bedprox.PingStatus {
	return bedprox.PingStatus{
		Edition:         cfg.Edition,
		ProtocolVersion: cfg.ProtocolVersion,
		VersionName:     cfg.VersionName,
		PlayerCount:     cfg.PlayerCount,
		MaxPlayerCount:  cfg.MaxPlayerCount,
		GameMode:        cfg.GameMode,
		GameModeNumeric: cfg.GameModeNumeric,
		MOTD:            cfg.MOTD,
	}
}

type listenerConfig struct {
	Bind       string           `mapstructure:"bind"`
	PingStatus pingStatusConfig `mapstructure:"ping_status"`
}

func newListener(cfg listenerConfig) bedprox.Listener {
	return bedprox.Listener{
		Bind:       cfg.Bind,
		PingStatus: newPingStatus(cfg.PingStatus),
	}
}

func loadListeners(gatewayID string) ([]bedprox.Listener, error) {
	var listeners []bedprox.Listener
	for _, v := range viper.GetStringMap("gateways." + gatewayID + ".listeners") {
		vpr := viper.Sub("defaults.gateway.listener")
		vMap := v.(map[string]interface{})
		if err := vpr.MergeConfigMap(vMap); err != nil {
			return nil, err
		}
		var cfg listenerConfig
		if err := vpr.Unmarshal(&cfg); err != nil {
			return nil, err
		}
		listeners = append(listeners, newListener(cfg))
	}

	return listeners, nil
}

type gatewayConfig struct {
	Listeners            []listenerConfig `mapstructure:"-"`
	ReceiveProxyProtocol bool             `mapstructure:"receive_proxy_protocol"`
	ClientTimeout        time.Duration    `mapstructure:"client_timeout"`
	Servers              []string         `mapstructure:"servers"`
}

func newGateway(id string, cfg gatewayConfig) (bedprox.Gateway, error) {
	listeners, err := loadListeners(id)
	if err != nil {
		return bedprox.Gateway{}, err
	}

	return bedprox.Gateway{
		ID:                   id,
		Listeners:            listeners,
		ReceiveProxyProtocol: cfg.ReceiveProxyProtocol,
		ClientTimeout:        cfg.ClientTimeout,
		ServerIDs:            cfg.Servers,
	}, nil
}

func loadGateways() ([]bedprox.Gateway, error) {
	var gateways []bedprox.Gateway
	for id, v := range viper.GetStringMap("gateways") {
		vpr := viper.Sub("defaults.gateway")
		vMap := v.(map[string]interface{})
		if err := vpr.MergeConfigMap(vMap); err != nil {
			return nil, err
		}
		var cfg gatewayConfig
		if err := vpr.Unmarshal(&cfg); err != nil {
			return nil, err
		}
		gateway, err := newGateway(id, cfg)
		if err != nil {
			return nil, err
		}
		gateways = append(gateways, gateway)
	}

	return gateways, nil
}

type serverConfig struct {
	Domains           []string      `mapstructure:"domains"`
	Address           string        `mapstructure:"address"`
	ProxyBind         string        `mapstructure:"proxy_bind"`
	DialTimeout       time.Duration `mapstructure:"dial_timeout"`
	SendProxyProtocol bool          `mapstructure:"send_proxy_protocol"`
	DisconnectMessage string        `mapstructure:"disconnect_message"`
}

func newServer(id string, cfg serverConfig) bedprox.Server {
	return bedprox.Server{
		ID:      id,
		Domains: cfg.Domains,
		Dialer: raknet.Dialer{
			UpstreamDialer: &net.Dialer{
				Timeout: cfg.DialTimeout,
				LocalAddr: &net.UDPAddr{
					IP: net.ParseIP(cfg.ProxyBind),
				},
			},
		},
		Address:           cfg.Address,
		SendProxyProtocol: cfg.SendProxyProtocol,
		DisconnectMessage: cfg.DisconnectMessage,
	}
}

func loadServers() ([]bedprox.Server, error) {
	var servers []bedprox.Server
	for id, v := range viper.GetStringMap("servers") {
		vpr := viper.Sub("defaults.server")
		vMap := v.(map[string]interface{})
		if err := vpr.MergeConfigMap(vMap); err != nil {
			return nil, err
		}
		var cfg serverConfig
		if err := vpr.Unmarshal(&cfg); err != nil {
			return nil, err
		}
		servers = append(servers, newServer(id, cfg))
	}

	return servers, nil
}

type cpnConfig struct {
	Count int `mapstructure:"count"`
}

func loadCPNs() ([]bedprox.ConnProcessor, error) {
	var cfg cpnConfig
	if err := viper.UnmarshalKey("processing_nodes", &cfg); err != nil {
		return nil, err
	}

	return make([]bedprox.ConnProcessor, cfg.Count), nil
}

type webhookConfig struct {
	ClientTimeout time.Duration `mapstructure:"client_timeout"`
	URL           string        `mapstructure:"url"`
	Events        []string      `mapstructure:"events"`
}

func newWebhook(id string, cfg webhookConfig) webhook.Webhook {
	return webhook.Webhook{
		ID: id,
		HTTPClient: &http.Client{
			Timeout: cfg.ClientTimeout,
		},
		URL:        cfg.URL,
		EventTypes: cfg.Events,
	}
}

func loadWebhooks() ([]webhook.Webhook, error) {
	var webhooks []webhook.Webhook
	for id, v := range viper.GetStringMap("webhooks") {
		vpr := viper.Sub("defaults.webhook")
		vMap := v.(map[string]interface{})
		if err := vpr.MergeConfigMap(vMap); err != nil {
			return nil, err
		}
		var cfg webhookConfig
		if err := vpr.Unmarshal(&cfg); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, newWebhook(id, cfg))
	}

	return webhooks, nil
}
