package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// release is a version of the app published for download.
type release struct {
	Version string
	URL     string
	Notes   string
}

// releaseRepo is the repository the app's releases are published in.
const releaseRepo = "younland/apigo"

// projectURL is the page the settings link to: where the project is kept, its
// releases beside it.
const projectURL = "https://github.com/" + releaseRepo

// releaseURL is where GitHub lists the newest release of the app.
const releaseURL = "https://api.github.com/repos/" + releaseRepo + "/releases/latest"

// releaseTimeout caps how long a check waits for GitHub.
const releaseTimeout = 15 * time.Second

// latestRelease asks GitHub for the newest published release of the app: a
// draft or a prerelease is not what the app updates to.
func latestRelease(ctx context.Context) (release, error) {
	return releaseAt(ctx, releaseURL)
}

// releaseAt reads the newest release from a URL of GitHub's API, as the tests
// point at a server of their own.
func releaseAt(ctx context.Context, url string) (release, error) {
	ctx, cancel := context.WithTimeout(ctx, releaseTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return release{}, err
	}
	// GitHub answers a request that does not name itself with a refusal.
	req.Header.Set("User-Agent", "apigo")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("GitHub 返回 %s", resp.Status)
	}
	var body struct {
		Tag        string `json:"tag_name"`
		URL        string `json:"html_url"`
		Notes      string `json:"body"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return release{}, err
	}
	if body.Draft || body.Prerelease || body.Tag == "" {
		return release{}, nil // nothing published to update to
	}
	return release{
		Version: strings.TrimPrefix(strings.TrimSpace(body.Tag), "v"),
		URL:     body.URL,
		Notes:   body.Notes,
	}, nil
}

// newerVersion reports whether latest is a newer version than current: numbers
// separated by dots, with a leading v on either, and anything a version cannot
// hold left out.
func newerVersion(latest, current string) bool {
	l, ok := versionParts(latest)
	if !ok {
		return false
	}
	c, _ := versionParts(current)
	for i := 0; i < len(l) || i < len(c); i++ {
		var a, b int
		if i < len(l) {
			a = l[i]
		}
		if i < len(c) {
			b = c[i]
		}
		if a != b {
			return a > b
		}
	}
	return false
}

// versionParts splits a version into the numbers that order it, and reports
// whether it holds any: "" is no version at all.
func versionParts(version string) ([]int, bool) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return nil, false
	}
	fields := strings.FieldsFunc(version, func(r rune) bool {
		return r == '.' || r == '-' || r == '+'
	})
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		n, err := strconv.Atoi(field)
		if err != nil {
			break // a suffix as -beta.1 orders nothing
		}
		parts = append(parts, n)
	}
	return parts, len(parts) > 0
}
