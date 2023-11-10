package controllers

import (
	"net/http"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/raja/argon2pw"
	"golang.org/x/exp/slices"

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
	configureMatrixCSEndpoints(e, matrixSvc, crawlerSvc, cacheSvc)
	rl := getRL(1, cacheSvc)
	e.GET("/metrics", echo.WrapHandler(&metrics.Handler{}), auth("metrics", &cfg.Get().Auth.Metrics))
	e.GET("/stats", stats(statsSvc))
	e.GET("/avatar/:name/:id", avatar(crawlerSvc), getRL(30, cacheSvc))

	searchCache := cacheSvc.MiddlewareSearch()
	e.GET("/search", search(searchSvc, cfg, false), searchCache, rl)
	e.GET("/search/:q", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l/:o", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l/:o/:s", search(searchSvc, cfg, true), searchCache, rl)

	e.POST("/discover/bulk", addServers(dataSvc, cfg), auth("discovery", &cfg.Get().Auth.Discovery))
	e.POST("/discover/:name", addServer(dataSvc), discoveryProtection(rl, cfg))

	e.POST("/mod/report/:room_id", report(modSvc), rl) // doesn't use mod group to allow without auth
	m := modGroup(e, cfg)
	m.GET("/list", listBanned(modSvc), rl)
	m.GET("/list/:server_name", listBanned(modSvc), rl)
	m.GET("/ban/:room_id", ban(modSvc), rl)
	m.GET("/unban/:room_id", unban(modSvc), rl)

	a := e.Group("-")
	a.Use(auth("admin", &cfg.Get().Auth.Admin))
	a.GET("/servers", servers(crawlerSvc))
	a.GET("/status", status(statsSvc))
	a.POST("/discover", discover(dataSvc, cfg))
	a.POST("/parse", parse(dataSvc, cfg))
	a.POST("/reindex", reindex(dataSvc))
	a.POST("/full", full(dataSvc, cfg))
}

func configureRouter(e *echo.Echo, cacheSvc cacheService) {
	e.Use(middleware.Recover())
	e.Use(sentryecho.New(sentryecho.Options{Repanic: true}))
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

func auth(name string, cfg *model.ConfigAuthItem) echo.MiddlewareFunc {
	return middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Skipper: func(c echo.Context) bool { // hash auth
			v := c.Get("authorized")
			if v == nil {
				return false
			}
			skip, ok := v.(bool)
			if !ok {
				return false
			}
			return skip
		},
		Validator: func(login, password string, ctx echo.Context) (bool, error) {
			allowedIP := true
			if len(cfg.IPs) != 0 {
				allowedIP = slices.Contains(cfg.IPs, ctx.RealIP())
			}
			match := utils.ConstantTimeEq(cfg.Login, login) && utils.ConstantTimeEq(cfg.Password, password)
			utils.Logger.
				Info().
				Str("section", name).
				Str("from", ctx.RealIP()).
				Str("path", ctx.Request().URL.Path).
				Bool("allowed_ip", allowedIP).
				Bool("allowed_credentials", match).
				Msg("authorization attempt")
			return match && allowedIP, nil
		},
	})
}

// discoveryProtection rate limits anonymous requests, but allows authorized with basic auth requests
func discoveryProtection(rl echo.MiddlewareFunc, cfg configService) echo.MiddlewareFunc {
	auth := auth("discovery", &cfg.Get().Auth.Discovery)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if len(c.Request().Header.Get(echo.HeaderAuthorization)) > 0 {
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
		utils.Logger.
			Info().
			Any("error", recover()).
			Str("section", "hash").
			Str("from", c.RealIP()).
			Str("path", c.Request().URL.Path).
			Bool("allowed_credentials", ok).
			Msg("authorization attempt")
	}()
	ok, _ = argon2pw.CompareHashWithPassword(hash, authPassword) //nolint:errcheck

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
	mod.Use(auth("mod", &cfg.Get().Auth.Moderation))
	return mod
}
