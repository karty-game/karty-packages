// Package catalog builds public discovery from curated packages and published releases.
package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const maximumChecksumBytes = 4096

var (
	namePattern       = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)
	tagPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)
	repositoryPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)
	pathPartPattern   = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
	digestPattern     = regexp.MustCompile(`^[a-f0-9]{64}$`)
	checksumPattern   = regexp.MustCompile(`^([a-f0-9]{64})[ \t]+\*?([^\r\n]+)\r?\n?$`)
	previewPattern    = regexp.MustCompile(`\.(png|jpe?g|webp|gif|svg)$`)
)

type metadata struct {
	FormatVersion int               `json:"format_version"`
	Packages      []packageMetadata `json:"packages"`
}

type packageMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Readme      string   `json:"readme"`
	Source      string   `json:"source"`
	Preview     string   `json:"preview"`
}

// Master contains only immutable published versions; an unreleased package has a nil Latest.
type Master struct {
	FormatVersion int       `json:"format_version"`
	Repository    string    `json:"repository"`
	Packages      []Package `json:"packages"`
}

type Package struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Readme      string    `json:"readme"`
	Source      string    `json:"source"`
	Preview     string    `json:"preview,omitempty"`
	Latest      *string   `json:"latest"`
	Versions    []Version `json:"versions"`
}

type Version struct {
	Version    string `json:"version"`
	Archive    string `json:"archive"`
	SHA256     string `json:"sha256"`
	ReleaseURL string `json:"release_url"`
}

