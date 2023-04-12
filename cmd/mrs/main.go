package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"gitlab.com/etke.cc/go/fswatcher"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/controllers"
	"gitlab.com/etke.cc/mrs/api/repository/data"
	"gitlab.com/etke.cc/mrs/api/repository/search"
	"gitlab.com/etke.cc/mrs/api/services"
)

var (
	configPath    string
	configWatcher *fswatcher.Watcher
	dataRepo      *data.Data
	index         *search.Index
	cfg           *config.Config
	e             *echo.Echo
)

func main() {
	quit := make(chan struct{})
	flag.StringVar(&configPath, "c", "config.yml", "Path to the config file")
	flag.Parse()
	err := loadConfig()
	if err != nil {
		log.Panic(err)
	}
	startConfigWatcher()
	dataRepo, err = data.New(cfg.Path.Data)
	if err != nil {
		log.Panic(err)
	}

	index = createOrOpenIndex(cfg.Path.Index)
	indexSvc := services.NewIndex(index, dataRepo)
	searchSvc := services.NewSearch(index)
	matrixSvc := services.NewMatrix(cfg.Servers, dataRepo)
	statsSvc := services.NewStats(dataRepo)
	go statsSvc.Collect()
	e = echo.New()
	controllers.ConfigureRouter(e, cfg, searchSvc, indexSvc, matrixSvc, statsSvc)

	initShutdown(quit)

	if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
		log.Fatal("shutting down the server", err)
	}

	<-quit
}

func createOrOpenIndex(indexPath string) *search.Index {
	searchIndex, err := search.OpenIndex(indexPath)
	if err == nil {
		return searchIndex
	}
	searchIndex, err = search.NewIndex(indexPath)
	if err != nil {
		log.Panic(err)
	}
	return searchIndex
}

func initShutdown(quit chan struct{}) {
	listener := make(chan os.Signal, 1)
	signal.Notify(listener, os.Interrupt, syscall.SIGABRT, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	go func() {
		<-listener
		defer close(quit)

		shutdown()
	}()
}

func shutdown() {
	log.Println("shutting down...")
	if err := configWatcher.Stop(); err != nil {
		log.Println(err)
	}
	if err := index.Close(); err != nil {
		log.Println(err)
	}
	dataRepo.Close()
	// api was not started yet
	if e == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err) //nolint:gocritic // that's intended
	}
}
