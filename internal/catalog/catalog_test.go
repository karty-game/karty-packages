package catalog

import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

const testRepository = "karty-game/karty-packages"

var testChecksum = strings.Repeat("a", 64)

func metadataJSON(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile("../../catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func fixtureRelease(name, version string) *release {
	tag := name + "-v" + version
	filename := name + "-" + version + ".kpkg"
	return &release{TagName: tag, HTMLURL: "https://github.com/" + testRepository + "/releases/tag/" + tag, Assets: []asset{{Name: filename, BrowserDownloadURL: "https://github.com/" + testRepository + "/releases/download/" + tag + "/" + filename, Digest: "sha256:" + testChecksum}}}
}

func buildFixture(t *testing.T, releases []*release, fetch FetchText) (Master, error) {
	t.Helper()
	if releases == nil {
		releases = []*release{}
	}
	encoded, err := json.Marshal(releases)
	if err != nil {
		t.Fatal(err)
	}
	return Build(metadataJSON(t), encoded, testRepository, fetch)
}

func packageByName(t *testing.T, master Master, name string) Package {
	t.Helper()
	for _, entry := range master.Packages {
		if entry.Name == name {
			return entry
		}
	}
	t.Fatalf("missing package %s", name)
	return Package{}
}

func TestUnreleasedCatalog(t *testing.T) {
	master, err := buildFixture(t, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, entry := range master.Packages {
		names = append(names, entry.Name)
		if entry.Latest != nil || len(entry.Versions) != 0 {
			t.Fatalf("invented version: %+v", entry)
		}
		if entry.Source != "https://github.com/"+testRepository+"/tree/main/packages/"+entry.Name {
			t.Fatal(entry.Source)
		}
	}
	if !reflect.DeepEqual(names, []string{"controllers-3d", "controllers-isometric", "controllers-topdown", "dialogue", "fps-kit", "savegame", "tiled"}) {
		t.Fatal(names)
	}
	encoded, err := json.Marshal(master)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"versions":null`) || !strings.Contains(string(encoded), `"latest":null`) {
		t.Fatal(string(encoded))
	}
	if packageByName(t, master, "fps-kit").Preview != "https://raw.githubusercontent.com/"+testRepository+"/main/packages/fps-kit/screenshots/materials.png" {
		t.Fatal("missing preview")
	}
}

func TestControllerPackagesReleaseIndependently(t *testing.T) {
	names := []string{"controllers-3d", "controllers-isometric", "controllers-topdown"}
	// A historical combined release must never become a split package's version.
	releases := []*release{fixtureRelease("controllers", "9.0.0"), fixtureRelease(names[0], "0.0.1"), fixtureRelease(names[1], "0.0.2")}
	master, err := buildFixture(t, releases, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range names {
		entry := packageByName(t, master, name)
		if i == 2 {
			if entry.Latest != nil || len(entry.Versions) != 0 {
				t.Fatal("unreleased controller inherited a version", entry)
			}
		} else if entry.Latest == nil || *entry.Latest != fmt.Sprintf("0.0.%d", i+1) || len(entry.Versions) != 1 {
			t.Fatal("controller release mixed with another package", entry)
		}
		path := "packages/" + name + "/previews/controller.jpg"
		if entry.Preview != "https://raw.githubusercontent.com/"+testRepository+"/main/"+path {
			t.Fatal("missing controller preview", entry.Preview)
		}
		preview, err := os.Open("../../" + path)
		if err != nil {
			t.Fatal(err)
		}
		config, decodeErr := jpeg.DecodeConfig(preview)
		closeErr := preview.Close()
		if decodeErr != nil || closeErr != nil || config.Width < 1 || config.Width*9 != config.Height*16 {
			t.Fatalf("invalid landscape preview for %s: %+v, %v, %v", name, config, decodeErr, closeErr)
		}
	}
}

func TestPublishedVersionsAndDeterminism(t *testing.T) {
	releases := []*release{fixtureRelease("fps-kit", "0.9.0"), fixtureRelease("fps-kit", "0.10.0"), fixtureRelease("fps-kit", "0.2.0"), fixtureRelease("tiled", "2.0.0")}
	noFetch := func(string) (string, error) { return "", fmt.Errorf("unexpected network") }
	first, err := buildFixture(t, releases, noFetch)
	if err != nil {
		t.Fatal(err)
	}
	for i, j := 0, len(releases)-1; i < j; i, j = i+1, j-1 {
		releases[i], releases[j] = releases[j], releases[i]
	}
	second, err := buildFixture(t, releases, noFetch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("release input order changes catalog")
	}
	kit := packageByName(t, first, "fps-kit")
	if kit.Latest == nil || *kit.Latest != "0.10.0" {
		t.Fatal(kit.Latest)
	}
	versions := []string{}
	for _, version := range kit.Versions {
		versions = append(versions, version.Version)
		if version.SHA256 != testChecksum {
			t.Fatal(version)
		}
	}
	if !reflect.DeepEqual(versions, []string{"0.10.0", "0.9.0", "0.2.0"}) {
		t.Fatal(versions)
	}
}

func TestExcludedReleases(t *testing.T) {
	draft := fixtureRelease("fps-kit", "1.0.0")
	draft.Draft = true
	draft.Assets = nil
	pre := fixtureRelease("fps-kit", "2.0.0")
	pre.Prerelease = true
	pre.Assets = nil
	master, err := buildFixture(t, []*release{draft, pre, fixtureRelease("fps-kit", "3.0.0-rc.1"), fixtureRelease("unknown", "1.0.0")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range master.Packages {
		if entry.Latest != nil {
			t.Fatal(entry)
		}
	}
}

func TestChecksumSidecar(t *testing.T) {
	published := fixtureRelease("fps-kit", "0.1.0")
	published.Assets[0].Digest = ""
	filename := published.Assets[0].Name
	address := published.Assets[0].BrowserDownloadURL + ".sha256"
	published.Assets = append(published.Assets, asset{Name: filename + ".sha256", BrowserDownloadURL: address})
	master, err := buildFixture(t, []*release{published}, func(got string) (string, error) {
		if got != address {
			t.Errorf("requested %q", got)
		}
		return testChecksum + "  " + filename + "\n", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if packageByName(t, master, "fps-kit").Versions[0].SHA256 != testChecksum {
		t.Fatal(master)
	}
	for _, contents := range []string{testChecksum + "  tiled-0.1.0.kpkg\n", testChecksum + "  ../" + filename + "\n", strings.ToUpper(testChecksum) + "  " + filename + "\n", testChecksum + "  " + filename + "\nextra\n", strings.Repeat("x", 4097)} {
		if _, err := buildFixture(t, []*release{published}, func(string) (string, error) { return contents, nil }); err == nil {
			t.Fatalf("accepted invalid sidecar %q", contents)
		}
	}
}

func TestInvalidPublishedReleases(t *testing.T) {
	cases := map[string]func(*release){
		"bad semver":               func(r *release) { r.TagName = "fps-kit-v01.0.0" },
		"abbreviated semver":       func(r *release) { r.TagName = "fps-kit-v1.0" },
		"bad prerelease":           func(r *release) { r.TagName = "fps-kit-v1.0.0-01" },
		"bad digest":               func(r *release) { r.Assets[0].Digest = "sha256:invalid" },
		"uppercase digest":         func(r *release) { r.Assets[0].Digest = "sha256:" + strings.ToUpper(testChecksum) },
		"wrong digest":             func(r *release) { r.Assets[0].Digest = "sha512:" + testChecksum },
		"missing checksum":         func(r *release) { r.Assets[0].Digest = "" },
		"wrong filename":           func(r *release) { r.Assets[0].Name = "fps-kit-0.2.0.kpkg" },
		"duplicate asset":          func(r *release) { r.Assets = append(r.Assets, r.Assets[0]) },
		"wrong release repository": func(r *release) { r.HTMLURL = strings.Replace(r.HTMLURL, testRepository, "other/repository", 1) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			published := fixtureRelease("fps-kit", "0.1.0")
			mutate(published)
			if _, err := buildFixture(t, []*release{published}, nil); err == nil {
				t.Fatal("accepted invalid release")
			}
		})
	}
	repeated := fixtureRelease("fps-kit", "0.1.0")
	if _, err := buildFixture(t, []*release{repeated, repeated}, nil); err == nil {
		t.Fatal("accepted duplicate version")
	}
}

func TestReleaseURLConfinement(t *testing.T) {
	valid := fixtureRelease("fps-kit", "0.1.0").Assets[0].BrowserDownloadURL
	for _, address := range []string{
		strings.Replace(valid, "https:", "http:", 1),
		strings.Replace(valid, testRepository, "other/repository", 1),
		strings.Replace(valid, "download/fps-kit-v0.1.0/", "download/tiled-v0.1.0/", 1),
		strings.Replace(valid, "/fps-kit-0.1.0.kpkg", "/%2e%2e%2ffps-kit-0.1.0.kpkg", 1),
		strings.Replace(valid, "github.com", "user:password@github.com", 1),
		strings.Replace(valid, "github.com", "github.com:443", 1),
		valid + "?token=bad", valid + "#fragment", valid + "?",
	} {
		published := fixtureRelease("fps-kit", "0.1.0")
		published.Assets[0].BrowserDownloadURL = address
		if _, err := buildFixture(t, []*release{published}, nil); err == nil {
			t.Fatalf("accepted %s", address)
		}
	}
}

func TestMetadataValidation(t *testing.T) {
	var original metadata
	if err := json.Unmarshal(metadataJSON(t), &original); err != nil {
		t.Fatal(err)
	}
	changes := map[string]func(*metadata){
		"unsafe name":       func(m *metadata) { m.Packages[0].Name = "../controllers" },
		"traversal":         func(m *metadata) { m.Packages[0].Readme = "packages/controllers/../../README.md" },
		"external preview":  func(m *metadata) { m.Packages[0].Preview = "https://example.com/image.png" },
		"unsafe source":     func(m *metadata) { m.Packages[0].Source = "packages/../dialogue" },
		"duplicate package": func(m *metadata) { m.Packages = append(m.Packages, m.Packages[0]) },
		"duplicate tags":    func(m *metadata) { m.Packages[0].Tags = []string{"input", "input"} },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			var input metadata
			if err := json.Unmarshal(metadataJSON(t), &input); err != nil {
				t.Fatal(err)
			}
			change(&input)
			encoded, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Build(encoded, []byte("[]"), testRepository, nil); err == nil {
				t.Fatal("accepted invalid metadata")
			}
		})
	}
	for _, repository := range []string{"../outside", "a/..", "https://github.com/a/b", "a/b/c"} {
		if _, err := Build(metadataJSON(t), []byte("[]"), repository, nil); err == nil {
			t.Fatal(repository)
		}
	}
	for _, encoded := range []string{`{"format_version":1,"packages":[],"unknown":true}`, `{"format_version":1,"packages":[{"name":"test","description":"test","unknown":true}]}`, `{"format_version":1,"packages":null}`, `{"format_version":1,"packages":[]} {}`} {
		if _, err := Build([]byte(encoded), []byte("[]"), testRepository, nil); err == nil {
			t.Fatal(encoded)
		}
	}
	for _, encoded := range []string{"null", "{}", "[null]"} {
		if _, err := Build(metadataJSON(t), []byte(encoded), testRepository, nil); err == nil {
			t.Fatal(encoded)
		}
	}
}

func TestDownloadRedirectPolicy(t *testing.T) {
	for _, address := range []string{"https://github.com/file", "https://release-assets.githubusercontent.com/file?signature=value", "https://objects.githubusercontent.com/file"} {
		parsed, err := url.Parse(address)
		if err != nil {
			t.Fatal(err)
		}
		if !allowedDownloadURL(parsed) {
			t.Fatal(address)
		}
	}
	for _, address := range []string{"http://github.com/file", "https://github.com.evil.invalid/file", "https://github.com:443/file", "https://user:password@github.com/file", "https://127.0.0.1/file", "https://github.com/file#fragment"} {
		parsed, err := url.Parse(address)
		if err != nil {
			t.Fatal(err)
		}
		if allowedDownloadURL(parsed) {
			t.Fatal(address)
		}
	}
}
