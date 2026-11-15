package httpclient

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/etkecc/go-kit/retry"
)

// Exported defaults, safe to reference when overriding one knob relative to another.
const (
	// DefaultPoolSize is the pool size NewSingleHost applies to all pool dims, overriding stdlib's 2-per-host default.
	DefaultPoolSize = 256
	// DefaultIdleConnTimeout is how long an idle connection is kept before closing.
	DefaultIdleConnTimeout = 90 * time.Second
	// DefaultPerAttemptTimeout is the deadline applied to each individual attempt.
	DefaultPerAttemptTimeout = 10 * time.Second
	// DefaultMaxRetryAfter is the ceiling on an honored Retry-After; past it, the response returns immediately, live.
	DefaultMaxRetryAfter = 30 * time.Second
	// DefaultMultiHostIdleConnsPerHost caps per-host idle conns under NewMultiHost, low since a host is rarely revisited.
	DefaultMultiHostIdleConnsPerHost = 2
	// DefaultMultiHostIdleConnTimeout is how long NewMultiHost keeps an idle conn, short so FDs return fast in fan-out.
	DefaultMultiHostIdleConnTimeout = 30 * time.Second
)

const (
	defaultMaxRetries            = 3
	defaultDelayStep             = 200 * time.Millisecond
	defaultTLSHandshakeTimeout   = 10 * time.Second
	defaultResponseHeaderTimeout = 10 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
	defaultH2PingInterval        = 15 * time.Second
	defaultH2PingTimeout         = 15 * time.Second
	defaultDialTimeout           = 30 * time.Second
	defaultDialKeepAlive         = 30 * time.Second
)

// dialIPKey carries a WithDialIP pin on the ctx; a struct key avoids colliding with another package's string key.
type dialIPKey struct{}

// Option configures New, NewSingleHost, or NewMultiHost; each kind's interface rejects the wrong knob at compile time.
type Option interface{ apply(*config) }

// TransportOption tunes the transport: valid on NewTransport and the full-client constructors, but not on Wrap.
type TransportOption interface {
	Option
	isTransport()
}

// RetryOption tunes the retry layer: valid on Wrap and the full-client constructors, but not on NewTransport.
type RetryOption interface {
	Option
	isRetry()
}

// optionFunc adapts a plain config mutator into a TransportOption.
type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }
func (optionFunc) isTransport()      {}

// retryOptionFunc adapts a config mutator into a RetryOption, satisfying both apply and the isRetry marker.
type retryOptionFunc func(*config)

func (f retryOptionFunc) apply(c *config) { f(c) }
func (retryOptionFunc) isRetry()          {}

// defaultConfig seeds the shared defaults both constructors start from.
func defaultConfig() *config {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	return &config{
		maxIdleConns:          DefaultPoolSize,
		maxIdleConnsPerHost:   DefaultPoolSize,
		maxConnsPerHost:       0,
		idleConnTimeout:       DefaultIdleConnTimeout,
		tlsHandshakeTimeout:   defaultTLSHandshakeTimeout,
		responseHeaderTimeout: defaultResponseHeaderTimeout,
		expectContinueTimeout: defaultExpectContinueTimeout,
		perAttemptTimeout:     DefaultPerAttemptTimeout,
		maxRetries:            defaultMaxRetries,
		delayStep:             defaultDelayStep,
		protocols:             protocols,
		tlsMinVersion:         tls.VersionTLS12,
		maxRetryAfter:         DefaultMaxRetryAfter,
		budget:                noopBudget{},
	}
}

// defaultHTTP2Config is the keepalive-ping config both presets layer on to catch a dead conn.
func defaultHTTP2Config() *http.HTTP2Config {
	return &http.HTTP2Config{
		SendPingTimeout: defaultH2PingInterval,
		PingTimeout:     defaultH2PingTimeout,
	}
}

// singleHostPreset caps per-host conns to the pool size and enables HTTP/2 keepalive, on top of the shared defaults.
func singleHostPreset() Option {
	return optionFunc(func(c *config) {
		WithMaxConnsPerHost(DefaultPoolSize).apply(c)
		WithHTTP2Config(defaultHTTP2Config()).apply(c)
	})
}

// multiHostPreset trims per-host idle conns and reclaims them fast: a wide crawl meets each host once, then ghosts it.
func multiHostPreset() Option {
	return optionFunc(func(c *config) {
		WithMaxIdleConnsPerHost(DefaultMultiHostIdleConnsPerHost).apply(c)
		WithIdleConnTimeout(DefaultMultiHostIdleConnTimeout).apply(c)
		WithHTTP2Config(defaultHTTP2Config()).apply(c)
	})
}

// WithMaxIdleConns sets the total idle-connection pool size across all hosts.
func WithMaxIdleConns(n int) TransportOption {
	return optionFunc(func(c *config) { c.maxIdleConns = n })
}

// WithMaxIdleConnsPerHost sets the idle-connection pool size per host.
func WithMaxIdleConnsPerHost(n int) TransportOption {
	return optionFunc(func(c *config) { c.maxIdleConnsPerHost = n })
}

