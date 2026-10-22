// Package website generates the package directory as static HTML and Markdown.
package website

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/karty-game/karty-packages/internal/catalog"
	"golang.org/x/mod/semver"
)

//go:embed templates/*.html assets/*
var files embed.FS

const MaximumMasterBytes = 16 << 20

var (
	namePattern       = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)
	tagPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)
	repositoryPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)
	digestPattern     = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type versionView struct {
	catalog.Version
	ID      string
	Command string
}

type packageView struct {
	catalog.Package
	DisplayName     string
	Initials        string
	Category        string
	Search          string
	Command         string
	ReleaseVersions []versionView
}

type tagCount struct {
	Name  string
	Count int
}

type pageData struct {
	Title       string
	Description string
	Prefix      string
	Repository  string
	Packages    []packageView
	Tags        []tagCount
	Package     *packageView
	ReadyCount  int
}

// Generate validates master.json before writing a deterministic, self-contained
// site. It does not fetch package README files, execute package code or require a
// JavaScript toolchain. Images link to the previews provided by the catalog.
func Generate(masterJSON []byte, output string) error {
	master, err := decodeMaster(masterJSON)
	if err != nil {
		return err
	}
	pages := pageData{Title: "Packages", Description: "Reusable code and assets for your next Karty game.", Repository: master.Repository}
	tagCounts := make(map[string]int)
	for _, item := range master.Packages {
		entry := packageView{Package: item, DisplayName: displayName(item.Name), Initials: initials(item.Name)}
		if len(item.Tags) > 0 {
			entry.Category = item.Tags[0]
		} else {
			entry.Category = "package"
		}
		entry.Search = strings.ToLower(entry.Name + " " + entry.DisplayName + " " + entry.Description + " " + entry.Repository + " " + strings.Join(entry.Tags, " "))
		for _, tag := range entry.Tags {
			tagCounts[tag]++
		}
		for i, version := range item.Versions {
			entry.ReleaseVersions = append(entry.ReleaseVersions, versionView{Version: version, ID: fmt.Sprintf("install-%s-%d", item.Name, i), Command: installCommand(item.Name, version)})
			if item.Latest != nil && version.Version == *item.Latest {
				entry.Command = installCommand(item.Name, version)
			}
		}
		if item.Latest != nil {
			pages.ReadyCount++
		}
		pages.Packages = append(pages.Packages, entry)
	}
	for tag, count := range tagCounts {
		pages.Tags = append(pages.Tags, tagCount{Name: tag, Count: count})
	}
	sort.Slice(pages.Tags, func(i, j int) bool { return pages.Tags[i].Name < pages.Tags[j].Name })
	tmpl, err := template.New("site.html").Funcs(template.FuncMap{
		"join":      strings.Join,
		"shortHash": func(value string) string { return value[:12] },
		"tagQuery":  url.QueryEscape,
	}).ParseFS(files, "templates/site.html")
	if err != nil {
		return err
	}
	generated := make(map[string][]byte)
	render := func(name string, data pageData) error {
		var buffer bytes.Buffer
		if err := tmpl.Execute(&buffer, data); err != nil {
			return err
		}
		generated[name] = buffer.Bytes()
		return nil
	}
	if err := render("index.html", pages); err != nil {
		return err
	}
	generated["index.md"] = []byte(indexMarkdown(pages))
	for i := range pages.Packages {
		entry := &pages.Packages[i]
		detail := pages
		detail.Title, detail.Description, detail.Prefix, detail.Package = entry.DisplayName, entry.Description, "../", entry
		if err := render("packages/"+entry.Name+".html", detail); err != nil {
			return err
		}
		generated["packages/"+entry.Name+".md"] = []byte(packageMarkdown(*entry))
	}
	for _, name := range []string{"theme.css", "catalog.js", "favicon.svg"} {
		contents, err := files.ReadFile("assets/" + name)
		if err != nil {
			return err
		}
		generated["assets/"+name] = contents
	}
	generated[".nojekyll"] = nil
	generated["master.json"] = append(bytes.TrimSpace(masterJSON), '\n')
	return writeSite(output, generated)
}

