package services

import (
	"bytes"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/model"
)

// MaxCacheAge to be used on immutable resources
const MaxCacheAge = "31536000"

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
	enabled bool
	url     string
	key     string
}

// NewCache service
func NewCache(maxAge int, bunnyURL string, bunnyKey string, stats cacheStats) *Cache {
	return &Cache{
		maxAge: strconv.Itoa(maxAge),
		stats:  stats,
		bunny: cacheBunny{
			enabled: bunnyURL != "" && bunnyKey != "",
			url:     bunnyURL,
			key:     bunnyKey,
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

			lastModified := cache.stats.Get().Indexing.FinishedAt.Format(http.TimeFormat)
			ifModifiedSince := c.Request().Header.Get("if-modified-since")
			if lastModified == ifModifiedSince && (c.Request().URL.Path != "/search") {
				return c.NoContent(http.StatusNotModified)
			}

			resp := c.Response()
			resp.Header().Set("Cache-Control", "max-age="+cache.maxAge+", public")
			if cache.bunny.enabled {
				resp.Header().Set("CDN-Tag", "mutable")
			}
			if c.Request().URL.Path != "/search" {
				resp.Header().Set("Last-Modified", lastModified)
			}
			return next(c)
		}
	}
}

// MiddlewareImmutable returns echo middleware with immutable in cache-control
func (cache *Cache) MiddlewareImmutable() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet {
				return next(c)
			}

			if c.Request().Header.Get("if-modified-since") != "" {
				return c.NoContent(http.StatusNotModified)
			}

			resp := c.Response()
			resp.Header().Del("Last-Modified")
			if cache.bunny.enabled {
				resp.Header().Set("CDN-Tag", "immutable")
			}
			resp.Header().Set("Cache-Control", "max-age="+MaxCacheAge+", immutable")
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
	if !cache.bunny.enabled {
		return
	}
	req, err := http.NewRequest("POST", cache.bunny.url, bytes.NewBuffer([]byte(`{"CacheTag":"mutable"}}`)))
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
