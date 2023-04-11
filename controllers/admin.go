package controllers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"gitlab.com/etke.cc/int/mrs/model"
	"gopkg.in/yaml.v3"
)

type matrixService interface {
	DiscoverServers(int)
	ParseRooms(int, func(string, string, model.MatrixRoom)) error
	EachRoom(func(string, model.MatrixRoom))
	EachServer(func(string, string))
}

type indexService interface {
	Index(string, model.Entry) error
}

func servers(matrix matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		servers := make([]string, 0)
		matrix.EachServer(func(name string, _ string) {
			servers = append(servers, name)
		})
		serversb, err := yaml.Marshal(servers)
		if err != nil {
			return err
		}
		return c.Blob(http.StatusOK, "application/x-yaml", serversb)
	}

}

func discover(matrix matrixService, stats statsService, workers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go func(matrix matrixService, stats statsService) {
			matrix.DiscoverServers(workers)
			stats.Collect()
		}(matrix, stats)

		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func parse(matrix matrixService, index indexService, stats statsService, workers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go func(matrix matrixService, index indexService, stats statsService) {
			matrix.ParseRooms(workers, func(serverName, roomID string, room model.MatrixRoom) {
				if err := index.Index(roomID, room.Entry(serverName)); err != nil {
					log.Println(room.Alias, "cannot index", err)
				}
			})
			stats.Collect()
		}(matrix, index, stats)

		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func reindex(matrix matrixService, index indexService, stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go func(matrix matrixService, index indexService, stats statsService) {
			matrix.EachRoom(func(roomID string, room model.MatrixRoom) {
				if err := index.Index(roomID, room.Entry("")); err != nil {
					log.Println(room.Alias, "cannot index", err)
				}
				stats.Collect()
			})
		}(matrix, index, stats)

		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func full(matrix matrixService, index indexService, stats statsService, discoveryWorkers int, parsingWorkers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go func(matrix matrixService, index indexService, stats statsService, discoveryWorkers int, parsingWorkers int) {
			matrix.DiscoverServers(discoveryWorkers)
			matrix.ParseRooms(parsingWorkers, func(serverName, roomID string, room model.MatrixRoom) {
				if err := index.Index(roomID, room.Entry(serverName)); err != nil {
					log.Println(room.Alias, "cannot index", err)
				}
			})
			stats.Collect()
		}(matrix, index, stats, discoveryWorkers, parsingWorkers)

		return c.NoContent(http.StatusCreated)
	}
}
