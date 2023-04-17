package services

import (
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/model"
)

type cacheStats interface {
	Get() *model.IndexStats
}

// Cache service
type Cache struct {
	maxAge string
	bunny  cacheBunny
	stats  cacheStats
}

type cacheBunny struct {
	url string
	key string
}

// NewCache service
func NewCache(maxAge int, bunnyURL string, bunnyKey string, stats cacheStats) *Cache {
	return &Cache{
		maxAge: strconv.Itoa(maxAge),
		stats:  stats,
		bunny: cacheBunny{
			url: bunnyURL,
			key: bunnyKey,
		},
	}
}

// Middleware returns echo middleware
func (cache *Cache) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet {
				return next(c)
			}

			lastModified := cache.stats.Get().Index.FinishedAt.Format(http.TimeFormat)
			ifModifiedSince := c.Request().Header.Get("if-modified-since")
			if lastModified == ifModifiedSince {
				return c.NoContent(http.StatusNotModified)
			}

			resp := c.Response()
			resp.Header().Set("Cache-Control", "max-age="+cache.maxAge+", public")
			resp.Header().Set("Last-Modified", lastModified)
			return next(c)
		}
	}
}

// Purge cache. At this moment works with BunnyCDN only
func (cache *Cache) Purge() {
	cache.purgeBunnyCDN()
}

// purgeBunnyCDN cache
// ref: https://docs.bunny.net/reference/pullzonepublic_purgecachepostbytag
func (cache *Cache) purgeBunnyCDN() {
	if cache.bunny.url == "" || cache.bunny.key == "" {
		return
	}
	req, err := http.NewRequest("POST", cache.bunny.url, nil)
	if err != nil {
		log.Println("cannot purge bunny cache", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AccessKey", cache.bunny.key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("cannot purge bunny cache", err)
		return
	}
	resp.Body.Close() // no need
	if resp.StatusCode != http.StatusNoContent {
		log.Println("cannot purge bunny cache - http status code is", resp.StatusCode)
	}
}
