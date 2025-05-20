package controllers

import (
	"io"
	"net/http"

	"github.com/etkecc/go-msc1929"
	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

func addServer(dataSvc dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := dataSvc.AddServer(c.Request().Context(), c.Param("name"))
		return c.NoContent(code)
	}
}

func addServers(dataSvc dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer c.Request().Body.Close()
		jsonb, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		var servers []string
		err = json.Unmarshal(jsonb, &servers)
		if err != nil {
			return err
		}

		go dataSvc.AddServers(c.Request().Context(), servers, cfg.Get().Workers.Discovery)
		return c.NoContent(http.StatusAccepted)
	}
}

// checkMSC1929 is a simple tool to check if a server has a valid MSC1929 support file.
func checkMSC1929() echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")
		resp, err := msc1929.GetWithContext(c.Request().Context(), name)
		if err != nil {
			return c.JSONBlob(http.StatusBadRequest, utils.MustJSON(&model.MatrixError{
				Code:    "CC.ETKE.MSC1929_ERROR",
				Message: err.Error(),
			}))
		}
		if resp.IsEmpty() {
			return c.JSONBlob(http.StatusBadRequest, utils.MustJSON(&model.MatrixError{
				Code:    "CC.ETKE.MSC1929_EMPTY",
				Message: "The support file is missing or empty (contains neither contacts nor support url).",
			}))
		}
		if len(resp.Admins) > 0 {
			return c.JSONBlob(http.StatusBadRequest, utils.MustJSON(&model.MatrixError{
				Code:    "CC.ETKE.MSC1929_OUTDATED",
				Message: "The support file uses the deprecated 'admins' field. Please use 'contacts' instead.",
			}))
		}
		if len(resp.AllEmails()) == 0 {
			return c.JSONBlob(http.StatusBadRequest, utils.MustJSON(&model.MatrixError{
				Code:    "CC.ETKE.MSC1929_NO_EMAILS",
				Message: "The support file doesn't contain any email addresses.",
			}))
		}

		return c.NoContent(http.StatusNoContent)
	}
}
