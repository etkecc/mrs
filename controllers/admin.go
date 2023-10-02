package controllers

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

type dataService interface {
	DiscoverServers(int)
	ParseRooms(int)
	Ingest()
	Full(int, int)
}

type matrixService interface {
	AddServer(string) int
	AddServers([]string, int)
	AllServers() map[string]string
	GetAvatar(string, string) (io.Reader, string)
}

func servers(matrix matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		srvmap := matrix.AllServers()
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

func discover(data dataService, workers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go data.DiscoverServers(workers)
		return c.NoContent(http.StatusCreated)
	}
}

func parse(data dataService, workers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go data.ParseRooms(workers)
		return c.NoContent(http.StatusCreated)
	}
}

func reindex(data dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go data.Ingest()
		return c.NoContent(http.StatusCreated)
	}
}

func full(data dataService, discoveryWorkers int, parsingWorkers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go data.Full(discoveryWorkers, parsingWorkers)
		return c.NoContent(http.StatusCreated)
	}
}