func decodeMaster(contents []byte) (catalog.Master, error) {
	var master catalog.Master
	if len(contents) > MaximumMasterBytes {
		return master, fmt.Errorf("master.json exceeds %d bytes", MaximumMasterBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&master); err != nil {
		return master, fmt.Errorf("decode master.json: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return master, fmt.Errorf("master.json must contain one JSON document")
	}
	if master.FormatVersion != 1 || master.Packages == nil || !validRepository(master.Repository) {
		return master, fmt.Errorf("invalid master.json format or repository")
	}
	seen := make(map[string]bool)
	for i := range master.Packages {
		entry := &master.Packages[i]
		if entry.Repository == "" {
			entry.Repository = master.Repository
		}
		if !namePattern.MatchString(entry.Name) || seen[entry.Name] || !validRepository(entry.Repository) || strings.TrimSpace(entry.Description) == "" {
			return master, fmt.Errorf("invalid or duplicate package: %s", entry.Name)
		}
		seen[entry.Name] = true
		if !safeHTTPS(entry.Readme) || !safeHTTPS(entry.Source) || (entry.Preview != "" && !safeHTTPS(entry.Preview)) {
			return master, fmt.Errorf("invalid package links: %s", entry.Name)
		}
		tags := make(map[string]bool)
		for _, tag := range entry.Tags {
			if !tagPattern.MatchString(tag) || tags[tag] {
				return master, fmt.Errorf("invalid tags: %s", entry.Name)
			}
			tags[tag] = true
		}
		sort.Strings(entry.Tags)
		versions := make(map[string]bool)
		for _, version := range entry.Versions {
			core := strings.SplitN(strings.SplitN(version.Version, "+", 2)[0], "-", 2)[0]
			if len(version.Version) > 200 || !semver.IsValid("v"+version.Version) || strings.Count(core, ".") != 2 || versions[version.Version] || !digestPattern.MatchString(version.SHA256) || !safeHTTPS(version.Archive) || !safeHTTPS(version.ReleaseURL) {
				return master, fmt.Errorf("invalid release for %s: %s", entry.Name, version.Version)
			}
			parsed, _ := url.Parse(version.Archive)
			if !strings.HasSuffix(parsed.Path, ".kpkg") {
				return master, fmt.Errorf("invalid archive for %s", entry.Name)
			}
			versions[version.Version] = true
		}
		sort.Slice(entry.Versions, func(i, j int) bool {
			left, right := entry.Versions[i].Version, entry.Versions[j].Version
			if compared := semver.Compare("v"+left, "v"+right); compared != 0 {
				return compared > 0
			}
			return left < right
		})
		if (entry.Latest == nil && len(entry.Versions) > 0) || (entry.Latest != nil && (!versions[*entry.Latest] || *entry.Latest != entry.Versions[0].Version)) {
			return master, fmt.Errorf("invalid latest version for %s", entry.Name)
		}
	}
	sort.Slice(master.Packages, func(i, j int) bool { return master.Packages[i].Name < master.Packages[j].Name })
	return master, nil
}

func validRepository(value string) bool {
	if !repositoryPattern.MatchString(value) {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}

func safeHTTPS(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.RawQuery == "" && !parsed.ForceQuery && parsed.Fragment == "" && !strings.ContainsAny(value, "\r\n\x00")
}

func displayName(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '-' })
	for i, word := range words {
		switch word {
		case "fps", "ui", "ai", "2d", "3d":
			words[i] = strings.ToUpper(word)
		default:
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func initials(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '-' })
	if len(words) > 1 {
		return strings.ToUpper(words[0][:1] + words[1][:1])
	}
	return strings.ToUpper(name[:min(2, len(name))])
}

func writeSite(output string, generated map[string][]byte) error {
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	info, err := os.Lstat(output)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("site output must be a real directory")
	}
	root, err := os.OpenRoot(output)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, directory := range []string{"assets", "packages"} {
		if err := root.MkdirAll(directory, 0o755); err != nil {
			return err
		}
	}
	// Prune only package pages owned by this generator, retaining unrelated output.
	listing, err := fs.ReadDir(root.FS(), "packages")
	if err != nil {
		return err
	}
	for _, entry := range listing {
		name := "packages/" + entry.Name()
		ext := filepath.Ext(entry.Name())
		if !entry.IsDir() && (ext == ".html" || ext == ".md") && namePattern.MatchString(strings.TrimSuffix(entry.Name(), ext)) {
			if _, exists := generated[name]; !exists {
				previous, err := root.ReadFile(name)
				if err != nil {
					return err
				}
				if !bytes.HasPrefix(previous, []byte(generatedMarker)) {
					continue
				}
				if err := root.Remove(name); err != nil {
					return err
				}
			}
		}
	}
	for name, contents := range generated {
		if strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".md") {
			contents = append([]byte(generatedMarker), contents...)
		}
		if err := root.WriteFile(name, contents, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

const generatedMarker = "<!-- Generated by karty-site. -->\n"
