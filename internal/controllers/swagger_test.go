package controllers

import (
	"regexp"
	"testing"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/docs"
	"github.com/etkecc/mrs/internal/model"
)

// stubConfig / stubCache are the only two dependencies ConfigureRouter touches at
// registration time (auth realms + gzip/cache middleware). Everything else is captured
// into handler closures that this test never invokes, so nil is fine for them.
type stubConfig struct{}

// Auth must be non-nil: ConfigureRouter reads cfg.Get().Auth.<realm> when it builds the
// basic-auth middleware at registration time. The realms themselves are zero-value (value
// fields), which is fine, the validators are never invoked here. Every other *Config* pointer
// is either nil-checked (Blocklist) or only touched at request time (Matrix), so nil is fine.
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

// TestSwaggerPathsMatchRouter asserts every documented @Router path is really registered,
// so an annotation cannot drift from its handler's echo.GET/POST without the suite noticing.
// This is the single guard the whole "document exactly what mrs does" mandate leans on.
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

// TestDoNotGZIPPathsRegistered asserts every gzip-skip path is a real route, so the map
// cannot rot into skipping a path that no longer exists (or miss one that was renamed).
func TestDoNotGZIPPathsRegistered(t *testing.T) {
	routes := routeSet(testRouter(t))
	for path := range doNotGZIP {
		if !routes[path] {
			t.Errorf("doNotGZIP path %q is not a registered route", path)
		}
	}
}
