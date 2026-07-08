package controllers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/etkecc/go-apm"
	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
)

func configureMatrixS2SEndpoints(e *echo.Echo, matrixSvc matrixService, plausible plausibleService, cacheSvc cacheService) {
	e.GET("/.well-known/matrix/server", wellKnownServer(matrixSvc), cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/federation/v1/version", serverVersion(matrixSvc), cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/key/v2/server", keyServer(matrixSvc))
	e.GET("/_matrix/key/v2/query/:serverName", queryServerKeys(matrixSvc, plausible))
	e.POST("/_matrix/key/v2/query", queryServersKeys(matrixSvc, plausible))
	e.GET("/_matrix/federation/v1/query/directory", queryDirectory(matrixSvc))
	e.GET("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc), cacheSvc.MiddlewareSearch())
	e.POST("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc), cacheSvc.MiddlewareSearch())
}

// @Summary		Server well-known
// @Description	Our federation delegation, one line: the `m.server` record telling other homeservers "do not knock here, our federation API answers over there." One redirect the whole federation agrees to honor.
// @Tags			matrix-s2s
// @Produce		json
// @Success		200	{object}	model.WellKnownServer	"Where our federation API actually answers"
// @Router			/.well-known/matrix/server [get]
func wellKnownServer(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetServerWellKnown())
	}
}

// @Summary		Server version
// @Description	Our federation server name and build. Static, cached, boring in the good way. The one endpoint here that has never once caused an incident, and we would like to keep it that way.
// @Tags			matrix-s2s
// @Produce		json
// @Success		200	{object}	model.ServerVersion	"Server software name and version"
// @Router			/_matrix/federation/v1/version [get]
func serverVersion(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetServerVersion())
	}
}

// @Summary		Our signing keys
// @Description	Our own published ed25519 signing keys, signed by us, so federation can check that the requests we send are actually ours. It is a naive notary. verify_keys is a map because the spec key set is dynamic, keyed by key ID.
// @Tags			matrix-s2s
// @Produce		json
// @Success		200	{object}	model.ServerKeys	"Our signed public signing keys"
// @Router			/_matrix/key/v2/server [get]
func keyServer(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetKeyServer(c.Request().Context()))
	}
}

// @Summary		Query one server's keys
// @Description	Notary lookup of one remote server's signing keys, handed back verbatim as the server signed them (we do not re-wrap or re-sign). A server we cannot reach or verify yields an empty key set, not a 5xx tantrum.
// @Tags			matrix-s2s
// @Produce		json
// @Param			serverName				path		string							true	"Server name to look up keys for"
// @Param			minimum_valid_until_ts	query		int								false	"Only return keys valid until at least this timestamp (ms). A malformed value is treated as 0."
// @Success		200						{object}	model.ServerKeysQueryResponse	"Signed server keys, or an empty key set"
// @Router			/_matrix/key/v2/query/{serverName} [get]
func queryServerKeys(matrixSvc matrixService, plausible plausibleService) echo.HandlerFunc {
	return func(c echo.Context) error {
		serverName := c.Param("serverName")
		if serverName == "" {
			return c.JSONBlob(http.StatusOK, []byte(model.EmptyServerKeysResp))
		}

		validUntilTStr := c.QueryParam("minimum_valid_until_ts")
		var validUntilTS int64
		if validUntilTStr != "" {
			validUntilTS, _ = strconv.ParseInt(validUntilTStr, 10, 64) //nolint:errcheck // 0 is handled properly
		}

		evt := model.NewAnalyticsEvent(c.Request().Context(), "Get Key", map[string]string{"server": serverName}, c.Request())
		go func(ctx context.Context, evt *model.AnalyticsEvent) {
			ctx = context.WithoutCancel(ctx)
			plausible.Track(ctx, evt)
		}(c.Request().Context(), evt)

		return c.JSONBlob(http.StatusOK, matrixSvc.QueryServerKeys(c.Request().Context(), serverName, validUntilTS))
	}
}

