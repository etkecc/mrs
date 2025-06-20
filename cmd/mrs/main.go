package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/go-healthchecks/v2"
	"github.com/etkecc/go-msc1929"
	"github.com/labstack/echo/v4"
	"github.com/mileusna/crontab"
	"github.com/pemistahl/lingua-go"
	"github.com/rs/zerolog"
	"github.com/ziflex/lecho/v3"

	"github.com/etkecc/mrs/internal/controllers"
	"github.com/etkecc/mrs/internal/repository/data"
	"github.com/etkecc/mrs/internal/repository/search"
	"github.com/etkecc/mrs/internal/services"
	"github.com/etkecc/mrs/internal/services/matrix"
	"github.com/etkecc/mrs/internal/version"
)

// AllLanguages to load all language models at once
const AllLanguages = "ALL"

var (
	configPath string
	runGenKey  bool
	dataRepo   *data.Data
	index      *search.Index
	cron       *crontab.Crontab
	log        *zerolog.Logger
	hc         *healthchecks.Client
	e          *echo.Echo
)

func main() {
	quit := make(chan struct{})
	flag.StringVar(&configPath, "c", "config.yml", "Path to the config file")
	flag.BoolVar(&runGenKey, "genkey", false, "Generate matrix signing key")
	flag.Parse()

	cfg, err := services.NewConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot read config")
	}

	apm.SetName("MRS")
	apm.SetSentryDSN(cfg.Get().SentryDSN)
	apm.SetLogLevel("info")
	log = apm.Log()
	if cfg.Get().Healthchecks.UUID != "" {
		hc = healthchecks.New(
			healthchecks.WithBaseURL(cfg.Get().Healthchecks.URL),
			healthchecks.WithCheckUUID(cfg.Get().Healthchecks.UUID),
		)
		apm.SetHealthchecks(hc)
		hc.Start(strings.NewReader("MRS is starting..."))
		go hc.Auto(60 * time.Second)
	}

	if runGenKey {
		if _, err := generateKey(); err != nil {
			log.Fatal().Err(err).Msg("cannot generate key")
		}
		return
	}

	msc1929.UserAgent = version.UserAgent
	dataRepo, err = data.New(cfg.Get().Path.Data)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot open data repo")
	}
	media, err := services.NewMedia(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot open media repo")
	}

	detector := getLanguageDetector(cfg.Get().Languages)
	index, err = search.NewIndex(cfg.Get().Path.Index, detector, "en")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot open index repo")
	}
	plausibleSvc := services.NewPlausible(cfg)
	blockSvc := services.NewBlocklist(cfg)
	statsSvc := services.NewStats(cfg, dataRepo, index, blockSvc)
	indexSvc := services.NewIndex(cfg, index)
	searchSvc := services.NewSearch(cfg, dataRepo, index, blockSvc, statsSvc, plausibleSvc)
	matrixSvc, err := matrix.NewServer(cfg, dataRepo, media, searchSvc, blockSvc)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start matrix service")
	}
	validatorSvc := services.NewValidator(cfg, blockSvc, matrixSvc)
	crawlerSvc := services.NewCrawler(cfg, matrixSvc, validatorSvc, blockSvc, media, dataRepo, detector)
	matrixSvc.SetDiscover(crawlerSvc.AddServer)
	cacheSvc := services.NewCache(cfg, statsSvc)
	dataSvc := services.NewDataFacade(crawlerSvc, indexSvc, statsSvc)
	mailSvc := services.NewEmail(cfg)
	modSvc := services.NewModeration(cfg, dataRepo, media, index, mailSvc, matrixSvc)

	e = echo.New()
	e.Logger = lecho.From(*log)
	controllers.ConfigureRouter(e, cfg, matrixSvc, dataSvc, cacheSvc, searchSvc, crawlerSvc, statsSvc, modSvc, plausibleSvc)

	initCron(cfg, dataSvc)
	initShutdown(quit)

	if err := e.Start(":" + cfg.Get().Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("http server failed")
	}

	<-quit
}

func generateKey() (string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("ed25519 %s %s", base64.RawURLEncoding.EncodeToString(pub[:4]), base64.RawStdEncoding.EncodeToString(priv.Seed()))
	log.Warn().Str("key", key).Msg("ATTENTION! A new key has been generated")

	return key, nil
}

func getLanguageDetector(inputLangs []string) lingua.LanguageDetector {
	builder := lingua.NewLanguageDetectorBuilder()
	if len(inputLangs) > 0 && inputLangs[0] == AllLanguages {
		return builder.
			FromAllSpokenLanguages().
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
	ctx := apm.NewContext()
	cron = crontab.New()
	if schedule := cfg.Get().Cron.Discovery; schedule != "" {
		log.Info().Str("job", "discovery").Msg("cron job enabled")
		cron.MustAddJob(schedule, dataSvc.DiscoverServers, ctx, cfg.Get().Workers.Discovery)
	}
	if schedule := cfg.Get().Cron.Parsing; schedule != "" {
		log.Info().Str("job", "parsing").Msg("cron job enabled")
		cron.MustAddJob(schedule, dataSvc.ParseRooms, ctx, cfg.Get().Workers.Parsing)
	}
	if schedule := cfg.Get().Cron.Indexing; schedule != "" {
		log.Info().Str("job", "indexing").Msg("cron job enabled")
		cron.MustAddJob(schedule, dataSvc.Ingest, ctx)
	}
	if schedule := cfg.Get().Cron.Full; schedule != "" {
		log.Info().Str("job", "full").Msg("cron job enabled")
		cron.MustAddJob(schedule, dataSvc.Full, ctx, cfg.Get().Workers.Discovery, cfg.Get().Workers.Parsing)
	}
}

func shutdown() {
	log.Info().Msg("shutting down...")
	defer apm.Flush()
	cron.Shutdown()
	if err := index.Close(); err != nil {
		log.Error().Err(err).Msg("cannot close the index")
	}
	dataRepo.Close()
	if hc != nil {
		hc.ExitStatus(0)
	}
	// api was not started yet
	if e == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("http server shutdown failed") //nolint:gocritic // we are shutting down
	}
}
