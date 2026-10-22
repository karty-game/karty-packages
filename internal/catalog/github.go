package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

const (
	githubReleasesPerPage = 100
	maximumRepositories   = 100
	maximumReleasePages   = 100
	maximumReleaseCount   = 10000
	maximumReleasePage    = 8 << 20
	maximumReleaseBytes   = 64 << 20
)

// FetchReleases discovers published releases for each GitHub repository. The
// returned JSON object maps repository names to release arrays for Build.
// Authentication is sent only to api.github.com; redirects are rejected.
func FetchReleases(repositories []string, token string) ([]byte, error) {
	return fetchReleases(repositories, token, &http.Client{Timeout: 30 * time.Second})
}

func fetchReleases(repositories []string, token string, client *http.Client) ([]byte, error) {
	repositories = append([]string(nil), repositories...)
	sort.Strings(repositories)
	unique := repositories[:0]
	for _, repository := range repositories {
		if !validRepository(repository) {
			return nil, fmt.Errorf("invalid GitHub repository: %q", repository)
		}
		if len(unique) == 0 || unique[len(unique)-1] != repository {
			unique = append(unique, repository)
		}
	}
	if len(unique) > maximumRepositories {
		return nil, fmt.Errorf("too many GitHub repositories (maximum %d)", maximumRepositories)
	}
	// Copy the client so caller configuration stays intact. Even a redirect to a
	// different api.github.com path is rejected: only constructed URLs are used.
	bounded := *client
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result := make(map[string][]json.RawMessage, len(unique))
	totalBytes, totalReleases := 0, 0
	for _, repository := range unique {
		releases := []json.RawMessage{}
		for page := 1; ; page++ {
			if page > maximumReleasePages {
				return nil, fmt.Errorf("too many GitHub release pages for %s", repository)
			}
			endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d&page=%d", repository, githubReleasesPerPage, page)
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
			if err != nil {
				return nil, fmt.Errorf("create GitHub release request: %w", err)
			}
			request.Header.Set("Accept", "application/vnd.github+json")
			request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
			request.Header.Set("User-Agent", "karty-package-catalog")
			if token != "" {
				request.Header.Set("Authorization", "Bearer "+token)
			}
			body, err := githubReleasePage(&bounded, request)
			if err != nil {
				return nil, fmt.Errorf("fetch GitHub releases for %s (page %d): %w", repository, page, err)
			}
			totalBytes += len(body)
			if totalBytes > maximumReleaseBytes {
				return nil, fmt.Errorf("GitHub release data exceeds %d bytes", maximumReleaseBytes)
			}
			var items []json.RawMessage
			decoder := json.NewDecoder(bytes.NewReader(body))
			if err := decoder.Decode(&items); err != nil {
				return nil, fmt.Errorf("decode GitHub releases for %s: %w", repository, err)
			}
			if items == nil {
				return nil, fmt.Errorf("GitHub releases for %s must be an array", repository)
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				return nil, fmt.Errorf("GitHub releases for %s must contain one JSON document", repository)
			}
			if len(items) > githubReleasesPerPage {
				return nil, fmt.Errorf("GitHub release page for %s exceeds %d releases", repository, githubReleasesPerPage)
			}
			for _, item := range items {
				if item[0] != '{' {
					return nil, fmt.Errorf("invalid GitHub release for %s", repository)
				}
			}
			totalReleases += len(items)
			if totalReleases > maximumReleaseCount {
				return nil, fmt.Errorf("too many GitHub releases (maximum %d)", maximumReleaseCount)
			}
			releases = append(releases, items...)
			if len(items) < githubReleasesPerPage {
				break
			}
		}
		result[repository] = releases
	}
	return json.Marshal(result)
}

func githubReleasePage(client *http.Client, request *http.Request) ([]byte, error) {
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maximumReleasePage+1))
	if err != nil {
		return nil, fmt.Errorf("read GitHub release response: %w", err)
	}
	if len(body) > maximumReleasePage {
		return nil, fmt.Errorf("GitHub release page exceeds %d bytes", maximumReleasePage)
	}
	return body, nil
}
