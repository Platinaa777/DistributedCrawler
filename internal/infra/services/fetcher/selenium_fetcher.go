package fetcher

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/tebeka/selenium"
)

const (
	defaultSeleniumTimeout = 30 * time.Second
)

// SeleniumFetcher renders pages using a remote Selenium WebDriver (e.g. Selenium Grid).
// remoteURL is the WebDriver hub URL, e.g. "http://selenium:4444/wd/hub".
type SeleniumFetcher struct {
	authOptions models.AuthOptions
	retryPolicy models.RetryPolicy
	timeout     time.Duration
	remoteURL   string
}

// NewSeleniumFetcher creates a Selenium-based fetcher.
// remoteURL is the Selenium WebDriver hub endpoint (e.g. "http://selenium:4444/wd/hub").
func NewSeleniumFetcher(auth models.AuthOptions, retry models.RetryPolicy, remoteURL string) *SeleniumFetcher {
	if retry.MaxAttempts == 0 {
		retry.MaxAttempts = 1
	}
	return &SeleniumFetcher{
		authOptions: auth,
		retryPolicy: retry,
		timeout:     defaultSeleniumTimeout,
		remoteURL:   remoteURL,
	}
}

// Fetch performs a Selenium-rendered fetch with retry logic.
func (f *SeleniumFetcher) Fetch(ctx context.Context, rawURL string) (*services.FetchResult, error) {
	var lastErr error
	backoff := time.Duration(f.retryPolicy.BackoffInitialMs) * time.Millisecond

	for attempt := uint64(0); attempt < f.retryPolicy.MaxAttempts; attempt++ {
		result, err := f.doFetch(ctx, rawURL)
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

func (f *SeleniumFetcher) doFetch(ctx context.Context, rawURL string) (*services.FetchResult, error) {
	caps := selenium.Capabilities{
		"browserName": "chrome",
		"goog:chromeOptions": map[string]any{
			"args": []string{
				"--headless",
				"--disable-gpu",
				"--no-sandbox",
				"--disable-dev-shm-usage",
				"--user-agent=" + defaultUserAgent,
				fmt.Sprintf("--window-size=%d,%d", defaultViewportWidth, defaultViewportHeight),
			},
		},
	}

	wd, err := selenium.NewRemote(caps, f.remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Selenium hub at %s: %w", f.remoteURL, err)
	}
	defer wd.Quit() //nolint:errcheck

	if err := wd.SetPageLoadTimeout(f.timeout); err != nil {
		return nil, fmt.Errorf("failed to set page load timeout: %w", err)
	}

	// For basic auth, embed credentials directly in the URL (user:pass@host).
	navigateURL := rawURL
	if f.authOptions.BasicUser != "" {
		navigateURL = embedBasicAuth(rawURL, f.authOptions.BasicUser, f.authOptions.BasicPassword)
	}

	done := make(chan error, 1)
	go func() {
		done <- wd.Get(navigateURL)
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while loading page: %w", ctx.Err())
	case navErr := <-done:
		if navErr != nil {
			return nil, fmt.Errorf("failed to navigate to %s: %w", rawURL, navErr)
		}
	}

	// Set cookies after initial navigation (WebDriver requires the browser to be on the same domain).
	if f.authOptions.Cookie != "" {
		if cookieErr := setCookies(wd, f.authOptions.Cookie); cookieErr != nil {
			return nil, fmt.Errorf("failed to set cookies: %w", cookieErr)
		}
		// Reload so the cookies are sent with the real request.
		reloadDone := make(chan error, 1)
		go func() { reloadDone <- wd.Get(rawURL) }()
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during cookie reload: %w", ctx.Err())
		case reloadErr := <-reloadDone:
			if reloadErr != nil {
				return nil, fmt.Errorf("failed to reload after cookie injection: %w", reloadErr)
			}
		}
	}

	// Wait for document.readyState == "complete".
	if waitErr := waitForSeleniumReady(ctx, wd, f.timeout); waitErr != nil {
		return nil, fmt.Errorf("page did not reach ready state: %w", waitErr)
	}

	finalURL, err := wd.CurrentURL()
	if err != nil || finalURL == "" {
		finalURL = rawURL
	}

	src, err := wd.PageSource()
	if err != nil {
		return nil, fmt.Errorf("failed to get page source: %w", err)
	}
	if src == "" {
		return nil, fmt.Errorf("empty page source")
	}

	return &services.FetchResult{
		Body:        []byte(src),
		FinalURL:    finalURL,
		StatusCode:  httpStatusOK, // Selenium doesn't expose HTTP status codes natively
		ContentType: "text/html",
	}, nil
}

// embedBasicAuth injects user:password into the URL authority component.
func embedBasicAuth(rawURL, user, password string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.User = url.UserPassword(user, password)
	return u.String()
}

// setCookies parses a raw "name=value; name2=value2" Cookie header and adds each
// cookie to the WebDriver session (the browser must already be on the target domain).
func setCookies(wd selenium.WebDriver, cookieHeader string) error {
	for _, pair := range strings.Split(cookieHeader, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.IndexByte(pair, '=')
		var name, value string
		if idx < 0 {
			name = pair
		} else {
			name = pair[:idx]
			value = pair[idx+1:]
		}
		if err := wd.AddCookie(&selenium.Cookie{Name: name, Value: value}); err != nil {
			return fmt.Errorf("add cookie %q: %w", name, err)
		}
	}
	return nil
}

// waitForSeleniumReady polls document.readyState until "complete" or timeout.
func waitForSeleniumReady(ctx context.Context, wd selenium.WebDriver, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		val, err := wd.ExecuteScript(`return document.readyState`, nil)
		if err == nil {
			if s, ok := val.(string); ok && s == "complete" {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return nil // proceed anyway after timeout
		}
		time.Sleep(300 * time.Millisecond)
	}
}
