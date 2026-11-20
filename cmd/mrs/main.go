package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/etkecc/go-apm/v2"
	"github.com/etkecc/go-healthchecks/v2"
	"github.com/etkecc/go-kit/crontab"
	"github.com/etkecc/go-msc1929"
	"github.com/labstack/echo/v5"
	"github.com/pemistahl/lingua-go"

	"github.com/etkecc/mrs/internal/controllers"
	"github.com/etkecc/mrs/internal/repository/data"
	"github.com/etkecc/mrs/internal/repository/search"
	"github.com/etkecc/mrs/internal/services"
	"github.com/etkecc/mrs/internal/services/matrix"
	"github.com/etkecc/mrs/internal/version"
)

// AllLanguages to load all language models at once
const AllLanguages = "ALL"

// gracefulTimeout bounds how long the HTTP server drains before it is stopped
const gracefulTimeout = 5 * time.Second

// healthchecksEvery is the healthchecks.io ping interval
const healthchecksEvery = 60 * time.Second

// teardownTimeout bounds the cron drain on shutdown
const teardownTimeout = 5 * time.Second

var (
	configPath string
	runGenKey  bool
	dataRepo   *data.Data
	index      *search.Index
	cron       *crontab.Crontab
	log        *slog.Logger
	hc         *healthchecks.Client
	e          *echo.Echo
)

//	@title			Matrix Rooms Search API
//	@version		0.0.0
//	@description	Matrix Rooms Search crawls the Matrix federation and serves public-room search. It also answers part of the Matrix federation and client-server APIs for the rooms it has indexed. Where our implementation diverges from the spec, that divergence is documented here on purpose, not hidden. matrixrooms.info is a public demo running MRS.

//	@accept		json
//	@produce	json
//	@schemes	https

//	@license.name	AGPL-3.0
//	@license.url	https://www.gnu.org/licenses/agpl-3.0

//	@securityDefinitions.basic	MetricsAuth
//	@securityDefinitions.basic	CatalogAuth
//	@securityDefinitions.basic	DiscoveryAuth
//	@securityDefinitions.basic	ModerationAuth
//	@securityDefinitions.basic	AdminAuth

// @securityDefinitions.apikey	FederationAuth
// @in							header
// @name						Authorization
// @description				Matrix federation signature auth: an `Authorization: X-Matrix origin=...,key=...,sig=...` header carrying an ed25519 signature over the request, which we verify against the origin server's published keys.
func main() {
	os.Exit(run())
}

// run parses the config, starts the service and reports the outcome to healthchecks
func run() (code int) {
	flag.StringVar(&configPath, "c", "config.yml", "Path to the config file")
	flag.BoolVar(&runGenKey, "genkey", false, "Generate matrix signing key")
	flag.Parse()

	cfg, err := services.NewConfig(configPath)
	if err != nil {
		apm.NewLoggerPlain().Error("cannot read config", "error", err)
		return 1
	}
	if cfg.Get() == nil {
		apm.NewLoggerPlain().Error("config is unreadable", "path", configPath)
		return 1
	}

	apm.SetName("MRS")
	apm.SetSentryDSN(cfg.Get().SentryDSN)
	apm.SetLogLevel("info")
	log = apm.Log()
	defer apm.Flush()

	if hcCfg := cfg.Get().Healthchecks; hcCfg != nil && hcCfg.UUID != "" {
		hc = healthchecks.New(
			healthchecks.WithBaseURL(hcCfg.URL),
			healthchecks.WithCheckUUID(hcCfg.UUID),
		)
		apm.SetHealthchecks(hc)
		hc.Start(strings.NewReader("MRS is starting..."))
		go hc.Auto(healthchecksEvery)
		defer func() {
			hc.ExitStatus(code, strings.NewReader(exitStatus(code)))
			hc.Shutdown()
		}()
	}

	return serve(cfg)
}

// exitStatus describes the process outcome in the healthchecks exit ping
func exitStatus(code int) string {
	if code == 0 {
		return "MRS stopped"
	}
	return fmt.Sprintf("MRS stopped with exit code %d", code)
}

