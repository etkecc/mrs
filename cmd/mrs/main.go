package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mileusna/crontab"
	"github.com/pemistahl/lingua-go"

	"gitlab.com/etke.cc/mrs/api/controllers"
	"gitlab.com/etke.cc/mrs/api/repository/data"
	"gitlab.com/etke.cc/mrs/api/repository/search"
	"gitlab.com/etke.cc/mrs/api/services"
)

// AllLanguages to load all language models at once
const AllLanguages = "ALL"

var (
	configPath string
	runGenKey  bool
	dataRepo   *data.Data
	index      *search.Index
	cron       *crontab.Crontab
	e          *echo.Echo
)

func main() {
	quit := make(chan struct{})
	flag.StringVar(&configPath, "c", "config.yml", "Path to the config file")
	flag.BoolVar(&runGenKey, "genkey", false, "Generate matrix signing key")
	flag.Parse()
	if runGenKey {
		if _, err := generateKey(); err != nil {
			log.Panic(err)
		}
		return
	}

	cfg, err := services.NewConfig(configPath)
	if err != nil {
		log.Panic(err)
	}
	dataRepo, err = data.New(cfg.Get().Path.Data)
	if err != nil {
		log.Panic(err)
	}

	detector := getLanguageDetector(cfg.Get().Languages)
	index, err = search.NewIndex(cfg.Get().Path.Index, detector, "en")
	if err != nil {
		log.Panic(err)
	}
	robotsSvc := services.NewRobots()
	blockSvc := services.NewBlocklist(cfg)
	statsSvc := services.NewStats(cfg, dataRepo, index, blockSvc)
	indexSvc := services.NewIndex(cfg, index, dataRepo)
	searchSvc := services.NewSearch(cfg, index, blockSvc, statsSvc)
	matrixSvc, err := services.NewMatrix(cfg, dataRepo, searchSvc)
	if err != nil {
		log.Panic(err)
	}
	crawlerSvc := services.NewCrawler(cfg, matrixSvc, robotsSvc, blockSvc, dataRepo, detector)
	matrixSvc.SetDiscover(crawlerSvc.AddServer)
	cacheSvc := services.NewCache(cfg, statsSvc)
	dataSvc := services.NewDataFacade(crawlerSvc, indexSvc, statsSvc, cacheSvc)
	mailSvc := services.NewEmail(cfg)
	modSvc := services.NewModeration(cfg, dataRepo, index, mailSvc)

	e = echo.New()
	controllers.ConfigureRouter(e, cfg, matrixSvc, dataSvc, cacheSvc, searchSvc, crawlerSvc, statsSvc, modSvc)

	initCron(cfg, dataSvc)
	initShutdown(quit)

	if err := e.Start(":" + cfg.Get().Port); err != nil && err != http.ErrServerClosed {
		log.Fatal("shutting down the server", err)
	}

	<-quit
}

func generateKey() (string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("ed25519 %s %s", base64.RawURLEncoding.EncodeToString(pub[:4]), base64.RawStdEncoding.EncodeToString(priv.Seed()))
	log.Println("WARNING", "AHTUNG", "ATTENTION!", "new key has been generated")
	log.Println(key)

	return key, nil
}

func getLanguageDetector(inputLangs []string) lingua.LanguageDetector {
	builder := lingua.NewLanguageDetectorBuilder()
	if len(inputLangs) > 0 && inputLangs[0] == AllLanguages {
		return builder.
			FromAllLanguages().
			Build()
	}

	all := lingua.AllLanguages()
	enabled := make([]lingua.Language, 0)
	langs := make(map[string]bool, len(inputLangs))
	for _, inputLang := range inputLangs {
		langs[inputLang] = true
	}
	for _, lang := range all {
		if langs[lang.IsoCode639_1().String()] {
			enabled = append(enabled, lang)
		}
	}

	return builder.
		FromLanguages(enabled...).
		Build()
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

func initCron(cfg *services.Config, dataSvc *services.DataFacade) {
	cron = crontab.New()
	if schedule := cfg.Get().Cron.Discovery; schedule != "" {
		log.Println("cron", "discovery job enabled")
		cron.MustAddJob(schedule, dataSvc.DiscoverServers, cfg.Get().Workers.Discovery)
	}
	if schedule := cfg.Get().Cron.Parsing; schedule != "" {
		log.Println("cron", "parsing job enabled")
		cron.MustAddJob(schedule, dataSvc.ParseRooms, cfg.Get().Workers.Parsing)
	}
	if schedule := cfg.Get().Cron.Indexing; schedule != "" {
		log.Println("cron", "indexing job enabled")
		cron.MustAddJob(schedule, dataSvc.Ingest)
	}
	if schedule := cfg.Get().Cron.Full; schedule != "" {
		log.Println("cron", "full job enabled")
		cron.MustAddJob(schedule, dataSvc.Full, cfg.Get().Workers.Discovery, cfg.Get().Workers.Parsing)
	}
}

func shutdown() {
	log.Println("shutting down...")
	cron.Shutdown()
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
