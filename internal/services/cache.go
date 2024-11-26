package services

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
)

// MaxCacheAge to be used on immutable resources
const MaxCacheAge = "31536000"

type cacheStats interface {
	Get() *model.IndexStats
}

var noncacheablePaths = map[string]struct{}{
	"/_health":                               {},
	"/_matrix/key/v2/server":                 {},
	"/_matrix/federation/v1/query/directory": {},
}

// Cache service
type Cache struct {
	cfg   ConfigService
	stats cacheStats
}

// NewCache service
func NewCache(cfg ConfigService, stats cacheStats) *Cache {
	cache := &Cache{
		cfg:   cfg,
		stats: stats,
	}
	return cache
}

func (cache *Cache) clearHeaders(c echo.Context) {
	c.Response().Header().Del("Cache-Control")
	c.Response().Header().Del("CDN-Tag")
	c.Response().Header().Del("Last-Modified")
}

func (cache *Cache) getLastModified() string {
	return cache.stats.Get().Indexing.FinishedAt.Format(http.TimeFormat)
}

// Middleware returns cache middleware
func (cache *Cache) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet {
				return next(c)
			}

			_, noncacheable := noncacheablePaths[c.Request().URL.Path]
			if noncacheable {
				cache.clearHeaders(c)
				c.Response().Header().Set("Cache-Control", "no-cache")
				return next(c)
			}

			lastModified := cache.getLastModified()
			ifModifiedSince := c.Request().Header.Get("if-modified-since")
			if lastModified == ifModifiedSince {
				return c.NoContent(http.StatusNotModified)
			}

			maxAge := strconv.Itoa(cache.cfg.Get().Cache.MaxAge)
			c.Response().Header().Set("Cache-Control", "max-age="+maxAge+", public")
			c.Response().Header().Set("CDN-Tag", "mutable")
			c.Response().Header().Set("Last-Modified", lastModified)
			return next(c)
		}
	}
}

// MiddlewareSearch returns cache middleware for search endpoints
func (cache *Cache) MiddlewareSearch() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet {
				return next(c)
			}

			// do not cache search results with query in GET params or matrix search results
			if c.Request().URL.Query().Has("q") || c.Request().URL.Query().Has("since") {
				cache.clearHeaders(c)
				return next(c)
			}

			lastModified := cache.getLastModified()
			ifModifiedSince := c.Request().Header.Get("if-modified-since")
			if lastModified == ifModifiedSince {
				return c.NoContent(http.StatusNotModified)
			}

			maxAge := strconv.Itoa(cache.cfg.Get().Cache.MaxAgeSearch)
			c.Response().Header().Set("Cache-Control", "max-age="+maxAge+", public")
			c.Response().Header().Set("CDN-Tag", "mutable")
			c.Response().Header().Set("Last-Modified", lastModified)
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

			cache.clearHeaders(c)
			c.Response().Header().Set("CDN-Tag", "immutable")
			c.Response().Header().Set("Cache-Control", "max-age="+MaxCacheAge+", immutable")
			return next(c)
		}
	}
}
