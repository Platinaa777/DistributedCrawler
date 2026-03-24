package env

import (
	"os"
	"strings"
)

const (
	fetcherTypeEnvName     = "FETCHER_TYPE"
	chromeRemoteURLEnvName = "CHROME_REMOTE_URL"

	FetcherTypeHTTP    = "http"
	FetcherTypeBrowser = "browser"
)

// GetFetcherType returns the configured fetcher type ("http" or "browser").
// Controlled by the FETCHER_TYPE environment variable. Defaults to "http".
func GetFetcherType() string {
	t := strings.ToLower(strings.TrimSpace(os.Getenv(fetcherTypeEnvName)))
	if t == FetcherTypeBrowser {
		return FetcherTypeBrowser
	}
	return FetcherTypeHTTP
}

// GetChromeRemoteURL returns the remote Chrome CDP HTTP endpoint (e.g. "http://chrome:9222").
// Used when FETCHER_TYPE=browser. Empty string means a local Chrome process is spawned.
func GetChromeRemoteURL() string {
	return strings.TrimSpace(os.Getenv(chromeRemoteURLEnvName))
}
