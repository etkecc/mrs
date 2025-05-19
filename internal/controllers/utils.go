package controllers

import (
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

var (
	rls        = map[rate.Limit]echo.MiddlewareFunc{}
	mForbidden = utils.MustJSON(&model.MatrixError{
		Code:    "M_FORBIDDEN",
		Message: "forbidden",
	})
)

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
	cfg.ErrorHandler = func(c echo.Context, err error) error {
		message := "error while extracting identifier" // default message from middleware
		if err != nil {
			message = err.Error()
		}
		return c.JSONBlob(http.StatusForbidden, utils.MustJSON(&model.MatrixError{
			Code:    "M_FORBIDDEN",
			Message: message,
		}))
	}
	cfg.DenyHandler = func(c echo.Context, _ string, _ error) error {
		c.Response().Header().Set(echo.HeaderRetryAfter, "10")
		return c.JSONBlob(http.StatusTooManyRequests, utils.MustJSON(&model.MatrixError{
			Code:    "M_LIMIT_EXCEEDED",
			Message: "rate limit exceeded",
		}))
	}
	cfg.Store = middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      limit,
		Burst:     int(limit),
		ExpiresIn: 5 * time.Minute,
	})
	rls[limit] = middleware.RateLimiterWithConfig(cfg)
	return rls[limit]
}

func withBlocklist(cfg configService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Get().Blocklist == nil || len(cfg.Get().Blocklist.IPs) == 0 {
				return next(c)
			}
			ip := c.RealIP()
			if slices.Contains(cfg.Get().Blocklist.IPs, ip) {
				return c.JSONBlob(http.StatusForbidden, mForbidden)
			}
			return next(c)
		}
	}
}
