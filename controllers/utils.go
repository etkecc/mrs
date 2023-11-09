package controllers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"gitlab.com/etke.cc/mrs/api/utils"
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

func getRL(rate rate.Limit, cacheSvc cacheService) echo.MiddlewareFunc {
	rl, ok := rls[rate]
	if ok {
		return rl
	}
	cfg := middleware.DefaultRateLimiterConfig
	cfg.Skipper = func(c echo.Context) bool {
		if c.Request().Method == http.MethodOptions {
			return true
		}
		return cacheSvc.IsBunny(c.RealIP())
	}
	cfg.Store = middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate,
		Burst:     int(rate),
		ExpiresIn: 5 * time.Minute,
	})
	rls[rate] = middleware.RateLimiterWithConfig(cfg)
	return rls[rate]
}
