package fetcher

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	defaultBrowserTimeout = 30 * time.Second
	defaultUserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	defaultViewportWidth  = 1366
	defaultViewportHeight = 768
	networkIdleWindow     = 800 * time.Millisecond
	networkIdleTimeout    = 8 * time.Second
	domStableTimeout      = 5 * time.Second
	domStableInterval     = 400 * time.Millisecond
	domStableSamples      = 3
)

// BrowserFetcher renders pages using a headless browser (Chromium).
// When remoteURL is set (e.g. "http://chrome:9222") it connects to an external
// Chrome instance via CDP instead of spawning a local process.
type BrowserFetcher struct {
	authOptions models.AuthOptions
	retryPolicy models.RetryPolicy
	timeout     time.Duration
	remoteURL   string // empty = local Chrome process; non-empty = remote CDP endpoint
}

// NewBrowserFetcher creates a browser-based fetcher with specified options.
// remoteURL is the HTTP endpoint of a remote Chrome instance (e.g. "http://chrome:9222").
// Pass an empty string to spawn a local Chrome process instead.
func NewBrowserFetcher(auth models.AuthOptions, retry models.RetryPolicy, remoteURL string) *BrowserFetcher {
	if retry.MaxAttempts == 0 {
		retry.MaxAttempts = 1
	}

	return &BrowserFetcher{
		authOptions: auth,
		retryPolicy: retry,
		timeout:     defaultBrowserTimeout,
		remoteURL:   remoteURL,
	}
}

// Fetch performs a browser-rendered fetch with retry logic and returns the result.
func (f *BrowserFetcher) Fetch(ctx context.Context, url string) (*services.FetchResult, error) {
	var lastErr error
	backoff := time.Duration(f.retryPolicy.BackoffInitialMs) * time.Millisecond

	for attempt := uint64(0); attempt < f.retryPolicy.MaxAttempts; attempt++ {
		result, err := f.doFetch(ctx, url)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if attempt < f.retryPolicy.MaxAttempts-1 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}

			backoff = time.Duration(float64(backoff) * f.retryPolicy.BackoffMultiplier)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", f.retryPolicy.MaxAttempts, lastErr)
}

func (f *BrowserFetcher) doFetch(ctx context.Context, url string) (*services.FetchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	var allocCtx context.Context
	var allocCancel context.CancelFunc
	if f.remoteURL != "" {
		wsURL, err := resolveWSURL(ctx, f.remoteURL)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve remote Chrome WS URL: %w", err)
		}
		allocCtx, allocCancel = chromedp.NewRemoteAllocator(ctx, wsURL)
	} else {
		allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
		)
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctx, allocOpts...)
	}
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	var (
		html        string
		finalURL    string
		statusCode  int
		contentType = "text/html"
	)
	var responseLock sync.Mutex
	var networkLock sync.Mutex
	inFlight := 0
	lastActivity := time.Now()

	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventResponseReceived:
			if e.Type == network.ResourceTypeDocument {
				responseLock.Lock()
				if statusCode == 0 {
					statusCode = int(e.Response.Status)
					if e.Response.MimeType != "" {
						contentType = e.Response.MimeType
					}
					if e.Response.URL != "" {
						finalURL = e.Response.URL
					}
				}
				responseLock.Unlock()
			}
		case *network.EventRequestWillBeSent:
			networkLock.Lock()
			inFlight++
			lastActivity = time.Now()
			networkLock.Unlock()
		case *network.EventLoadingFinished, *network.EventLoadingFailed:
			networkLock.Lock()
			if inFlight > 0 {
				inFlight--
			}
			lastActivity = time.Now()
			networkLock.Unlock()
		}
	})

	headers := network.Headers{
		"User-Agent":      defaultUserAgent,
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7",
	}
	if f.authOptions.Cookie != "" {
		headers["Cookie"] = f.authOptions.Cookie
	}
	if f.authOptions.BasicUser != "" {
		cred := f.authOptions.BasicUser + ":" + f.authOptions.BasicPassword
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(cred))
	}
	if f.authOptions.BearerToken != "" {
		headers["Authorization"] = "Bearer " + f.authOptions.BearerToken
	}

	if err := chromedp.Run(
		browserCtx,
		network.Enable(),
		network.SetExtraHTTPHeaders(headers),
		emulation.SetUserAgentOverride(defaultUserAgent),
		emulation.SetDeviceMetricsOverride(defaultViewportWidth, defaultViewportHeight, 1, false),
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		waitForReadyState("complete"),
		waitForNetworkIdle(&networkLock, &inFlight, &lastActivity, networkIdleWindow, networkIdleTimeout),
		waitForDOMStable(domStableTimeout, domStableInterval, domStableSamples),
		chromedp.Location(&finalURL),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("failed to render page: %w", err)
	}

	if html == "" {
		return nil, fmt.Errorf("empty HTML after render")
	}
	if finalURL == "" {
		finalURL = url
	}
	if statusCode == 0 {
		statusCode = httpStatusOK
	}

	body := []byte(html)

	return &services.FetchResult{
		Body:        body,
		FinalURL:    finalURL,
		StatusCode:  statusCode,
		ContentType: contentType,
	}, nil
}

const httpStatusOK = 200

// resolveWSURL fetches the WebSocket debugger URL from a remote Chrome CDP endpoint.
// remoteURL is the HTTP base, e.g. "http://chrome:9222".
func resolveWSURL(ctx context.Context, remoteURL string) (string, error) {
	endpoint := strings.TrimRight(remoteURL, "/") + "/json/version"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	var result struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if result.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("empty webSocketDebuggerUrl from %s", endpoint)
	}
	return result.WebSocketDebuggerURL, nil
}

func waitForReadyState(state string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		var readyState string
		if err := chromedp.Evaluate(`document.readyState`, &readyState).Do(ctx); err != nil {
			return err
		}
		if readyState == state {
			return nil
		}
		pollCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		return chromedp.Poll(
			`document.readyState`,
			&readyState,
			chromedp.WithPollingInterval(200*time.Millisecond),
		).Do(pollCtx)
	}
}

func waitForNetworkIdle(lock *sync.Mutex, inFlight *int, lastActivity *time.Time, idleWindow time.Duration, timeout time.Duration) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		deadline := time.Now().Add(timeout)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				lock.Lock()
				active := *inFlight
				last := *lastActivity
				lock.Unlock()

				if active == 0 && time.Since(last) >= idleWindow {
					return nil
				}
				if time.Now().After(deadline) {
					return nil
				}
			}
		}
	}
}

func waitForDOMStable(timeout time.Duration, interval time.Duration, samples int) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		deadline := time.Now().Add(timeout)
		prev := -1
		stable := 0

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var length int
			if err := chromedp.Evaluate(`document.documentElement ? document.documentElement.outerHTML.length : 0`, &length).Do(ctx); err != nil {
				return err
			}

			if length == prev {
				stable++
				if stable >= samples {
					return nil
				}
			} else {
				stable = 0
				prev = length
			}

			if time.Now().After(deadline) {
				return nil
			}

			if err := chromedp.Sleep(interval).Do(ctx); err != nil {
				return err
			}
		}
	}
}
