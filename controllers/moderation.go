package controllers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/utils"
)

type moderationService interface {
	Report(string, string) error
	List(...string) ([]string, error)
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
			utils.Logger.Error().Err(err).Msg("cannot bind report")
			return err
		}

		if err := svc.Report(report.RoomID, report.Reason); err != nil {
			utils.Logger.Error().Err(err).Msg("cannot report room")
			return err
		}

		return c.NoContent(http.StatusAccepted)
	}
}

func listBanned(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if strings.Contains(c.Request().UserAgent(), "bot") {
			return c.NoContent(http.StatusForbidden)
		}

		serverName := c.Param("server_name")
		var list []string
		var err error
		if serverName != "" {
			list, err = svc.List(serverName)
		} else {
			list, err = svc.List()
		}
		if err != nil {
			return err
		}

		if len(list) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusOK, list)
	}
}

func ban(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if strings.Contains(c.Request().UserAgent(), "bot") {
			return c.NoContent(http.StatusForbidden)
		}

		roomID := c.Param("room_id")
		if err := svc.Ban(roomID); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "the room has been banned"})
	}
}

func unban(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if strings.Contains(c.Request().UserAgent(), "bot") {
			return c.NoContent(http.StatusForbidden)
		}

		roomID := c.Param("room_id")
		if err := svc.Unban(roomID); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "the room has been unbanned"})
	}
}
