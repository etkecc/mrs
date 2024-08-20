package utils

import (
	"context"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/version"
)

const (
	// DefaultTimeout for http requests
	DefaultTimeout = 120 * time.Second
	// MaxRetries for http requests
	MaxRetries = 5
	// RetryDelay for http requests
	RetryDelay = 5 * time.Second
)

// httpClient with timeout
var httpClient = &http.Client{Timeout: DefaultTimeout}

// Get performs HTTP GET request with timeout, User-Agent, and retrier
func Get(ctx context.Context, uri string, maxRetries ...int) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
	if err != nil {
		return nil, err
	}
	return Do(req, maxRetries...)
}

// Do performs HTTP request with timeout, User-Agent, and retrier
func Do(req *http.Request, maxRetries ...int) (*http.Response, error) {
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

	// edge case: when function ends it execution and automatically calls cancel(),
	// it causes error "context canceled" when the function caller tries to read the body of the response
	// so we defer the cancel() function to be called only when there is an error
	var err error
	var resp *http.Response
	ctx, cancel := context.WithTimeout(span.Context(), DefaultTimeout)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", version.UserAgent)
	// no direct return, to use response and error in defer
	var retries int
	if len(maxRetries) > 0 {
		retries = maxRetries[0]
	} else {
		retries = MaxRetries
	}
	resp, err = httpRetry(ctx, req, retries)
	return resp, err
}

// httpRetry is a simple retry mechanism for http requests with exponential backoff
// that retries only on 5xx status codes
func httpRetry(ctx context.Context, req *http.Request, retries int, currentRetry ...int) (*http.Response, error) {
	if retries == 0 {
		return httpClient.Do(req)
	}

	retry := 1
	if len(currentRetry) > 0 {
		retry = currentRetry[0]
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return resp, err
	}
	if resp != nil && resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		log := zerolog.Ctx(ctx).With().
			Int("try", retry).
			Int("of", retries).
			Str("reason", resp.Status).
			Str("req", req.Method+" "+req.URL.String()).
			Logger()
		if retry <= retries {
			delay := time.Duration(retry) * RetryDelay
			log.Warn().Str("in", delay.String()).Msg("retrying")
			time.Sleep(delay)
			retry++
			return httpRetry(ctx, req, retries, retry)
		}
		log.Warn().Msg("max retries reached")
	}
	return resp, err
}
