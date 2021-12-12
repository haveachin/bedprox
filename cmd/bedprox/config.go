package main

import (
	"bytes"
	"errors"
	"log"
	"os"

	_ "embed"

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
