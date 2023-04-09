package main

import (
	"log"

	"github.com/fsnotify/fsnotify"
	"gitlab.com/etke.cc/go/fswatcher"

	"gitlab.com/etke.cc/int/mrs/config"
)

func loadConfig() error {
	newcfg, err := config.Read(configPath)
	if err == nil {
		cfg = newcfg
		return nil
	}
	return err
}

func startConfigWatcher() {
	var err error
	configWatcher, err = fswatcher.New([]string{configPath}, 0)
	if err != nil {
		log.Panic(err)
	}

	go configWatcher.Start(func(_ fsnotify.Event) {
		err := loadConfig()
		if err != nil {
			log.Println("cannot reload config:", err)
		}
	})
}