type release struct {
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	TagName    string  `json:"tag_name"`
	HTMLURL    string  `json:"html_url"`
	Assets     []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

// FetchText can supply checksum sidecars in offline callers and tests. A nil value
// uses a bounded HTTPS downloader restricted to GitHub release asset hosts.
type FetchText func(string) (string, error)

// Build validates both documents before constructing a deterministic catalog.
func Build(catalogJSON, releasesJSON []byte, repository string, fetch FetchText) (Master, error) {
	result := Master{FormatVersion: 1, Repository: repository, Packages: []Package{}}
	if !repositoryPattern.MatchString(repository) {
		return result, fmt.Errorf("invalid GitHub repository")
	}
	for _, part := range strings.Split(repository, "/") {
		if part == "." || part == ".." {
			return result, fmt.Errorf("invalid GitHub repository")
		}
	}
	var input metadata
	decoder := json.NewDecoder(bytes.NewReader(catalogJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return result, fmt.Errorf("decode catalog: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return result, fmt.Errorf("catalog must contain one JSON document")
	}
	if input.FormatVersion != 1 || input.Packages == nil {
		return result, fmt.Errorf("unsupported catalog format")
	}
	byName := make(map[string]int)
	for _, item := range input.Packages {
		entry, err := publicPackage(item, repository)
		if err != nil {
			return result, err
		}
		if _, exists := byName[entry.Name]; exists {
			return result, fmt.Errorf("duplicate catalog package name: %s", entry.Name)
		}
		byName[entry.Name] = len(result.Packages)
		result.Packages = append(result.Packages, entry)
	}
	sort.Slice(result.Packages, func(i, j int) bool { return result.Packages[i].Name < result.Packages[j].Name })
	var releases []*release
	if err := json.Unmarshal(releasesJSON, &releases); err != nil {
		return result, fmt.Errorf("decode GitHub releases: %w", err)
	}
	if releases == nil {
		return result, fmt.Errorf("GitHub releases must be an array")
	}
	if fetch == nil {
		fetch = fetchChecksumText
	}
	for _, published := range releases {
		if published == nil {
			return result, fmt.Errorf("invalid GitHub release")
		}
		if published.Draft || published.Prerelease {
			continue
		}
		if published.TagName == "" {
			return result, fmt.Errorf("release is missing its tag")
		}
		for i := range result.Packages {
			entry := &result.Packages[i]
			prefix := entry.Name + "-v"
			if !strings.HasPrefix(published.TagName, prefix) {
				continue
			}
			version := strings.TrimPrefix(published.TagName, prefix)
			// x/mod accepts abbreviated versions, while package archives require x.y.z.
			if len(version) > 200 || !semver.IsValid("v"+version) || strings.Count(strings.SplitN(strings.SplitN(version, "+", 2)[0], "-", 2)[0], ".") != 2 {
				return result, fmt.Errorf("invalid package version: %s", version)
			}
			if semver.Prerelease("v"+version) != "" {
				break
			}
			for _, existing := range entry.Versions {
				if existing.Version == version {
					return result, fmt.Errorf("duplicate release version for %s: %s", entry.Name, version)
				}
			}
			filename := entry.Name + "-" + version + ".kpkg"
			archiveAsset, err := namedAsset(published.Assets, filename)
			if err != nil {
				return result, err
			}
			archiveURL, err := releaseURL(archiveAsset.BrowserDownloadURL, repository, "download", published.TagName, filename)
			if err != nil {
				return result, err
			}
			pageURL, err := releaseURL(published.HTMLURL, repository, "tag", published.TagName, "")
			if err != nil {
				return result, err
			}
			checksum, err := archiveChecksum(archiveAsset, published.Assets, repository, published.TagName, filename, fetch)
			if err != nil {
				return result, err
			}
			entry.Versions = append(entry.Versions, Version{Version: version, Archive: archiveURL, SHA256: checksum, ReleaseURL: pageURL})
			break
		}
	}
	for i := range result.Packages {
		entry := &result.Packages[i]
		sort.Slice(entry.Versions, func(i, j int) bool {
			left, right := entry.Versions[i].Version, entry.Versions[j].Version
			if compared := semver.Compare("v"+left, "v"+right); compared != 0 {
				return compared > 0
			}
			return left < right
		})
		if len(entry.Versions) > 0 {
			entry.Latest = &entry.Versions[0].Version
		}
	}
	return result, nil
}

func publicPackage(item packageMetadata, repository string) (Package, error) {
	entry := Package{Name: item.Name, Description: strings.TrimSpace(item.Description), Tags: append([]string{}, item.Tags...), Versions: []Version{}}
	if !namePattern.MatchString(item.Name) {
		return entry, fmt.Errorf("invalid catalog package name")
	}
	if entry.Description == "" || len(item.Description) > 500 {
		return entry, fmt.Errorf("invalid description for %s", item.Name)
	}
	if len(item.Tags) > 20 {
		return entry, fmt.Errorf("invalid tags for %s", item.Name)
	}
	seen := make(map[string]bool)
	for _, tag := range item.Tags {
		if !tagPattern.MatchString(tag) || seen[tag] {
			return entry, fmt.Errorf("invalid or duplicate tags for %s", item.Name)
		}
		seen[tag] = true
	}
	sort.Strings(entry.Tags)
	source := "packages/" + item.Name
	if item.Source != "" && item.Source != source {
		return entry, fmt.Errorf("invalid source path for %s", item.Name)
	}
	readme := item.Readme
	if readme == "" {
		readme = source + "/README.md"
	}
	if !packagePath(readme, item.Name) || !strings.HasSuffix(readme, ".md") {
		return entry, fmt.Errorf("invalid README path for %s", item.Name)
	}
	entry.Readme = "https://github.com/" + repository + "/blob/main/" + readme
	entry.Source = "https://github.com/" + repository + "/tree/main/" + source
	if item.Preview != "" {
		if !packagePath(item.Preview, item.Name) || !previewPattern.MatchString(item.Preview) {
			return entry, fmt.Errorf("invalid preview for %s", item.Name)
		}
		entry.Preview = "https://raw.githubusercontent.com/" + repository + "/main/" + item.Preview
	}
	return entry, nil
}

func packagePath(value, name string) bool {
	if !strings.HasPrefix(value, "packages/"+name+"/") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if !pathPartPattern.MatchString(part) || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func releaseURL(value, repository, kind, tag, filename string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid release URL: %w", err)
	}
	expected := "/" + repository + "/releases/" + kind + "/" + tag
	canonical := "https://github.com/" + repository + "/releases/" + kind + "/" + url.PathEscape(tag)
	if filename != "" {
		expected += "/" + filename
		canonical += "/" + url.PathEscape(filename)
	}
	if parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Path != expected {
		return "", fmt.Errorf("release URL must match expected GitHub repository, tag and asset: %s", value)
	}
	return canonical, nil
}

func namedAsset(assets []asset, name string) (asset, error) {
	var found asset
	count := 0
	for _, candidate := range assets {
		if candidate.Name == name {
			found = candidate
			count++
		}
	}
	if count != 1 {
		return found, fmt.Errorf("expected one release asset named %s", name)
	}
	return found, nil
}

func archiveChecksum(archive asset, assets []asset, repository, tag, filename string, fetch FetchText) (string, error) {
	if archive.Digest != "" {
		if !strings.HasPrefix(archive.Digest, "sha256:") || !digestPattern.MatchString(strings.TrimPrefix(archive.Digest, "sha256:")) {
			return "", fmt.Errorf("invalid archive checksum for %s", filename)
		}
		return strings.TrimPrefix(archive.Digest, "sha256:"), nil
	}
	sidecar, err := namedAsset(assets, filename+".sha256")
	if err != nil {
		return "", err
	}
	address, err := releaseURL(sidecar.BrowserDownloadURL, repository, "download", tag, filename+".sha256")
	if err != nil {
		return "", err
	}
	contents, err := fetch(address)
	if err != nil {
		return "", fmt.Errorf("fetch checksum: %w", err)
	}
	if len(contents) > maximumChecksumBytes {
		return "", fmt.Errorf("checksum sidecar exceeds bounds")
	}
	match := checksumPattern.FindStringSubmatch(contents)
	if match == nil || match[2] != filename {
		return "", fmt.Errorf("invalid checksum sidecar for %s", filename)
	}
	return match[1], nil
}

func allowedDownloadURL(address *url.URL) bool {
	if address.Scheme != "https" || address.User != nil || address.Port() != "" || address.Fragment != "" {
		return false
	}
	switch address.Host {
	case "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return true
	default:
		return false
	}
}

func fetchChecksumText(address string) (string, error) {
	initial, err := url.Parse(address)
	if err != nil || !allowedDownloadURL(initial) {
		return "", fmt.Errorf("unsupported checksum download URL")
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return fmt.Errorf("too many checksum redirects")
		}
		if !allowedDownloadURL(req.URL) {
			return fmt.Errorf("unsupported checksum download redirect")
		}
		return nil
	}}
	response, err := client.Get(address)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("checksum download failed: HTTP %d", response.StatusCode)
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, maximumChecksumBytes+1))
	if err != nil {
		return "", err
	}
	if len(contents) > maximumChecksumBytes {
		return "", fmt.Errorf("checksum sidecar exceeds bounds")
	}
	return string(contents), nil
}
