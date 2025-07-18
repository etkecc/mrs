package controllers

import (
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/model/mcontext"
	"github.com/etkecc/mrs/internal/utils"
)

var mForbidden = utils.MustJSON(&model.MatrixError{
	Code:    "M_FORBIDDEN",
	Message: "forbidden",
})

func getRL(limit rate.Limit) echo.MiddlewareFunc {
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

	return middleware.RateLimiterWithConfig(cfg)
}

func getBlocklist(cfg configService) ([]string, []*net.IPNet) {
	blockIPs := []string{}
	blockCIDRs := []*net.IPNet{}
	if cfg.Get().Blocklist == nil || len(cfg.Get().Blocklist.IPs) == 0 {
		return blockIPs, blockCIDRs
	}

	for _, ip := range cfg.Get().Blocklist.IPs {
		if _, ipnet, err := net.ParseCIDR(ip); err == nil {
			blockCIDRs = append(blockCIDRs, ipnet)
		} else {
			blockIPs = append(blockIPs, ip)
		}
	}

	return blockIPs, blockCIDRs
}

func withBlocklist(cfg configService) echo.MiddlewareFunc {
	blockIPs, blockCIDRs := getBlocklist(cfg)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if ip == "" {
				return next(c)
			}

			if slices.Contains(blockIPs, ip) {
				return c.JSONBlob(http.StatusForbidden, mForbidden)
			}

			parsed := net.ParseIP(ip)
			for _, ipnet := range blockCIDRs {
				if ipnet.Contains(parsed) {
					return c.JSONBlob(http.StatusForbidden, mForbidden)
				}
			}

			return next(c)
		}
	}
}

// withMContext is a middleware that sets the context for the request
func withMContext(cfg configService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			ctx = mcontext.WithIP(ctx, c.RealIP())

			var origin string
			if parsed := utils.ParseURL(c.Request().Header.Get("Origin")); parsed != nil {
				origin = parsed.Hostname()
			}
			if origin == "" {
				if parsed := utils.ParseURL(c.Request().Header.Get("Referer")); parsed != nil {
					origin = parsed.Hostname()
				}
			}
			if origin == "" {
				origin = cfg.Get().Matrix.ServerName
			}
			ctx = mcontext.WithOrigin(ctx, origin)

			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
