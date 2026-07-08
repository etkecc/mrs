package controllers

import (
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
)

// @Summary		Crawler statistics
// @Description	Live index counts: servers found online and indexable, rooms indexed and parsed, plus the server software breakdown (how much of the federation is Synapse, basically). Timeline is present only once we have history behind us; on a fresh index the key is absent, not an empty array.
// @Produce		json
// @Success		200	{object}	model.StatsResponse
// @Router			/stats [get]
func stats(stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		info := stats.Get()
		resp := model.StatsResponse{
			Servers: info.Servers.Online,
			Rooms:   info.Rooms.Indexed,
			Details: statsDetails(info),
		}

		tl := stats.GetTL(c.Request().Context())
		keys := make([]time.Time, 0, len(tl))
		for k := range tl {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Before(keys[j])
		})
		for _, k := range keys {
			resp.Timeline = append(resp.Timeline, model.StatsTimelineEntry{
				Date:    k.Format(time.DateOnly),
				Details: statsDetails(tl[k]),
			})
		}

		return c.JSON(http.StatusOK, resp)
	}
}

func statsDetails(stats *model.IndexStats) model.StatsDetails {
	return model.StatsDetails{
		Servers: model.StatsDetailsServers{
			Online:    stats.Servers.Online,
			Indexable: stats.Servers.Indexable,
			Software:  stats.Servers.Software,
		},
		Rooms: model.StatsDetailsRooms{
			Indexed: stats.Rooms.Indexed,
			Parsed:  stats.Rooms.Parsed,
		},
	}
}
