package services

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

const (
	// MaxCacheAge to be used on immutable resources
	MaxCacheAge   = "31536000"
	bunnyIPv4List = "https://bunnycdn.com/api/system/edgeserverlist"
	bunnyIPv6List = "https://bunnycdn.com/api/system/edgeserverlist/IPv6"
)

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
	cfg      ConfigService
	bunnyIPs map[string]struct{}
	stats    cacheStats
}

// NewCache service
func NewCache(cfg ConfigService, stats cacheStats) *Cache {
	cache := &Cache{
		cfg:      cfg,
		bunnyIPs: make(map[string]struct{}),
		stats:    stats,
	}
	cache.initBunnyIPs(utils.NewContext())
	return cache
}

func (cache *Cache) pullBunnyIPs(ctx context.Context, uri string) []string {
	log := zerolog.Ctx(ctx)
	resp, err := utils.Get(ctx, uri)
	if err != nil {
		log.Error().Err(err).Msg("cannot get bunny ips")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status_code", resp.StatusCode).Msg("cannot get bunny ips")
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("cannot read bunny ips")
		return nil
	}
	var ips []string
	if err := json.Unmarshal(body, &ips); err != nil {
		log.Error().Err(err).Msg("cannot unmarshal bunny ips")
		return nil
	}
	return ips
}

func (cache *Cache) initBunnyIPs(ctx context.Context) {
	if cache.cfg.Get().Cache.Bunny.Key == "" {
		return
	}
	log := zerolog.Ctx(ctx)
	for _, ip := range append(cache.pullBunnyIPs(ctx, bunnyIPv4List), cache.pullBunnyIPs(ctx, bunnyIPv6List)...) {
		cache.bunnyIPs[ip] = struct{}{}
	}
	log.Info().Int("count", len(cache.bunnyIPs)).Msg("bunny ips loaded")
}

// IsBunny returns true if the IP is a BunnyCDN IP
func (cache *Cache) IsBunny(ip string) bool {
	_, ok := cache.bunnyIPs[ip]
	return ok
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

// Purge cache. At this moment works with BunnyCDN only
func (cache *Cache) Purge(ctx context.Context) {
	cache.purgeBunnyCDN(ctx)
}

// purgeBunnyCDN cache
// ref: https://docs.bunny.net/reference/pullzonepublic_purgecachepostbytag
func (cache *Cache) purgeBunnyCDN(ctx context.Context) {
	bunny := cache.cfg.Get().Cache.Bunny
	if bunny.Key == "" || bunny.URL == "" {
		return
	}
	log := zerolog.Ctx(ctx)
	req, err := http.NewRequest(http.MethodPost, bunny.URL, bytes.NewBufferString(`{"CacheTag":"mutable"}}`))
	if err != nil {
		log.Error().Err(err).Msg("cannot purge bunny cache")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AccessKey", bunny.Key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("cannot purge bunny cache")
		return
	}
	resp.Body.Close() // no need
	if resp.StatusCode != http.StatusNoContent {
		log.Error().Err(err).Int("status_code", resp.StatusCode).Msg("cannot purge bunny cache")
	}
}
