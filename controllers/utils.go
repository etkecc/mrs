package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"gitlab.com/etke.cc/mrs/api/utils"
)

var rls = map[rate.Limit]echo.MiddlewareFunc{}

// SentryTransaction is a middleware that creates a new transaction for each request.
func SentryTransaction() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().URL.Path == "/_health" {
				return next(c)
			}
			ctx := c.Request().Context()
			ctx = context.WithoutCancel(ctx) // especially useful for log running admin requests
			ctx = utils.NewContext(ctx)
			options := []sentry.SpanOption{
				sentry.WithOpName("http.server"),
				sentry.ContinueFromRequest(c.Request()),
				sentry.WithTransactionSource(sentry.SourceURL),
			}

			path := c.Path()
			if path == "" || path == "/" {
				path = c.Request().URL.Path
			}

			transaction := sentry.StartTransaction(ctx, fmt.Sprintf("%s %s", c.Request().Method, path), options...)
			defer transaction.Finish()

			c.SetRequest(c.Request().WithContext(transaction.Context()))

			if err := next(c); err != nil {
				transaction.Status = sentry.HTTPtoSpanStatus(c.Response().Status)
				return err
			}
			transaction.Status = sentry.SpanStatusOK
			return nil
		}
	}
}

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
