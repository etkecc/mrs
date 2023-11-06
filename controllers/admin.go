package controllers

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

type dataService interface {
	AddServer(string) int
	AddServers([]string, int)
	DiscoverServers(int)
	ParseRooms(int)
	Ingest()
	Full(int, int)
}

type crawlerService interface {
	OnlineServers() map[string]string
	GetAvatar(string, string) (io.Reader, string)
}

func servers(crawler crawlerService) echo.HandlerFunc {
	return func(c echo.Context) error {
		srvmap := crawler.OnlineServers()
		servers := make([]string, 0)
		for name := range srvmap {
			servers = append(servers, name)
		}
		serversb, err := yaml.Marshal(servers)
		if err != nil {
			return err
		}
		return c.Blob(http.StatusOK, "application/x-yaml", serversb)
	}
}

func status(stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, stats.Get())
	}
}

func discover(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go data.DiscoverServers(cfg.Get().Workers.Discovery)
		return c.NoContent(http.StatusCreated)
	}
}

func parse(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go data.ParseRooms(cfg.Get().Workers.Parsing)
		return c.NoContent(http.StatusCreated)
	}
}

func reindex(data dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go data.Ingest()
		return c.NoContent(http.StatusCreated)
	}
}

func full(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go data.Full(cfg.Get().Workers.Discovery, cfg.Get().Workers.Parsing)
		return c.NoContent(http.StatusCreated)
	}
}
