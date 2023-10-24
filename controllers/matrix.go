package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/matrix-org/gomatrixserverlib"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/version"
)

type unsignedKeyResp struct {
	ServerName    string                       `json:"server_name"`
	ValidUntilTS  int64                        `json:"valid_until_ts"`
	VerifyKeys    map[string]map[string]string `json:"verify_keys"`
	OldVerifyKeys map[string]any               `json:"old_verify_keys"`
}

type roomDirectoryReq struct {
	Filter               roomDirectoryReqFilter `json:"filter"`
	IncludeAllNetworks   bool                   `query:"include_all_networks" json:"include_all_networks"`
	Limit                int                    `query:"limit" json:"limit"`
	Since                string                 `query:"since" json:"since"`
	ThirdPartyInstanceID string                 `query:"third_party_instance_id" json:"third_party_instance_id"`
}

type roomDirectoryReqFilter struct {
	GenericSearchTerm string `json:"generic_search_term"`
	RoomTypes         string `json:"room_types,omitempty"`
}

type roomDirectoryResp struct {
	Chunk     []roomDirectoryItem `json:"chunk"`
	NextBatch string              `json:"next_batch"`
	PrevBatch string              `json:"prev_batch"`
	Total     int                 `json:"total_room_count_estimate"`
}

type roomDirectoryItem struct {
	Avatar        string `json:"avatar_url"`
	Alias         string `json:"canonical_alias"`
	Guest         bool   `json:"guest_can_join"`
	Name          string `json:"name"`
	Members       int    `json:"num_joined_members"`
	ID            string `json:"room_id"`
	Topic         string `json:"topic"`
	WorldReadable bool   `json:"world_readable"`
}

// /.well-known/matrix/server
func wellKnownServer(host string) echo.HandlerFunc {
	uri, err := url.Parse(host)
	if err != nil {
		log.Println("ERROR: cannot parse public api host to use in /.well-known/matrix/server:", err)
	}
	port := uri.Port()
	if port == "" {
		port = "443"
	}

	host = uri.Hostname() + ":" + port
	value := map[string]string{"m.server": host}
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, value)
	}
}

// /_matrix/federation/v1/version
func matrixFederationVersion() echo.HandlerFunc {
	value := map[string]map[string]string{
		"server": {
			"name":    version.Name,
			"version": version.Version,
		},
	}
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, value)
	}
}

// /_matrix/key/v2/server
func matrixKeyServer(matrix *config.Matrix) echo.HandlerFunc {
	keys, err := model.KeysFrom(matrix.Keys)
	if err != nil {
		log.Println("ERROR: cannot parse key from string:", err)
	}

	resp := unsignedKeyResp{
		ServerName:    matrix.ServerName,
		ValidUntilTS:  time.Now().UTC().Add(24*time.Hour - 1*time.Second).UnixMilli(),
		VerifyKeys:    map[string]map[string]string{},
		OldVerifyKeys: map[string]any{},
	}
	for _, key := range keys {
		resp.VerifyKeys[key.ID] = map[string]string{"key": key.Public}
	}

	payload, err := json.Marshal(&resp)
	if err != nil {
		log.Println("ERROR: cannot marshal matrix server key payload:", err)
	}
	for _, key := range keys {
		payload, err = gomatrixserverlib.SignJSON(matrix.ServerName, gomatrixserverlib.KeyID(key.ID), key.Private, payload)
		if err != nil {
			log.Println("ERROR: cannot sign payload:", err)
		}
	}
	return func(c echo.Context) error { return c.JSONBlob(http.StatusOK, payload) }
}

// /_matrix/federation/v1/publicRooms
// TODO: authentication of the requester
// TODO: handle params
// TODO: document in swagger
func matrixRoomDirectory(matrix *config.Matrix) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := &roomDirectoryReq{}
		if err := c.Bind(req); err != nil {
			log.Println("directory request binding failed:", err)
		}
		log.Printf("room directory:\nGET params: %+v\nHeaders: %+v", c.QueryParams(), c.Request().Header)
		return c.JSON(http.StatusOK, roomDirectoryResp{
			Chunk: []roomDirectoryItem{
				{
					Avatar:        "mxc://etke.cc/EPswbbDKYLNEjRYgEpHpRQue", // mrs room avatar
					Alias:         "#todo:" + matrix.ServerName,
					Guest:         false,
					Name:          "Not implemented yet",
					Members:       999,
					ID:            "!IyxAXBqViWHZfUkWjh:etke.cc", // mrs room id
					Topic:         "This is a stub endpoint, not fully implemented yet",
					WorldReadable: true,
				},
			},
			Total: 1,
		})
	}
}
