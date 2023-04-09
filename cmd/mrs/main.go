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

	"gitlab.com/etke.cc/int/mrs/config"
	"gitlab.com/etke.cc/int/mrs/controllers"
	"gitlab.com/etke.cc/int/mrs/model"
	"gitlab.com/etke.cc/int/mrs/repository/search"
	"gitlab.com/etke.cc/int/mrs/services"
)

var (
	configPath    string
	configWatcher *fswatcher.Watcher
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
	index = createOrOpenIndex(cfg.Path.Index)
	searchSvc := services.NewSearch(index)
	initShutdown(quit)

	e = echo.New()
	controllers.ConfigureRouter(e, cfg, searchSvc)

	if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
		log.Fatal("shutting down the server", err)
	}
	index.Search("something", 1, 0)

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
	searchIndex.Index("!gqlCuoCdhufltluRXk:etke.cc", model.Entry{
		ID:     "!gqlCuoCdhufltluRXk:etke.cc",
		Alias:  "#news:etke.cc",
		Name:   "etke.cc | news",
		Topic:  "#service:etke.cc Matrix homeservers hosting, setup and maintenance",
		Avatar: "mxc://etke.cc/sENrPlXsrqeJworlfatgzHmu",
	})
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
