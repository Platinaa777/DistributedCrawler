package env

import (
	"os"
	"strings"
)

const (
	fetcherTypeEnvName        = "FETCHER_TYPE"
	chromeRemoteURLEnvName    = "CHROME_REMOTE_URL"
	seleniumRemoteURLEnvName  = "SELENIUM_REMOTE_URL"

	FetcherTypeHTTP     = "http"
	FetcherTypeBrowser  = "browser"
	FetcherTypeSelenium = "selenium"
)

// GetFetcherType returns the configured fetcher type ("http", "browser", or "selenium").
// Controlled by the FETCHER_TYPE environment variable. Defaults to "http".
func GetFetcherType() string {
	t := strings.ToLower(strings.TrimSpace(os.Getenv(fetcherTypeEnvName)))
	switch t {
	case FetcherTypeBrowser:
		return FetcherTypeBrowser
	case FetcherTypeSelenium:
		return FetcherTypeSelenium
	default:
		return FetcherTypeHTTP
	}
}

// GetChromeRemoteURL returns the remote Chrome CDP HTTP endpoint (e.g. "http://chrome:9222").
// Used when FETCHER_TYPE=browser. Empty string means a local Chrome process is spawned.
func GetChromeRemoteURL() string {
	return strings.TrimSpace(os.Getenv(chromeRemoteURLEnvName))
}

// GetSeleniumRemoteURL returns the Selenium WebDriver hub endpoint (e.g. "http://selenium:4444/wd/hub").
// Used when FETCHER_TYPE=selenium.
func GetSeleniumRemoteURL() string {
	return strings.TrimSpace(os.Getenv(seleniumRemoteURLEnvName))
}
