package controllers

import (
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

func stats(stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		info := stats.Get()
		resp := map[string]any{
			"servers": info.Servers.Online,
			"rooms":   info.Rooms.Indexed,
		}
		resp["details"] = statsDetails(info)

		tl := stats.GetTL(c.Request().Context())
		keys := utils.MapKeys(tl)
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Before(keys[j])
		})
		timeline := []map[string]any{}
		for _, k := range keys {
			timeline = append(timeline, map[string]any{
				"date":    k.Format(time.DateOnly),
				"details": statsDetails(tl[k]),
			})
		}

		if len(keys) > 0 {
			resp["timeline"] = timeline
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func statsDetails(stats *model.IndexStats) map[string]any {
	return map[string]any{
		"servers": map[string]any{
			"online":    stats.Servers.Online,
			"indexable": stats.Servers.Indexable,
		},
		"rooms": map[string]any{
			"indexed": stats.Rooms.Indexed,
			"parsed":  stats.Rooms.Parsed,
		},
	}
}
