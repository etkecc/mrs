package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

func configureMatrixCSEndpoints(e *echo.Echo, matrixSvc matrixService, cacheSvc cacheService) {
	rl := getRL(30)
	e.GET("/.well-known/matrix/client", wellKnownClient(matrixSvc), cacheSvc.MiddlewareImmutable())
	e.GET("/.well-known/matrix/support", wellKnownSupport(matrixSvc), cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/client/versions", clientVersions(matrixSvc), cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/media/r0/thumbnail/:name/:id", avatar(matrixSvc), rl, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/media/v3/thumbnail/:name/:id", avatar(matrixSvc), rl, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/client/r0/directory/room/:room_alias", clientDirectoryRoom(matrixSvc), rl)
	e.GET("/_matrix/client/v3/directory/room/:room_alias", clientDirectoryRoom(matrixSvc), rl)
	// uncached on purpose: visibility is ban/index-state-dependent and there is no CDN purge, so an edge-cached "public" would outlive a ban.
	e.GET("/_matrix/client/r0/directory/list/room/:room_id", clientDirectoryList(matrixSvc), rl)
	e.GET("/_matrix/client/v3/directory/list/room/:room_id", clientDirectoryList(matrixSvc), rl)

	// one handler, three routes: the stable room-summary path plus the two matrix.to-compatible unstable paths.
	summary := clientRoomSummary(matrixSvc)
	e.GET("/_matrix/client/v1/room_summary/:room_id_alias", summary, rl)
	e.GET("/_matrix/client/unstable/im.nheko.summary/summary/:room_id_alias", summary, rl)
	e.GET("_matrix/client/unstable/im.nheko.summary/rooms/:room_id_alias/summary", summary, rl)
}

// @Summary		Client well-known
// @Description	The client well-known: the breadcrumb a Matrix client follows to find our homeserver base URL, because SRV records never quite caught on with clients. We only publish m.homeserver; no identity server, we are not that kind of server.
// @Tags			matrix-cs
// @Produce		json
// @Success		200	{object}	model.WellKnownClient	"Where a client should point its homeserver base URL"
// @Router			/.well-known/matrix/client [get]
func wellKnownClient(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetClientWellKnown())
	}
}

// @Summary		Support contacts
// @Description	The MSC1929 support well-known: who to yell at when this server misbehaves, admin and support contacts plus an optional support page. Straight from config, so if it is empty, someone forgot to fill it in.
// @Tags			matrix-cs
// @Produce		json
// @Success		200	{object}	msc1929.Response	"Admin and support contacts"
// @Router			/.well-known/matrix/support [get]
func wellKnownSupport(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetSupportWellKnown())
	}
}

// @Summary		Client-server versions
// @Description	The client-server spec versions and unstable features we claim to speak. Mostly polite fiction: a hardcoded list of versions we do not really implement, kept only because matrix.to and older clients refuse to talk to anything that will not recite them. We are a search index wearing a homeserver's coat for a handful of read endpoints, not a real homeserver.
// @Tags			matrix-cs
// @Produce		json
// @Success		200	{object}	model.ClientVersions	"The spec versions we pretend to support so clients stop sulking"
// @Router			/_matrix/client/versions [get]
func clientVersions(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetClientVersion())
	}
}

// @Summary		Resolve a room alias (client)
// @Description	Turns a human-friendly room alias like #coffee:example.com into the actual room ID plus a list of servers that might know where it lives, because nobody memorizes a !opaque:id and clients still have to route the join somewhere. The r0 and v3 paths are the same handler; r0 is the legacy alias we keep alive for clients that never upgraded.
// @Tags			matrix-cs
// @Produce		json
// @Param			room_alias	path		string							true	"Room alias to resolve, e.g. #room:server"
// @Success		200			{object}	model.QueryDirectoryResponse	"Room ID and the servers that can help you join"
// @Failure		404			{object}	model.MatrixError				"We could not resolve that alias"
// @Router			/_matrix/client/r0/directory/room/{room_alias} [get]
// @Router			/_matrix/client/v3/directory/room/{room_alias} [get]
func clientDirectoryRoom(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientDirectory(c.Request().Context(), c.Param("room_alias")))
	}
}

// @Summary		Room visibility in the directory
// @Description	Whether a room is listed in the public directory. MRS only indexes public rooms, so if we hold the room the answer is "public"; if we have never crawled it (or it is banned) you get a 404, same as any homeserver would. This is a search engine, not a speakeasy: there is no "private" to report, only "we have it" or "we don't". r0 and v3 share one handler; r0 is the legacy alias.
// @Tags			matrix-cs
// @Produce		json
// @Param			room_id	path		string					true	"Room ID to check"
// @Success		200		{object}	model.RoomVisibility	"The room is indexed, and therefore public"
// @Failure		404		{object}	model.MatrixError		"We have never crawled that room"
// @Router			/_matrix/client/r0/directory/list/room/{room_id} [get]
// @Router			/_matrix/client/v3/directory/list/room/{room_id} [get]
func clientDirectoryList(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomVisibility(c.Request().Context(), c.Param("room_id")))
	}
}

// @Summary		Room summary
// @Description	Room summary (name, topic, member count, join rules) for a room ID or alias, per MSC3266. We answer this for rooms we have crawled, so it works even though we are a search index and not the room's homeserver. Registered at the stable v1 path plus the two unstable im.nheko.summary paths, because matrix.to and older clients still call those.
// @Tags			matrix-cs
// @Produce		json
// @Param			room_id_alias	path		string					true	"Room ID or alias to summarize"
// @Param			via				query		string					false	"A server to try for the summary if we do not hold the room locally. Only the first value is used."
// @Success		200				{object}	model.RoomDirectoryRoom	"Room summary"
// @Failure		404				{object}	model.MatrixError		"Room not found"
// @Router			/_matrix/client/v1/room_summary/{room_id_alias} [get]
// @Router			/_matrix/client/unstable/im.nheko.summary/summary/{room_id_alias} [get]
// @Router			/_matrix/client/unstable/im.nheko.summary/rooms/{room_id_alias}/summary [get]
func clientRoomSummary(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		code, room := matrixSvc.GetClientRoomSummary(c.Request().Context(), c.Param("room_id_alias"), c.QueryParam("via"), false)
		if code != http.StatusOK {
			return c.JSONBlob(code, utils.MustJSON(model.MatrixError{
				Code:    "M_NOT_FOUND",
				Message: "room not found",
			}))
		}
		return c.JSONBlob(code, utils.MustJSON(room))
	}
}
