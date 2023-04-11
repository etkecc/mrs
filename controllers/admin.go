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

func servers(svc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		servers := make([]string, 0)
		svc.EachServer(func(name string, _ string) {
			servers = append(servers, name)
		})
		serversb, err := yaml.Marshal(servers)
		if err != nil {
			return err
		}
		return c.Blob(http.StatusOK, "application/x-yaml", serversb)
	}

}

func discover(svc matrixService, workers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go svc.DiscoverServers(workers)
		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func parse(parseSvc matrixService, indexSvc indexService, workers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go parseSvc.ParseRooms(workers, func(serverName, roomID string, room model.MatrixRoom) {
			if err := indexSvc.Index(roomID, room.Entry(serverName)); err != nil {
				log.Println(room.Alias, "cannot index", err)
			}
		})

		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func reindex(matrixSvc matrixService, indexSvc indexService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go matrixSvc.EachRoom(func(roomID string, room model.MatrixRoom) {
			if err := indexSvc.Index(roomID, room.Entry("")); err != nil {
				log.Println(room.Alias, "cannot index", err)
			}
		})

		return c.NoContent(http.StatusCreated)
	}
}

//nolint:errcheck
func full(matrixSvc matrixService, indexSvc indexService, discoveryWorkers int, parsingWorkers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		go func(matrixSvc matrixService, indexSvc indexService, discoveryWorkers int, parsingWorkers int) {
			matrixSvc.DiscoverServers(discoveryWorkers)
			matrixSvc.ParseRooms(parsingWorkers, func(serverName, roomID string, room model.MatrixRoom) {
				if err := indexSvc.Index(roomID, room.Entry(serverName)); err != nil {
					log.Println(room.Alias, "cannot index", err)
				}
			})

		}(matrixSvc, indexSvc, discoveryWorkers, parsingWorkers)

		return c.NoContent(http.StatusCreated)
	}
}
