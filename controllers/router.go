package controllers

import (
	"context"
	"net/http"
	"time"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/raja/argon2pw"
	"github.com/rs/zerolog"
	echobasicauth "gitlab.com/etke.cc/go/echo-basic-auth"

	"gitlab.com/etke.cc/mrs/api/metrics"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
	"gitlab.com/etke.cc/mrs/api/version"
)

type configService interface {
	Get() *model.Config
}

type statsService interface {
	Get() *model.IndexStats
	GetTL(context.Context) map[time.Time]*model.IndexStats
}

type cacheService interface {
	IsBunny(string) bool
	Middleware() echo.MiddlewareFunc
	MiddlewareSearch() echo.MiddlewareFunc
	MiddlewareImmutable() echo.MiddlewareFunc
}

// ConfigureRouter configures echo router
func ConfigureRouter(
	e *echo.Echo,
	cfg configService,
	matrixSvc matrixService,
	dataSvc dataService,
	cacheSvc cacheService,
	searchSvc searchService,
	crawlerSvc crawlerService,
	statsSvc statsService,
	modSvc moderationService,
) {
	configureRouter(e, cacheSvc)
	configureMatrixS2SEndpoints(e, matrixSvc, cacheSvc)
	configureMatrixCSEndpoints(e, matrixSvc, cacheSvc)
	rl := getRL(1, cacheSvc)
	e.GET("/metrics", echo.WrapHandler(&metrics.Handler{}), echobasicauth.NewMiddleware(&cfg.Get().Auth.Metrics))
	e.GET("/stats", stats(statsSvc))
	e.GET("/avatar/:name/:id", avatar(matrixSvc), getRL(30, cacheSvc))

	searchCache := cacheSvc.MiddlewareSearch()
	e.GET("/search", search(searchSvc, cfg, false), searchCache, rl)
	e.GET("/search/:q", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l/:o", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l/:o/:s", search(searchSvc, cfg, true), searchCache, rl)

	e.GET("/catalog/servers", catalogServers(dataSvc), cacheSvc.Middleware(), rl)

	e.POST("/discover/bulk", addServers(dataSvc, cfg), echobasicauth.NewMiddleware(&cfg.Get().Auth.Discovery))
	e.POST("/discover/:name", addServer(dataSvc), discoveryProtection(rl, cfg))

	e.POST("/mod/report/:room_id", report(modSvc), rl) // doesn't use mod group to allow without auth
	m := modGroup(e, cfg)
	m.GET("/list", listBanned(modSvc), rl)
	m.GET("/list/:server_name", listBanned(modSvc), rl)
	m.GET("/ban/:room_id", ban(modSvc), rl)
	m.GET("/unban/:room_id", unban(modSvc), rl)

	a := e.Group("-")
	a.Use(echobasicauth.NewMiddleware(&cfg.Get().Auth.Admin))
	a.GET("/servers", servers(crawlerSvc))
	a.GET("/status", status(statsSvc))
	a.POST("/discover", discover(dataSvc, cfg))
	a.POST("/parse", parse(dataSvc, cfg))
	a.POST("/reindex", reindex(dataSvc))
	a.POST("/full", full(dataSvc, cfg))
}

func configureRouter(e *echo.Echo, cacheSvc cacheService) {
	e.Use(middleware.Recover())
	e.Use(sentryecho.New(sentryecho.Options{}))
	e.Use(SentryTransaction())
	e.Use(cacheSvc.Middleware())
	e.Use(middleware.Secure())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderReferrerPolicy, "origin")
			c.Response().Header().Set(echo.HeaderServer, version.Server)
			return next(c)
		}
	})
	e.HideBanner = true
	e.IPExtractor = echo.ExtractIPFromXFFHeader(
		echo.TrustLoopback(true),
		echo.TrustLinkLocal(true),
		echo.TrustPrivateNet(true),
	)
	e.GET("/_health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
}

// discoveryProtection rate limits anonymous requests, but allows authorized with basic auth requests
func discoveryProtection(rl echo.MiddlewareFunc, cfg configService) echo.MiddlewareFunc {
	auth := echobasicauth.NewMiddleware(&cfg.Get().Auth.Discovery)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get(echo.HeaderAuthorization) != "" {
				return auth(next)(c)
			}
			return rl(next)(c)
		}
	}
}

func hashAuth(c echo.Context, authPassword string) *bool {
	hash := c.QueryParam("auth")
	if hash == "" {
		return nil
	}
	hashDecoded := utils.URLSafeDecode(hash)
	if hashDecoded != "" {
		hash = hashDecoded
	}

	var ok bool
	defer func() {
		zerolog.Ctx(c.Request().Context()).
			Info().
			Any("error", recover()).
			Str("section", "hash").
			Str("from", c.RealIP()).
			Str("path", c.Request().URL.Path).
			Bool("allowed_credentials", ok).
			Msg("authorization attempt")
	}()
	ok, _ = argon2pw.CompareHashWithPassword(hash, authPassword) //nolint:errcheck // we don't care about the error here

	return &ok
}

func modGroup(e *echo.Echo, cfg configService) *echo.Group {
	mod := e.Group("mod")
	authPassword := cfg.Get().Auth.Moderation.Login + cfg.Get().Auth.Moderation.Password
	mod.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ok := hashAuth(c, authPassword)
			if ok == nil {
				return next(c)
			}
			if !*ok {
				return echo.ErrUnauthorized
			}

			c.Set("authorized", true)
			return next(c)
		}
	})
	mod.Use(echobasicauth.NewMiddleware(&cfg.Get().Auth.Moderation))
	return mod
}
