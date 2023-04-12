package controllers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"

	"gitlab.com/etke.cc/mrs/api/model"
)

type matrixService interface {
	DiscoverServers(int)
	ParseRooms(int, func(string, string, *model.MatrixRoom)) error
	EachRoom(func(string, *model.MatrixRoom))
	EachServer(func(string, string))
}

type indexService interface {
	Index(string, *model.Entry) error
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
			log.Println("discovering matrix servers...")
			matrix.DiscoverServers(workers)
			log.Println("servers discovery has been finished")

			log.Println("collecting stats...")
			stats.Collect()
			log.Println("stats have been collected")
		}(matrix, stats)

		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func parse(matrix matrixService, index indexService, stats statsService, workers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go func(matrix matrixService, index indexService, stats statsService) {
			log.Println("parsing matrix rooms...")
			matrix.ParseRooms(workers, func(serverName, roomID string, room *model.MatrixRoom) {
				if err := index.Index(roomID, room.Entry(serverName)); err != nil {
					log.Println(room.Alias, "cannot index", err)
				}
			})
			log.Println("all available matrix rooms have been parsed")

			log.Println("collecting stats...")
			stats.Collect()
			log.Println("stats have been collected")
		}(matrix, index, stats)

		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func reindex(matrix matrixService, index indexService, stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go func(matrix matrixService, index indexService, stats statsService) {
			log.Println("ingesting matrix rooms...")
			matrix.EachRoom(func(roomID string, room *model.MatrixRoom) {
				if err := index.Index(roomID, room.Entry("")); err != nil {
					log.Println(room.Alias, "cannot index", err)
				}
				log.Println("all available matrix rooms have been ingested")

				log.Println("collecting stats...")
				stats.Collect()
				log.Println("stats have been collected")
			})
		}(matrix, index, stats)

		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func full(matrix matrixService, index indexService, stats statsService, discoveryWorkers int, parsingWorkers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go func(matrix matrixService, index indexService, stats statsService, discoveryWorkers int, parsingWorkers int) {
			log.Println("discovering matrix servers...")
			matrix.DiscoverServers(discoveryWorkers)
			log.Println("servers discovery has been finished")

			log.Println("parsing matrix rooms...")
			matrix.ParseRooms(parsingWorkers, func(serverName, roomID string, room *model.MatrixRoom) {
				if err := index.Index(roomID, room.Entry(serverName)); err != nil {
					log.Println(room.Alias, "cannot index", err)
				}
			})
			log.Println("all available matrix rooms have been parsed")

			log.Println("collecting stats...")
			stats.Collect()
			log.Println("stats have been collected")
		}(matrix, index, stats, discoveryWorkers, parsingWorkers)

		return c.NoContent(http.StatusCreated)
	}
}