// @Summary		Batch query server keys
// @Description	Notary lookup for a batch of servers, each returned verbatim as its origin signed it. Malformed JSON gets a 400. An empty batch (valid body, no server_keys) gets a 200 with an empty key set: you asked for nothing, you got nothing.
// @Tags			matrix-s2s
// @Accept			json
// @Produce		json
// @Param			request					body		model.QueryServerKeysRequest	true	"Servers and key IDs to look up"
// @Param			minimum_valid_until_ts	query		int								false	"Only return keys valid until at least this timestamp (ms). A malformed value is treated as 0."
// @Success		200						{object}	model.ServerKeysQueryResponse	"Signed server keys, or an empty key set for an empty batch"
// @Failure		400						{object}	model.MatrixError				"Request body is not valid JSON"
// @Router			/_matrix/key/v2/query [post]
func queryServersKeys(matrixSvc matrixService, plausible plausibleService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req *model.QueryServerKeysRequest
		if err := c.Bind(&req); err != nil {
			apm.Log(c.Request().Context()).Warn().Err(err).Msg("failed to bind query server keys request")
			return c.JSON(http.StatusBadRequest, &model.MatrixError{Code: "M_NOT_JSON", Message: "request body is not valid JSON"})
		}
		if req == nil {
			return c.JSON(http.StatusBadRequest, &model.MatrixError{Code: "M_NOT_JSON", Message: "request body is required"})
		}

		validUntilTStr := c.QueryParam("minimum_valid_until_ts")
		var validUntilTS int64
		if validUntilTStr != "" {
			validUntilTS, _ = strconv.ParseInt(validUntilTStr, 10, 64) //nolint:errcheck // 0 is handled properly
		}

		if len(req.ServerKeys) == 0 {
			return c.JSONBlob(http.StatusOK, []byte(model.EmptyServerKeysResp))
		}

		for srv := range req.ServerKeys {
			if srv == "" {
				continue
			}
			evt := model.NewAnalyticsEvent(c.Request().Context(), "Get Key", map[string]string{"server": srv}, c.Request())
			go func(ctx context.Context, evt *model.AnalyticsEvent) {
				ctx = context.WithoutCancel(ctx)
				plausible.Track(ctx, evt)
			}(c.Request().Context(), evt)
		}

		return c.JSONBlob(http.StatusOK, matrixSvc.QueryServersKeys(c.Request().Context(), req, validUntilTS))
	}
}

// @Summary		Resolve a room alias
// @Description	Resolves a room alias to a room ID and the servers that can help you join. This is an authenticated federation endpoint: we check the incoming X-Matrix signature first, so a missing or bad one is a 401 before any resolution happens. Past that we pass back whatever status the lookup produced, so a 404 means we genuinely could not resolve it, not that the endpoint is missing.
// @Tags			matrix-s2s
// @Produce		json
// @Param			room_alias	query		string							true	"Room alias to resolve, e.g. #room:server"
// @Success		200			{object}	model.QueryDirectoryResponse	"Room ID and the servers that can help you join"
// @Failure		401			{object}	model.MatrixError				"Federation auth failed (missing or invalid X-Matrix signature)"
// @Failure		404			{object}	model.MatrixError				"Alias could not be resolved"
// @Security		FederationAuth
// @Router			/_matrix/federation/v1/query/directory [get]
func queryDirectory(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.QueryDirectory(c.Request().Context(), c.Request(), c.QueryParam("room_alias")))
	}
}

// @Summary		Public rooms directory
// @Description	Our slice of the federation public-rooms directory: the rooms we have crawled and indexed. Authenticated federation endpoint, so a missing or invalid X-Matrix signature is a 401. GET (query params) and POST (JSON filter body) share one handler, and a malformed POST body is logged then ignored, so you get the unfiltered listing rather than an error.
// @Tags			matrix-s2s
// @Accept			json
// @Produce		json
// @Param			request	body		model.RoomDirectoryRequest	false	"Filter and pagination, POST only"
// @Success		200		{object}	model.RoomDirectoryResponse	"Our indexed slice of the public-rooms directory"
// @Failure		401		{object}	model.MatrixError			"Federation auth failed (missing or invalid X-Matrix signature)"
// @Security		FederationAuth
// @Router			/_matrix/federation/v1/publicRooms [get]
// @Router			/_matrix/federation/v1/publicRooms [post]
func matrixRoomDirectory(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := apm.Log(c.Request().Context())
		r := c.Request()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		c.SetRequest(r)

		var req model.RoomDirectoryRequest
		if err := c.Bind(&req); err != nil {
			log.Error().Err(err).Msg("POST directory request binding failed")
		}
		req.IP = c.RealIP()
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		c.SetRequest(r)

		return c.JSONBlob(matrixSvc.PublicRooms(c.Request().Context(), c.Request(), &req))
	}
}
