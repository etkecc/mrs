package controllers

import (
	"context"
	"net/http"
	"strings"

	"github.com/etkecc/go-apm"
	"github.com/labstack/echo/v4"
)

type moderationService interface {
	Report(context.Context, string, string, bool) error
	List(context.Context, ...string) ([]string, error)
	Ban(context.Context, string) error
	Unban(context.Context, string) error
}

type reportSubmission struct {
	RoomID    string `param:"room_id"`
	Reason    string `json:"reason"`
	NoMSC1929 bool   `json:"no_msc1929"`
}

func report(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := apm.Log(c.Request().Context())
		var report reportSubmission
		if err := c.Bind(&report); err != nil {
			log.Error().Err(err).Msg("cannot bind report")
			return err
		}

		if err := svc.Report(c.Request().Context(), report.RoomID, report.Reason, report.NoMSC1929); err != nil {
			log.Error().Err(err).Msg("cannot report room")
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
			list, err = svc.List(c.Request().Context(), serverName)
		} else {
			list, err = svc.List(c.Request().Context())
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
		if err := svc.Ban(c.Request().Context(), roomID); err != nil {
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
		if err := svc.Unban(c.Request().Context(), roomID); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "the room has been unbanned"})
	}
}
