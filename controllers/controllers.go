package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"gitlab.com/etke.cc/int/mrs/config"
)

type indexerService interface {
	searchService
	indexService
}

// ConfigureRouter configures echo router
func ConfigureRouter(e *echo.Echo, cfg *config.Config, indexSvc indexerService, matrixSvc matrixService) {
	configureRouter(e, cfg)
	a := adminGroup(e, cfg)
	e.GET("/search", search(indexSvc))
	a.GET("/servers", servers(matrixSvc))
	a.POST("/discover", discover(matrixSvc, cfg.Workers.Discovery))
	a.POST("/parse", parse(matrixSvc, indexSvc, cfg.Workers.Parsing))
	a.POST("/reindex", reindex(matrixSvc, indexSvc))
	a.POST("/full", full(matrixSvc, indexSvc, cfg.Workers.Discovery, cfg.Workers.Parsing))
}

func configureRouter(e *echo.Echo, cfg *config.Config) {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(cfg.CORS))
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