// WithMaxConnsPerHost caps total (active plus idle) connections per host; 0 is unlimited.
func WithMaxConnsPerHost(n int) TransportOption {
	return optionFunc(func(c *config) { c.maxConnsPerHost = n })
}

// WithIdleConnTimeout sets how long an idle connection is kept before closing.
func WithIdleConnTimeout(d time.Duration) TransportOption {
	return optionFunc(func(c *config) { c.idleConnTimeout = d })
}

// WithTLSHandshakeTimeout sets the TLS handshake deadline.
func WithTLSHandshakeTimeout(d time.Duration) TransportOption {
	return optionFunc(func(c *config) { c.tlsHandshakeTimeout = d })
}

// WithResponseHeaderTimeout sets how long to wait for response headers after the request.
func WithResponseHeaderTimeout(d time.Duration) TransportOption {
	return optionFunc(func(c *config) { c.responseHeaderTimeout = d })
}

// WithExpectContinueTimeout sets the wait for a 100-Continue after Expect headers.
func WithExpectContinueTimeout(d time.Duration) TransportOption {
	return optionFunc(func(c *config) { c.expectContinueTimeout = d })
}

// WithProtocols sets the HTTP protocols the transport negotiates.
func WithProtocols(p *http.Protocols) TransportOption {
	return optionFunc(func(c *config) { c.protocols = p })
}

// WithHTTP2Config sets the transport's HTTP/2 configuration.
func WithHTTP2Config(h2 *http.HTTP2Config) TransportOption {
	return optionFunc(func(c *config) { c.http2 = h2 })
}

// WithTLSMinVersion sets the minimum accepted TLS version (e.g. tls.VersionTLS13).
func WithTLSMinVersion(v uint16) TransportOption {
	return optionFunc(func(c *config) { c.tlsMinVersion = v })
}

// WithDialContext redirects the dial to a resolved IP; dial exactly that IP or the pin drops; WithDialGuard skips this.
func WithDialContext(fn func(context.Context, string, string) (net.Conn, error)) TransportOption {
	return optionFunc(func(c *config) { c.dialContext = fn })
}

// WithDialGuard refuses dials to private/loopback/link-local/metadata IPs at dial time, and nulls the proxy too.
func WithDialGuard() TransportOption {
	return optionFunc(func(c *config) { c.dialControl = dialGuard })
}

// WithDialIP pins the dial target for a delegated host whose resolved IP differs from its name; empty ip is a no-op.
func WithDialIP(ctx context.Context, ip string) context.Context {
	if ip == "" {
		return ctx
	}
	return context.WithValue(ctx, dialIPKey{}, ip)
}

func dialIPFromContext(ctx context.Context) string {
	ip, ok := ctx.Value(dialIPKey{}).(string)
	if !ok {
		return ""
	}
	return ip
}

// WithPerAttemptTimeout sets the deadline applied to each attempt; 0 disables it.
func WithPerAttemptTimeout(d time.Duration) RetryOption {
	return retryOptionFunc(func(c *config) { c.perAttemptTimeout = d })
}

// WithMaxRetryAfter sets the ceiling on an honored Retry-After header.
func WithMaxRetryAfter(d time.Duration) RetryOption {
	return retryOptionFunc(func(c *config) { c.maxRetryAfter = d })
}

// WithRetry replaces the default retrier; the caller then owns backoff, jitter, and the predicate.
func WithRetry(r *retry.Retry) RetryOption {
	return retryOptionFunc(func(c *config) { c.retrier = r })
}

// WithRetryIf overrides which errors are retryable; ignored when WithRetry supplies a full retrier.
func WithRetryIf(predicate func(error) bool) RetryOption {
	return retryOptionFunc(func(c *config) { c.retryIf = predicate })
}

// WithMaxRetries sets the default retrier's total attempt count, first try included: n=1 is one shot, zero mercy.
func WithMaxRetries(n int) RetryOption {
	return retryOptionFunc(func(c *config) { c.maxRetries = n })
}

// WithRetryDelayStep sets the default retrier's linear backoff step: retry i waits step*(i+1), jittered.
func WithRetryDelayStep(d time.Duration) RetryOption {
	return retryOptionFunc(func(c *config) { c.delayStep = d })
}

// WithRetryNonIdempotent opts non-idempotent methods (POST, PATCH) into retry; off by default to avoid double effects.
func WithRetryNonIdempotent(on bool) RetryOption {
	return retryOptionFunc(func(c *config) { c.retryNonIdempotent = on })
}

// WithRetryBudget sets the cross-request retry budget.
func WithRetryBudget(b RetryBudget) RetryOption {
	return retryOptionFunc(func(c *config) {
		if b != nil {
			c.budget = b
		}
	})
}

// WithOnAttempt registers a hook run after every attempt; it fires concurrently, so it must be concurrency-safe.
func WithOnAttempt(hook func(AttemptInfo)) RetryOption {
	return retryOptionFunc(func(c *config) { c.onAttempt = hook })
}
