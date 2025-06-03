package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/etkecc/go-apm"
	echobasicauth "github.com/etkecc/go-echo-basic-auth"
	_ "github.com/etkecc/mrs/docs" // required for swaggo
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/etkecc/mrs/internal/metrics"
	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/version"
)

type configService interface {
	Get() *model.Config
}

type statsService interface {
	Get() *model.IndexStats
	GetTL(context.Context) map[time.Time]*model.IndexStats
}

type cacheService interface {
	Middleware() echo.MiddlewareFunc
	MiddlewareSearch() echo.MiddlewareFunc
	MiddlewareImmutable() echo.MiddlewareFunc
}

type plausibleService interface {
	Track(ctx context.Context, evt *model.AnalyticsEvent)
}

var doNotGZIP = map[string]bool{
	"/avatar/:name/:id":                     true, // binary data
	"/_matrix/media/r0/thumbnail/:name/:id": true, // binary data
	"/_matrix/media/v3/thumbnail/:name/:id": true, // binary data
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
	plausibleSvc plausibleService,
) {
	configureRouter(e, cfg, cacheSvc)
	configureMatrixS2SEndpoints(e, matrixSvc, cacheSvc)
	configureMatrixCSEndpoints(e, matrixSvc, cacheSvc)

	e.GET("/metrics", echo.WrapHandler(&metrics.Handler{}), echobasicauth.NewMiddleware(&cfg.Get().Auth.Metrics))
	e.GET("/stats", stats(statsSvc))
	e.GET("/avatar/:name/:id", avatar(matrixSvc), cacheSvc.MiddlewareImmutable(), getRL(100))
	e.GET("/room/:room_id_or_alias", catalogRoom(dataSvc, matrixSvc, plausibleSvc), cacheSvc.Middleware(), getRL(3))
	e.GET("/catalog/rooms", rooms(dataSvc), echobasicauth.NewMiddleware(&cfg.Get().Auth.Catalog))
	e.GET("/catalog/servers", servers(crawlerSvc), echobasicauth.NewMiddleware(&cfg.Get().Auth.Catalog))

	rl := getRL(3)
	searchCache := cacheSvc.MiddlewareSearch()
	e.GET("/search", search(searchSvc, cfg, false), searchCache, rl)
	e.GET("/search/:q", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l/:o", search(searchSvc, cfg, true), searchCache, rl)
	e.GET("/search/:q/:l/:o/:s", search(searchSvc, cfg, true), searchCache, rl)

	e.POST("/discover/bulk", addServers(dataSvc, cfg), echobasicauth.NewMiddleware(&cfg.Get().Auth.Discovery))
	e.POST("/discover/:name", addServer(dataSvc), discoveryProtection(rl, cfg))
	e.POST("/discover/msc1929/:name", checkMSC1929(), getRL(1))

	e.POST("/mod/report/:room_id", report(modSvc), getRL(1)) // doesn't use mod group to allow without auth
	m := e.Group("mod")
	m.Use(echobasicauth.NewMiddleware(&cfg.Get().Auth.Moderation))
	m.GET("/list", listBanned(modSvc), rl)
	m.GET("/list/:server_name", listBanned(modSvc), rl)
	m.GET("/ban/:room_id", ban(modSvc), rl)
	m.GET("/unban/:room_id", unban(modSvc), rl)

	a := e.Group("-")
	a.Use(echobasicauth.NewMiddleware(&cfg.Get().Auth.Admin))
	a.GET("/status", status(statsSvc))
	a.POST("/discover", discover(dataSvc, cfg))
	a.POST("/parse", parse(dataSvc, cfg))
	a.POST("/reindex", reindex(dataSvc))
	a.POST("/full", full(dataSvc, cfg))
}

func configureRouter(e *echo.Echo, cfgSvc configService, cacheSvc cacheService) {
	e.Use(middleware.Recover())
	e.Use(apm.WithSentry())
	e.Use(cacheSvc.Middleware())
	e.Use(middleware.Secure())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{MaxAge: 86400}))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Skipper: func(c echo.Context) bool { return doNotGZIP[c.Path()] }}))
	e.Use(withBlocklist(cfgSvc))
	e.Use(withMContext(cfgSvc))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
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
	e.Any("/_health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/robots.txt", func(c echo.Context) error {
		return c.String(http.StatusOK, "User-agent: *\nDisallow: /")
	})
	e.GET("/_docs", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/_docs/index.html")
	})
	e.GET("/_docs/*", echoSwagger.WrapHandler)
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
