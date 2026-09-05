package controllers

import (
	"regexp"
	"testing"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/docs"
	"github.com/etkecc/mrs/internal/model"
)

// stubConfig / stubCache are the only deps ConfigureRouter touches at registration; the rest are unused closures.
type stubConfig struct{}

// Auth must be non-nil: ConfigureRouter reads cfg.Get().Auth at registration; other *Config* fields are nil-safe.
func (stubConfig) Get() *model.Config { return &model.Config{Auth: &model.ConfigAuth{}} }

type stubCache struct{}

func passthroughMiddleware(next echo.HandlerFunc) echo.HandlerFunc { return next }

func (stubCache) Middleware() echo.MiddlewareFunc          { return passthroughMiddleware }
func (stubCache) MiddlewareSearch() echo.MiddlewareFunc    { return passthroughMiddleware }
func (stubCache) MiddlewareImmutable() echo.MiddlewareFunc { return passthroughMiddleware }

func testRouter(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	ConfigureRouter(e, stubConfig{}, nil, nil, stubCache{}, nil, nil, nil, nil, nil)
	return e
}

func routeSet(e *echo.Echo) map[string]bool {
	set := make(map[string]bool)
	for _, r := range e.Routes() {
		set[r.Path] = true
	}
	return set
}

var swaggerParam = regexp.MustCompile(`\{([^}]+)\}`)

// swaggerToEcho rewrites `{param}` path segments into echo's `:param` form.
func swaggerToEcho(path string) string {
	return swaggerParam.ReplaceAllString(path, ":$1")
}

// TestSwaggerPathsMatchRouter asserts every documented @Router path is really registered in ConfigureRouter.
func TestSwaggerPathsMatchRouter(t *testing.T) {
	routes := routeSet(testRouter(t))

	var spec struct {
		Paths map[string]any `json:"paths"`
	}
	if err := json.Unmarshal([]byte(docs.SwaggerInfo.ReadDoc()), &spec); err != nil {
		t.Fatalf("parse generated swagger spec: %v", err)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("generated swagger spec has no paths; run `just swaggerfix`")
	}

	for path := range spec.Paths {
		echoPath := swaggerToEcho(path)
		if !routes[echoPath] {
			t.Errorf("documented path %q (echo %q) is not registered in ConfigureRouter", path, echoPath)
		}
	}
}

// TestDoNotGZIPPathsRegistered asserts every gzip-skip path is a real, currently-registered route.
func TestDoNotGZIPPathsRegistered(t *testing.T) {
	routes := routeSet(testRouter(t))
	for path := range doNotGZIP {
		if !routes[path] {
			t.Errorf("doNotGZIP path %q is not a registered route", path)
		}
	}
}
