package controllers

import (
	"context"
	"io"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
)

// @Summary		Add a server
// @Description	Queues a single server for discovery. Auth is optional: send Discovery credentials to skip the rate limit, or call anonymously and take the limit. We dial the server before answering, so the status code tells you what actually happened: 201 if it is reachable and freshly added, 208 if we already knew it, 422 if it would not answer (we record it offline rather than re-dial it on every retry).
// @Tags			discovery
// @Produce		json
// @Param			name	path	string	true	"Server name to add"
// @Success		201		"Server was reachable and queued"
// @Success		208		"Server already known, nothing to do"
// @Failure		401		"Invalid Discovery credentials (only if you send them)"
// @Failure		422		"Server could not be reached, nothing added"
// @Failure		429		"Rate limited (anonymous callers)"
// @Router			/discover/{name} [post]
func addServer(dataSvc dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := dataSvc.AddServer(c.Request().Context(), c.Param("name"))
		return c.NoContent(code)
	}
}

// @Summary		Add servers in bulk
// @Description	Queues a batch of servers for discovery. Requires Discovery credentials. Fire-and-forget: we take the JSON array, return 202 immediately, and discover them in the background.
// @Tags			discovery
// @Accept			json
// @Produce		json
// @Security		DiscoveryAuth
// @Param			request	body	[]string	true	"Server names to add"
// @Success		202		"Batch accepted for background discovery"
// @Failure		400		{object}	model.MatrixError	"Malformed request body (not a JSON array of server names)"
// @Failure		401		"Invalid Discovery credentials"
// @Router			/discover/bulk [post]
func addServers(dataSvc dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer c.Request().Body.Close()
		jsonb, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusBadRequest, &model.MatrixError{Code: "M_NOT_JSON", Message: "cannot read request body"})
		}
		var servers []string
		if err := json.Unmarshal(jsonb, &servers); err != nil {
			return c.JSON(http.StatusBadRequest, &model.MatrixError{Code: "M_NOT_JSON", Message: "request body must be a JSON array of server names"})
		}

		ctx := context.WithoutCancel(c.Request().Context())
		go dataSvc.AddServers(ctx, servers, cfg.Get().Workers.Discovery)
		return c.NoContent(http.StatusAccepted)
	}
}
