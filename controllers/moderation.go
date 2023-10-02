package controllers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

type moderationService interface {
	Report(string, string) error
	List() ([]string, error)
	Ban(string) error
	Unban(string) error
}

type reportSubmission struct {
	RoomID string `param:"room_id"`
	Reason string `json:"reason"`
}

func report(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var report reportSubmission
		if err := c.Bind(&report); err != nil {
			log.Println("report", "bind error:", err)
			return err
		}

		if err := svc.Report(report.RoomID, report.Reason); err != nil {
			log.Println("report", err)
			return err
		}

		return c.NoContent(http.StatusAccepted)
	}
}

func listBanned(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		list, err := svc.List()
		if err != nil {
			return err
		}

		if len(list) == 0 {
			return c.NoContent(http.StatusCreated)
		}
		return c.JSON(http.StatusOK, list)
	}
}

func ban(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		roomID := c.Param("room_id")
		if err := svc.Ban(roomID); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "the room has been banned"})
	}
}

func unban(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		roomID := c.Param("room_id")
		if err := svc.Unban(roomID); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "the room has been unbanned"})
	}
}
