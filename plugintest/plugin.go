package main

import "log"

type plugin struct{}

func (p plugin) Load() error {
	log.Println("Plugin: Load")
	return nil
}

func (p plugin) Unload() error {
	log.Println("Plugin: Unload")
	return nil
}

func (p plugin) OnPlayerJoin() error {
	log.Println("Plugin: OnPlayerJoin")
	return nil
}

func (p plugin) OnPlayerLeave() error {
	log.Println("Plugin: OnPlayerLeave")
	return nil
}

var Plugin plugin
