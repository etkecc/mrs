package controllers

import (
	"bytes"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/raja/argon2pw"
	"golang.org/x/exp/slices"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
	"gitlab.com/etke.cc/mrs/api/version"
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

	e.GET("/.well-known/matrix/server", wellKnownServer(cfg.Public.API))
	e.GET("/_matrix/federation/v1/version", matrixFederationVersion())
	e.GET("/_matrix/key/v2/server", matrixKeyServer(&cfg.Matrix))

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
		Format:           `${remote_ip} - ${custom} [${time_custom}] "${method} ${path} ${protocol}" ${status} ${bytes_out} "${referer}" "${user_agent}"` + "\n",
		CustomTimeFormat: "2/Jan/2006:15:04:05 -0700",
		CustomTagFunc:    logBasicAuthLogin,
	}))
	e.Use(middleware.Recover())
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

// logBasicAuthLogin parses basic auth login (if provided) from headers
func logBasicAuthLogin(c echo.Context, buf *bytes.Buffer) (int, error) {
	auth := c.Request().Header.Get(echo.HeaderAuthorization)
	l := len("basic")

	if len(auth) > l+1 && strings.EqualFold(auth[:l], "basic") {
		// Invalid base64 shouldn't be treated as error
		// instead should be treated as invalid client input
		b, err := base64.StdEncoding.DecodeString(auth[l+1:])
		if err != nil {
			return buf.WriteRune('-') //nolint:gocritic // interface constraint
		}

		cred := string(b)
		for i := 0; i < len(cred); i++ {
			if cred[i] == ':' {
				return buf.WriteString(cred[:i])
			}
		}
	}
	return buf.WriteRune('-') //nolint:gocritic // interface constraint
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
			allowedIP := true
			if len(cfg.Auth.Admin.IPs) != 0 {
				allowedIP = slices.Contains(cfg.Auth.Admin.IPs, ctx.RealIP())
			}
			match := utils.ConstantTimeEq(cfg.Auth.Admin.Login, login) && utils.ConstantTimeEq(cfg.Auth.Admin.Password, password)

			log.Printf("admin authorization attempt from=%s path=%s allowed_ip=%t allowed_credentials=%t",
				ctx.RealIP(), ctx.Request().URL.Path, allowedIP, match,
			)
			return match && allowedIP, nil
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

	var ok bool
	defer func() {
		log.Printf("hash authorization attempt from=%s path=%s allowed_credentials=%t panic=%v", c.RealIP(), c.Request().URL.Path, ok, recover())
	}()
	ok, _ = argon2pw.CompareHashWithPassword(hash, authPassword) //nolint:errcheck

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
			match := utils.ConstantTimeEq(cfg.Auth.Moderation.Login, login) && utils.ConstantTimeEq(cfg.Auth.Moderation.Password, password)
			log.Printf("mod authorization attempt from=%s path=%s allowed_credentials=%t",
				ctx.RealIP(), ctx.Request().URL.Path, match,
			)
			return match, nil
		},
	}))
	return mod
}
