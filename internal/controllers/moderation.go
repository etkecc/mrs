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

// @Summary		Report a room
// @Description	Report a room for moderation. Open on purpose, no auth: anyone can flag a room, that is the whole point. The JSON body carries `reason` (5 chars minimum, or you get a 400) and an optional `no_msc1929` flag. This is the one honest, no-auth POST in the moderation set.
// @Tags			moderation
// @Accept			json
// @Produce		json
// @Param			room_id	path	string				true	"Room ID being reported"
// @Param			request	body	reportSubmission	true	"Report reason (5 chars minimum) and options"
// @Success		202		"Report accepted"
// @Failure		400		{object}	model.MatrixError	"Invalid room ID, or reason too short or missing"
// @Failure		500		{object}	model.MatrixError	"Internal error while processing the report"
// @Router			/mod/report/{room_id} [post]
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

// @Summary		Clear a room's reports
// @Description	Clears reports on a room. Two warts, both owned: it is a state-changing GET, and a User-Agent containing "bot" gets a 403 even with valid moderation credentials, a scar from crawlers tripping these endpoints, not a feature. There is also a no-room-id form (GET /mod/unreport) that hits the same handler with an empty ID.
// @Tags			moderation
// @Produce		json
// @Security		ModerationAuth
// @Param			room_id	path	string	true	"Room ID to clear reports for"
// @Success		204		"Reports cleared"
// @Failure		403		"User-Agent contains 'bot'"
// @Router			/mod/unreport/{room_id} [get]
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

// @Summary		List banned rooms
// @Description	Lists banned room IDs. Append /{server_name} to filter to a single server. An empty list is a 204, not an empty 200. As with the rest of the mod group, a "bot" User-Agent gets a 403 even authenticated.
// @Tags			moderation
// @Produce		json
// @Security		ModerationAuth
// @Success		200	{array}	string	"Banned room IDs"
// @Success		204	"No banned rooms"
// @Failure		403	"User-Agent contains 'bot'"
// @Router			/mod/list [get]
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

// @Summary		List reported rooms
// @Description	Lists reported rooms as a room-ID to reason map. Append /{server_name} to filter to a single server. Empty is a 204. A "bot" User-Agent gets a 403 even authenticated.
// @Tags			moderation
// @Produce		json
// @Security		ModerationAuth
// @Success		200	{object}	map[string]string	"Room ID to report reason"
// @Success		204	"No reported rooms"
// @Failure		403	"User-Agent contains 'bot'"
// @Router			/mod/list-reported [get]
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

// @Summary		Ban a room
// @Description	Bans a room from the index. Yes, it is a GET that mutates state, we know, and no, we are not proud of it. And a User-Agent containing "bot" gets a 403 even with valid credentials, a scar from crawlers tripping this, not a feature. Both are real behavior, documented on purpose so they surprise you here and not in production.
// @Tags			moderation
// @Produce		json
// @Security		ModerationAuth
// @Param			room_id	path		string				true	"Room ID to ban"
// @Success		200		{object}	map[string]string	"Confirmation message"
// @Failure		403		"User-Agent contains 'bot'"
// @Router			/mod/ban/{room_id} [get]
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

// @Summary		Unban a room
// @Description	Lifts a room ban. Same shape as ban: a state-changing GET, and a "bot" User-Agent gets a 403 even authenticated. Real behavior, owned.
// @Tags			moderation
// @Produce		json
// @Security		ModerationAuth
// @Param			room_id	path		string				true	"Room ID to unban"
// @Success		200		{object}	map[string]string	"Confirmation message"
// @Failure		403		"User-Agent contains 'bot'"
// @Router			/mod/unban/{room_id} [get]
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
