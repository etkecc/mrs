package controllers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"gitlab.com/etke.cc/int/mrs/model"
)

type matrixService interface {
	DiscoverServers(int)
	ParseRooms(int, func(string, model.MatrixRoom)) error
	EachRoom(func(string, model.MatrixRoom))
}

type indexService interface {
	Index(string, model.Entry) error
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
		go parseSvc.ParseRooms(workers, func(roomID string, room model.MatrixRoom) {
			if err := indexSvc.Index(roomID, model.Entry(room)); err != nil {
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
			if err := indexSvc.Index(roomID, model.Entry(room)); err != nil {
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
			matrixSvc.ParseRooms(parsingWorkers, func(roomID string, room model.MatrixRoom) {
				if err := indexSvc.Index(roomID, model.Entry(room)); err != nil {
					log.Println(room.Alias, "cannot index", err)
				}
			})

		}(matrixSvc, indexSvc, discoveryWorkers, parsingWorkers)

		return c.NoContent(http.StatusCreated)
	}
}
