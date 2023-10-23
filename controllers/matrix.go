package controllers

import (
	"log"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/version"
)

// /.well-known/matrix/server
func wellKnownServer(host string) echo.HandlerFunc {
	uri, err := url.Parse(host)
	if err != nil {
		log.Println("ERROR: cannot parse public api host to use in /.well-known/matrix/server:", err)
	}
	port := uri.Port()
	if port == "" {
		port = "443"
	}

	host = uri.Hostname() + ":" + port
	value := map[string]string{"m.server": host}
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, value)
	}
}

// /_matrix/federation/v1/version
func matrixFederationVersion() echo.HandlerFunc {
	value := map[string]map[string]string{
		"server": {
			"name":    version.Name,
			"version": version.Version,
		},
	}
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, value)
	}
}
