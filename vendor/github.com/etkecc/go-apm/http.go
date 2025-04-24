package apm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

const (
	// MaxRetries for http requests
	MaxRetries = 5
	// RetryDelay for http requests
	RetryDelay = 5 * time.Second
)

// RoundTripper is an http.RoundTripper that instruments http requests
type RoundTripper struct {
	rt         http.RoundTripper
	maxRetries int
	retryDelay time.Duration
}

// RoundTripperOption is a function that configures an APMRoundTripper
type RoundTripperOption func(*RoundTripper)

// WithMaxRetries sets the maximum number of retries for http requests, otherwise defaults to 5
func WithMaxRetries(maxRetries int) RoundTripperOption {
	return func(rt *RoundTripper) {
		rt.maxRetries = maxRetries
	}
}

// WithRetryDelay sets the delay between retries for http requests, otherwise defaults to 5 seconds
func WithRetryDelay(retryDelay time.Duration) RoundTripperOption {
	return func(rt *RoundTripper) {
		rt.retryDelay = retryDelay
	}
}

// WrapClient wraps an http.Client with APM instrumentation and retry logic
func WrapClient(c *http.Client, opts ...RoundTripperOption) *http.Client {
	if c == nil {
		c = http.DefaultClient
	}
	c.Transport = WrapRoundTripper(c.Transport, opts...)
	return c
}

// WrapRoundTripper wraps an http.RoundTripper with APM instrumentation and retry logic
func WrapRoundTripper(rt http.RoundTripper, opts ...RoundTripperOption) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}
	apmrt := &RoundTripper{
		rt:         rt,
		maxRetries: MaxRetries,
		retryDelay: RetryDelay,
	}
	for _, opt := range opts {
		opt(apmrt)
	}
	return apmrt
}

// RoundTrip implements the http.RoundTripper interface, creating a transaction and span for each http request
// and handling retries for 5xx responses
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// creating a custom http.client transaction if not already present to avoid unlabeled transactions
	name := req.Method + " " + req.URL.String()
	transaction := sentry.TransactionFromContext(req.Context())
	if transaction == nil {
		transaction = sentry.StartTransaction(req.Context(), name,
			sentry.WithOpName("http.client"),
			sentry.WithTransactionSource(sentry.SourceURL),
		)
		defer transaction.Finish()
		req = req.WithContext(transaction.Context())
	}

	// creating a custom span for the http.client transaction, duplicating transaction options, to avoid missing context
	span := sentry.StartSpan(req.Context(), "http.client",
		sentry.WithOpName("http.client"),
		sentry.WithDescription(name),
		sentry.WithTransactionName(name),
		sentry.WithTransactionSource(sentry.SourceURL),
	)
	defer span.Finish()

	var resp *http.Response
	var err error
	if rt.maxRetries == 0 {
		resp, err = rt.rt.RoundTrip(req)
	} else {
		resp, err = rt.retry(req)
	}

	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		HealthcheckFail(strings.NewReader(fmt.Sprintf("http.RoundTripper: %s %s failed: %+v", req.Method, req.URL.String(), err)))
	}
	return resp, err
}

// retry is a simple retry mechanism for http requests with exponential backoff
func (rt *RoundTripper) retry(req *http.Request, currentRetry ...int) (*http.Response, error) {
	retry := 1
	if len(currentRetry) > 0 {
		retry = currentRetry[0]
	}

	var body []byte
	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		body = data
		req.Body = io.NopCloser(bytes.NewReader(data))
	}

	resp, err := rt.rt.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp != nil && resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		log := zerolog.Ctx(req.Context()).With().
			Int("try", retry).
			Int("of", rt.maxRetries).
			Str("reason", resp.Status).
			Str("req", req.Method+" "+req.URL.String()).
			Logger()
		if retry <= rt.maxRetries {
			delay := time.Duration(retry) * rt.retryDelay
			log.Warn().Str("in", delay.String()).Msg("retrying")
			if body != nil {
				req.Body = io.NopCloser(bytes.NewReader(body))
			}
			time.Sleep(delay)
			retry++
			return rt.retry(req, retry)
		}
		log.Warn().Msg("max retries reached")
	}
	return resp, err
}