// serve builds the services, serves until a signal or a listener failure and returns the exit code
func serve(cfg *services.Config) (code int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Error("panic recovered", "error", fmt.Sprintf("%+v", recovered), "stack", string(debug.Stack()))
			code = 1
		}
	}()

	if err := cfg.Validate(); err != nil {
		log.Error("invalid config", "error", err)
		return 1
	}

	if runGenKey {
		if _, err := generateKey(); err != nil {
			log.Error("cannot generate key", "error", err)
			return 1
		}
		return 0
	}

	msc1929.UserAgent = version.UserAgent
	repo, err := data.New(cfg.Get().Path.Data)
	if err != nil {
		log.Error("cannot open data repo", "error", err)
		return 1
	}
	dataRepo = repo
	media, err := services.NewMedia(cfg)
	if err != nil {
		log.Error("cannot open media repo", "error", err)
		return 1
	}

	detector := getLanguageDetector(cfg.Get().Languages)
	idx, err := search.NewIndex(cfg.Get().Path.Index, detector, "en")
	if err != nil {
		log.Error("cannot open index repo", "error", err)
		return 1
	}
	index = idx
	plausibleSvc := services.NewPlausible(cfg)
	blockSvc := services.NewBlocklist(cfg)
	statsSvc := services.NewStats(cfg, dataRepo, index, blockSvc)
	indexSvc := services.NewIndex(cfg, index)
	searchSvc := services.NewSearch(cfg, dataRepo, index, blockSvc, statsSvc, plausibleSvc)
	matrixSvc, err := matrix.NewServer(cfg, dataRepo, media, searchSvc, blockSvc)
	if err != nil {
		log.Error("cannot start matrix service", "error", err)
		return 1
	}
	validatorSvc := services.NewValidator(cfg, blockSvc, matrixSvc)
	crawlerSvc := services.NewCrawler(cfg, matrixSvc, validatorSvc, blockSvc, media, dataRepo, detector)
	matrixSvc.SetDiscover(crawlerSvc.AddServer)
	cacheSvc := services.NewCache(cfg, statsSvc)
	dataSvc := services.NewDataFacade(crawlerSvc, indexSvc, statsSvc)
	mailSvc := services.NewEmail(cfg)
	modSvc := services.NewModeration(cfg, dataRepo, media, index, mailSvc, matrixSvc)

	e = echo.New()
	e.Logger = log
	controllers.ConfigureRouter(e, cfg, matrixSvc, dataSvc, cacheSvc, searchSvc, crawlerSvc, statsSvc, modSvc, plausibleSvc)

	initCron(cfg, dataSvc)
	defer stop()

	serveCtx, cancelServe := context.WithCancel(apm.NewContext())
	defer cancelServe()
	initShutdown(cancelServe)

	sc := echo.StartConfig{
		Address:         cfg.Get().Address + ":" + cfg.Get().Port,
		HideBanner:      true,
		GracefulTimeout: gracefulTimeout,
	}
	if serveErr := sc.Start(serveCtx, e); serveErr != nil {
		log.Error("http server failed", "error", serveErr)
		return 1
	}

	log.Info("stopped")
	return 0
}

func generateKey() (string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("ed25519 %s %s", base64.RawURLEncoding.EncodeToString(pub[:4]), base64.RawStdEncoding.EncodeToString(priv.Seed()))
	log.Warn("ATTENTION! A new key has been generated", "key", key)

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

// initShutdown cancels the serve context on the first termination signal
func initShutdown(cancel context.CancelFunc) {
	listener := make(chan os.Signal, 1)
	signal.Notify(listener, os.Interrupt, syscall.SIGABRT, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	go func() {
		log.Info("shutting down...", "signal", (<-listener).String())
		cancel()
	}()
}

func initCron(cfg *services.Config, dataSvc *services.DataFacade) {
	ctx := apm.NewContext()
	cron = crontab.New(crontab.WithPanicHandler(func(spec string, recovered any) {
		log.Error("cron job panicked", "spec", spec, "recover", recovered)
	}))

	if schedule := cfg.Get().Cron.Discovery; schedule != "" {
		log.Info("cron job enabled", "job", "discovery")
		workers := cfg.Get().Workers.Discovery
		cron.MustAddJob(schedule, func() { dataSvc.DiscoverServers(ctx, workers) })
	}
	if schedule := cfg.Get().Cron.Parsing; schedule != "" {
		log.Info("cron job enabled", "job", "parsing")
		workers := cfg.Get().Workers.Parsing
		cron.MustAddJob(schedule, func() { dataSvc.ParseRooms(ctx, workers) })
	}
	if schedule := cfg.Get().Cron.Indexing; schedule != "" {
		log.Info("cron job enabled", "job", "indexing")
		cron.MustAddJob(schedule, func() { dataSvc.Ingest(ctx) })
	}
	if schedule := cfg.Get().Cron.Full; schedule != "" {
		log.Info("cron job enabled", "job", "full")
		dw, pw := cfg.Get().Workers.Discovery, cfg.Get().Workers.Parsing
		cron.MustAddJob(schedule, func() { dataSvc.Full(ctx, dw, pw) })
	}
}

// stop drains cron and closes the repositories once the HTTP server is down
func stop() {
	cronCtx, cronCancel := context.WithTimeout(apm.NewContext(), teardownTimeout)
	defer cronCancel()
	if err := cron.Shutdown(cronCtx); err != nil {
		log.Warn("cron shutdown did not drain cleanly", "error", err)
	}
	if err := index.Close(); err != nil {
		log.Error("cannot close the index", "error", err)
	}
	dataRepo.Close()
}
