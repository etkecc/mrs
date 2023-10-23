package controllers

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/version"
)

type serverKeyResp struct {
	ServerName    string                       `json:"server_name"`
	ValidUntilTS  int64                        `json:"valid_unit_ts"`
	Signatures    map[string]map[string]string `json:"signatures,omitempty"`
	VerifyKeys    map[string]map[string]string `json:"verify_keys"`
	OldVerifyKeys map[string]map[string]string `json:"old_verify_keys"`
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
	oldKeys, err := model.KeysFrom(matrix.OldKeys)
	if err != nil {
		log.Println("ERROR: cannot parse old key from string:", err)
	}

	resp := &serverKeyResp{
		ServerName:    matrix.ServerName,
		ValidUntilTS:  time.Now().Add(time.Hour * 24).UnixMilli(),
		VerifyKeys:    map[string]map[string]string{},
		OldVerifyKeys: map[string]map[string]string{},
	}
	for _, key := range keys {
		resp.VerifyKeys[key.ID] = map[string]string{"key": key.Public}
	}
	for _, key := range oldKeys {
		resp.OldVerifyKeys[key.ID] = map[string]string{"key": key.Public}
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		log.Println("ERROR: cannot marshal matrix server key payload:", err)
	}
	resp.Signatures = map[string]map[string]string{matrix.ServerName: {}}
	for _, key := range keys {
		resp.Signatures[matrix.ServerName][key.ID] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(key.Private, payload))
	}
	return func(c echo.Context) error { return c.JSON(http.StatusOK, resp) }
}
