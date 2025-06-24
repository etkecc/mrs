package controllers

import (
	"context"
	"net/http"
	"strings"

	"github.com/etkecc/go-apm"
	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

type moderationService interface {
	Report(context.Context, string, string, string, bool) error
	List(context.Context, ...string) ([]string, error)
	ListReported(context.Context, ...string) (map[string]string, error)
	Ban(context.Context, string) error
	Unban(context.Context, string) error
	Unreport(context.Context, string) error
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

		if !utils.IsValidID(report.RoomID) {
			return c.JSON(http.StatusBadRequest, &model.MatrixError{
				Code:    "M_INVALID_PARAM",
				Message: "Invalid room ID format.",
			})
		}

		if report.Reason == "" || len(report.Reason) < 5 {
			return c.JSON(http.StatusBadRequest, &model.MatrixError{
				Code:    "M_INVALID_PARAM",
				Message: "Report reason is too short or missing.",
			})
		}

		if err := svc.Report(c.Request().Context(), c.RealIP(), report.RoomID, report.Reason, report.NoMSC1929); err != nil {
			log.Error().Err(err).Msg("cannot report room")
			return c.JSON(http.StatusInternalServerError, &model.MatrixError{
				Code:    "M_INTERNAL_ERROR",
				Message: "An internal error occurred while processing your request.",
			})
		}

		return c.NoContent(http.StatusAccepted)
	}
}

func unreport(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if strings.Contains(c.Request().UserAgent(), "bot") {
			return c.NoContent(http.StatusForbidden)
		}

		roomID := c.Param("room_id")
		if err := svc.Unreport(c.Request().Context(), roomID); err != nil {
			return err
		}

		return c.NoContent(http.StatusNoContent)
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

func listReported(svc moderationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if strings.Contains(c.Request().UserAgent(), "bot") {
			return c.NoContent(http.StatusForbidden)
		}

		serverName := c.Param("server_name")
		var list map[string]string
		var err error
		if serverName != "" {
			list, err = svc.ListReported(c.Request().Context(), serverName)
		} else {
			list, err = svc.ListReported(c.Request().Context())
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
