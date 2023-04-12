package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"gitlab.com/etke.cc/mrs/api/config"
)

type statsService interface {
	GetRooms() int
	GetServers() int
	Collect()
}

// ConfigureRouter configures echo router
func ConfigureRouter(e *echo.Echo, cfg *config.Config, searchSvc searchService, indexSvc indexService, matrixSvc matrixService, statsSvc statsService) {
	configureRouter(e, cfg)
	e.GET("/stats", stats(statsSvc))
	e.POST("/discover/:name", addServer(matrixSvc), middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(1)))

	a := adminGroup(e, cfg)
	e.GET("/search", search(searchSvc))
	a.GET("/servers", servers(matrixSvc))
	a.POST("/discover", discover(matrixSvc, statsSvc, cfg.Workers.Discovery))
	a.POST("/parse", parse(matrixSvc, statsSvc, cfg.Workers.Parsing))
	a.POST("/reindex", reindex(matrixSvc, indexSvc, statsSvc))
	a.POST("/full", full(matrixSvc, indexSvc, statsSvc, cfg.Workers.Discovery, cfg.Workers.Parsing))
}

func configureRouter(e *echo.Echo, cfg *config.Config) {
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/_health"
		},
		Format:           `${remote_ip} - - [${time_custom}] "${method} ${path} ${protocol}" ${status} ${bytes_out} "${referer}" "${user_agent}"` + "\n",
		CustomTimeFormat: "2/Jan/2006:15:04:05 -0700",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(cfg.CORS))
	e.Use(cacheMiddleware)
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

func adminGroup(e *echo.Echo, cfg *config.Config) *echo.Group {
	admin := e.Group("-")
	admin.Use(middleware.BasicAuth(func(login, password string, ctx echo.Context) (bool, error) {
		if login != cfg.Admin.Login || password != cfg.Admin.Password {
			return false, nil
		}
		if len(cfg.Admin.IPs) == 0 {
			return true, nil
		}
		var allowed bool
		realIP := ctx.RealIP()
		for _, ip := range cfg.Admin.IPs {
			if ip == realIP {
				allowed = true
				break
			}
		}

		if allowed {
			return true, nil
		}

		return false, nil
	}))
	return admin
}

func cacheMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Method == http.MethodGet {
			c.Response().
				Header().
				Set("Cache-Control", "max-age=86400")
		}
		return next(c)
	}
}