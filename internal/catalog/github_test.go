package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

type githubRoundTripper func(*http.Request) (*http.Response, error)

func (transport githubRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func githubTestResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestFetchReleasesSortedRepositories(t *testing.T) {
	var visited []string
	client := &http.Client{Transport: githubRoundTripper(func(request *http.Request) (*http.Response, error) {
		visited = append(visited, request.URL.String())
		if request.Method != http.MethodGet || request.URL.Scheme != "https" || request.URL.Host != "api.github.com" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL)
		}
		if request.Header.Get("Authorization") != "Bearer test-secret" || request.Header.Get("Accept") != "application/vnd.github+json" || request.Header.Get("X-GitHub-Api-Version") != "2022-11-28" || request.Header.Get("User-Agent") == "" {
			t.Fatalf("missing GitHub request headers: %v", request.Header)
		}
		if deadline, ok := request.Context().Deadline(); !ok || time.Until(deadline) > 2*time.Minute {
			t.Fatal("request has no bounded deadline")
		}
		return githubTestResponse(http.StatusOK, `[{"tag_name":"kit-v1.0.0","assets":[]}]`), nil
	})}
	input := []string{"z/project", "a/project", "z/project"}
	actual, err := fetchReleases(input, "test-secret", client)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"https://api.github.com/repos/a/project/releases?per_page=100&page=1",
		"https://api.github.com/repos/z/project/releases?per_page=100&page=1",
	}
	if !reflect.DeepEqual(visited, want) || !reflect.DeepEqual(input, []string{"z/project", "a/project", "z/project"}) {
		t.Fatalf("requests = %v; input = %v", visited, input)
	}
	if string(actual) != `{"a/project":[{"tag_name":"kit-v1.0.0","assets":[]}],"z/project":[{"tag_name":"kit-v1.0.0","assets":[]}]}` {
		t.Fatalf("unexpected deterministic JSON: %s", actual)
	}
}

func TestFetchReleasesPagination(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: githubRoundTripper(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.Query().Get("page") != fmt.Sprint(calls) {
			t.Fatalf("unexpected page: %s", request.URL)
		}
		if request.Header.Get("Authorization") != "" {
			t.Fatal("anonymous discovery added authentication")
		}
		if calls == 1 {
			response := githubTestResponse(http.StatusOK, githubFullPage())
			// Pagination never follows server-controlled URLs, including Link.
			response.Header.Set("Link", `<https://other.example/steal>; rel="next"`)
			return response, nil
		}
		return githubTestResponse(http.StatusOK, `[{"tag_name":"last"}]`), nil
	})}
	data, err := fetchReleases([]string{"owner/project"}, "", client)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string][]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(result["owner/project"]) != 101 {
		t.Fatalf("calls = %d; releases = %d", calls, len(result["owner/project"]))
	}
}

func TestFetchReleasesRejectsRedirects(t *testing.T) {
	calls := 0
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			t.Fatal("discovery used the caller's redirect policy")
			return nil
		},
		Transport: githubRoundTripper(func(request *http.Request) (*http.Response, error) {
			calls++
			if request.URL.Host != "api.github.com" {
				t.Fatal("authentication followed a redirect")
			}
			response := githubTestResponse(http.StatusFound, "")
			response.Header.Set("Location", "https://other.example/releases")
			return response, nil
		}),
	}
	if _, err := fetchReleases([]string{"owner/project"}, "secret", client); err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("redirect error = %v", err)
	}
	if calls != 1 || client.CheckRedirect == nil {
		t.Fatalf("redirect requests = %d; client was mutated", calls)
	}
}

func TestFetchReleasesInvalidRepository(t *testing.T) {
	client := &http.Client{Transport: githubRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid repository reached the network")
		return nil, nil
	})}
	for _, repository := range []string{"../project", "owner/..", "owner/project/extra", "owner/project?token=value", "https://github.com/owner/project", "owner/"} {
		t.Run(repository, func(t *testing.T) {
			if _, err := fetchReleases([]string{repository}, "secret", client); err == nil {
				t.Fatal("accepted invalid repository")
			}
		})
	}
	repositories := make([]string, maximumRepositories+1)
	for i := range repositories {
		repositories[i] = fmt.Sprintf("owner/project-%d", i)
	}
	if _, err := fetchReleases(repositories, "", client); err == nil || !strings.Contains(err.Error(), "too many GitHub repositories") {
		t.Fatalf("repository bound error = %v", err)
	}
}

func TestFetchReleasesEmptyRepositories(t *testing.T) {
	data, err := fetchReleases(nil, "", &http.Client{})
	if err != nil || string(data) != "{}" {
		t.Fatalf("empty discovery = %s, %v", data, err)
	}
}

func TestFetchReleasesResponseErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"not-found", 404, "private server error", "HTTP 404"},
		{"rate-limit", 403, "rate limit details", "HTTP 403"},
		{"invalid-json", 200, "[", "decode GitHub releases"},
		{"object", 200, "{}", "decode GitHub releases"},
		{"null", 200, "null", "must be an array"},
		{"trailing-json", 200, "[] []", "one JSON document"},
		{"trailing-invalid", 200, "[] x", "one JSON document"},
		{"null-entry", 200, "[null]", "invalid GitHub release"},
		{"number-entry", 200, "[1]", "invalid GitHub release"},
		{"extra-items", 200, "[{}" + strings.Repeat(",{}", 100) + "]", "exceeds 100 releases"},
		{"oversized", 200, strings.Repeat(" ", maximumReleasePage+1), "exceeds"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			client := &http.Client{Transport: githubRoundTripper(func(*http.Request) (*http.Response, error) {
				return githubTestResponse(item.status, item.body), nil
			})}
			_, err := fetchReleases([]string{"owner/project"}, "", client)
			if err == nil || !strings.Contains(err.Error(), item.want) {
				t.Fatalf("error = %v; want %q", err, item.want)
			}
		})
	}
}

func TestFetchReleasesPageBound(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: githubRoundTripper(func(*http.Request) (*http.Response, error) {
		calls++
		return githubTestResponse(http.StatusOK, githubFullPage()), nil
	})}
	_, err := fetchReleases([]string{"owner/project"}, "", client)
	if err == nil || !strings.Contains(err.Error(), "too many GitHub release pages") || calls != maximumReleasePages {
		t.Fatalf("page limit: calls = %d, error = %v", calls, err)
	}
}

func TestFetchReleasesCountBound(t *testing.T) {
	client := &http.Client{Transport: githubRoundTripper(func(request *http.Request) (*http.Response, error) {
		if strings.Contains(request.URL.Path, "/a/") && request.URL.Query().Get("page") == "100" {
			return githubTestResponse(http.StatusOK, "[]"), nil
		}
		return githubTestResponse(http.StatusOK, githubFullPage()), nil
	})}
	_, err := fetchReleases([]string{"a/project", "b/project"}, "", client)
	if err == nil || !strings.Contains(err.Error(), "too many GitHub releases") {
		t.Fatalf("release count limit error = %v", err)
	}
}

func TestFetchReleasesTransportError(t *testing.T) {
	client := &http.Client{Transport: githubRoundTripper(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport unavailable")
	})}
	if _, err := fetchReleases([]string{"owner/project"}, "", client); err == nil || !strings.Contains(err.Error(), "transport unavailable") {
		t.Fatalf("transport error = %v", err)
	}
}

func githubFullPage() string {
	return "[{}" + strings.Repeat(",{}", githubReleasesPerPage-1) + "]"
}
