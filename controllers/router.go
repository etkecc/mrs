package controllers

import (
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/raja/argon2pw"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type statsService interface {
	Get() *model.IndexStats
}

type cacheService interface {
	Middleware() echo.MiddlewareFunc
	MiddlewareImmutable() echo.MiddlewareFunc
}

var basicAuthSkipper = func(c echo.Context) bool {
	v := c.Get("authorized")
	if v == nil {
		return false
	}
	skip, ok := v.(bool)
	if !ok {
		return false
	}
	return skip
}

// ConfigureRouter configures echo router
func ConfigureRouter(
	e *echo.Echo,
	cfg *config.Config,
	dataSvc dataService,
	cacheSvc cacheService,
	searchSvc searchService,
	matrixSvc matrixService,
	statsSvc statsService,
	modSvc moderationService,
) {
	configureRouter(e, cacheSvc)
	rl := middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(1))

	e.GET("/stats", stats(statsSvc))
	e.GET("/avatar/:name/:id", avatar(matrixSvc), middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: 30, Burst: 30, ExpiresIn: 5 * time.Minute})), cacheSvc.MiddlewareImmutable())
	e.GET("/search", search(searchSvc, false))
	e.GET("/search/:q", search(searchSvc, true))
	e.GET("/search/:q/:l", search(searchSvc, true))
	e.GET("/search/:q/:l/:o", search(searchSvc, true))
	e.GET("/search/:q/:l/:o/:s", search(searchSvc, true))
	e.POST("/discover/bulk", addServers(matrixSvc, cfg.Workers.Discovery), discoveryAuth(cfg))
	e.POST("/discover/:name", addServer(matrixSvc), discoveryProtection(rl, cfg))

	e.POST("/mod/report/:room_id", report(modSvc), rl) // doesn't use mod group to allow without auth
	m := modGroup(e, cfg)
	m.GET("/list", listBanned(modSvc), rl)
	m.GET("/list/:server_name", listBanned(modSvc), rl)
	m.GET("/ban/:room_id", ban(modSvc), rl)
	m.GET("/unban/:room_id", unban(modSvc), rl)

	a := adminGroup(e, cfg)
	a.GET("/servers", servers(matrixSvc))
	a.GET("/status", status(statsSvc))
	a.POST("/discover", discover(dataSvc, cfg.Workers.Discovery))
	a.POST("/parse", parse(dataSvc, cfg.Workers.Parsing))
	a.POST("/reindex", reindex(dataSvc))
	a.POST("/full", full(dataSvc, cfg.Workers.Discovery, cfg.Workers.Parsing))
}

func configureRouter(e *echo.Echo, cacheSvc cacheService) {
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/_health"
		},
		Format:           `${remote_ip} - - [${time_custom}] "${method} ${path} ${protocol}" ${status} ${bytes_out} "${referer}" "${user_agent}"` + "\n",
		CustomTimeFormat: "2/Jan/2006:15:04:05 -0700",
	}))
	e.Use(middleware.Recover())
	e.Use(cacheSvc.Middleware())
	e.Use(middleware.Secure())
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

func discoveryAuth(cfg *config.Config) echo.MiddlewareFunc {
	return middleware.BasicAuth(func(login, password string, ctx echo.Context) (bool, error) {
		if login != cfg.Auth.Discovery.Login || password != cfg.Auth.Discovery.Password {
			return false, nil
		}
		return true, nil
	})
}

// discoveryProtection rate limits anonymous requests, but allows authorized with basic auth requests
func discoveryProtection(rl echo.MiddlewareFunc, cfg *config.Config) echo.MiddlewareFunc {
	auth := discoveryAuth(cfg)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if len(c.Request().Header.Get(echo.HeaderAuthorization)) > 0 {
				return auth(next)(c)
			}
			return rl(next)(c)
		}
	}
}

func adminGroup(e *echo.Echo, cfg *config.Config) *echo.Group {
	admin := e.Group("-")
	admin.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Skipper: basicAuthSkipper,
		Validator: func(login, password string, ctx echo.Context) (bool, error) {
			if login != cfg.Auth.Admin.Login || password != cfg.Auth.Admin.Password {
				return false, nil
			}
			log.Println("attempt to authorize as admin from:", ctx.RealIP())
			if len(cfg.Auth.Admin.IPs) == 0 {
				return true, nil
			}
			var allowed bool
			realIP := ctx.RealIP()
			for _, ip := range cfg.Auth.Admin.IPs {
				if ip == realIP {
					allowed = true
					break
				}
			}

			if allowed {
				return true, nil
			}

			return false, nil
		},
	}))

	return admin
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
	ok, _ := argon2pw.CompareHashWithPassword(hash, authPassword) //nolint:errcheck
	return &ok
}

func modGroup(e *echo.Echo, cfg *config.Config) *echo.Group {
	mod := e.Group("mod")
	authPassword := cfg.Auth.Moderation.Login + cfg.Auth.Moderation.Password
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
	mod.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Skipper: basicAuthSkipper,
		Validator: func(login, password string, ctx echo.Context) (bool, error) {
			if login != cfg.Auth.Moderation.Login || password != cfg.Auth.Moderation.Password {
				return false, nil
			}
			return true, nil
		},
	}))
	return mod
}
