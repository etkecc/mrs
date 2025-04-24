package controllers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/etkecc/mrs/internal/utils"
)

var rls = map[rate.Limit]echo.MiddlewareFunc{}

// getOrigin returns the origin of the request (if provided), or referer (if provided), or the MRS server name
func getOrigin(cfg configService, r *http.Request) string {
	var origin string
	if parsed := utils.ParseURL(r.Header.Get("Origin")); parsed != nil {
		origin = parsed.Hostname()
	}
	if origin == "" {
		if parsed := utils.ParseURL(r.Header.Get("Referer")); parsed != nil {
			origin = parsed.Hostname()
		}
	}
	if origin == "" {
		origin = cfg.Get().Matrix.ServerName
	}
	return origin
}

func getRL(limit rate.Limit) echo.MiddlewareFunc {
	rl, ok := rls[limit]
	if ok {
		return rl
	}
	cfg := middleware.DefaultRateLimiterConfig
	cfg.Skipper = func(c echo.Context) bool {
		return c.Request().Method == http.MethodOptions
	}
	cfg.Store = middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      limit,
		Burst:     int(limit),
		ExpiresIn: 5 * time.Minute,
	})
	rls[limit] = middleware.RateLimiterWithConfig(cfg)
	return rls[limit]
}
