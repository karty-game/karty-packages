package catalog

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const externalRepository = "community/karty-props"

func externalMetadata(t *testing.T) []byte {
	t.Helper()
	var input metadata
	if err := json.Unmarshal(metadataJSON(t), &input); err != nil {
		t.Fatal(err)
	}
	input.Packages = append(input.Packages, packageMetadata{
		Name: "community-props", Repository: externalRepository,
		Description: "Community prototype props", Tags: []string{"3d"},
		Preview: "screenshots/props.png", Ref: "stable",
	})
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func releaseInRepository(name, version, repository string) *release {
	published := fixtureRelease(name, version)
	published.HTMLURL = strings.ReplaceAll(published.HTMLURL, testRepository, repository)
	published.Assets[0].BrowserDownloadURL = strings.ReplaceAll(published.Assets[0].BrowserDownloadURL, testRepository, repository)
	return published
}

func encodeReleases(t *testing.T, releases map[string][]*release) []byte {
	t.Helper()
	encoded, err := json.Marshal(releases)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestExternalRepositoryCatalog(t *testing.T) {
	repositories, err := Repositories(externalMetadata(t), testRepository)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repositories, []string{externalRepository, testRepository}) {
		t.Fatal(repositories)
	}
	releases := map[string][]*release{
		testRepository: {fixtureRelease("fps-kit", "0.1.0"), fixtureRelease("community-props", "9.0.0")},
		externalRepository: {
			releaseInRepository("community-props", "0.9.0", externalRepository),
			releaseInRepository("community-props", "0.10.0", externalRepository),
			releaseInRepository("fps-kit", "9.0.0", externalRepository),
		},
	}
	master, err := Build(externalMetadata(t), encodeReleases(t, releases), testRepository, nil)
	if err != nil {
		t.Fatal(err)
	}
	props := packageByName(t, master, "community-props")
	if props.Repository != externalRepository || props.Latest == nil || *props.Latest != "0.10.0" || len(props.Versions) != 2 {
		t.Fatalf("wrong external releases: %+v", props)
	}
	if props.Readme != "https://github.com/"+externalRepository+"/blob/stable/README.md" || props.Source != "https://github.com/"+externalRepository+"/tree/stable" || props.Preview != "https://raw.githubusercontent.com/"+externalRepository+"/stable/screenshots/props.png" {
		t.Fatalf("incorrect source links: %+v", props)
	}
	if !strings.HasPrefix(props.Versions[0].Archive, "https://github.com/"+externalRepository+"/releases/download/") || props.Versions[0].SHA256 != testChecksum {
		t.Fatal(props.Versions)
	}
	if kit := packageByName(t, master, "fps-kit"); kit.Latest == nil || *kit.Latest != "0.1.0" {
		t.Fatalf("external releases leaked into local package: %+v", kit)
	}
}

func TestExternalChecksumSidecar(t *testing.T) {
	published := releaseInRepository("community-props", "1.0.0", externalRepository)
	published.Assets[0].Digest = ""
	filename := published.Assets[0].Name
	address := published.Assets[0].BrowserDownloadURL + ".sha256"
	published.Assets = append(published.Assets, asset{Name: filename + ".sha256", BrowserDownloadURL: address})
	releases := map[string][]*release{testRepository: {}, externalRepository: {published}}
	_, err := Build(externalMetadata(t), encodeReleases(t, releases), testRepository, func(got string) (string, error) {
		if got != address {
			t.Fatalf("wrong repository for sidecar: %s", got)
		}
		return testChecksum + "  " + filename + "\n", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	published.Assets[1].BrowserDownloadURL = strings.ReplaceAll(address, externalRepository, testRepository)
	if _, err := Build(externalMetadata(t), encodeReleases(t, releases), testRepository, nil); err == nil {
		t.Fatal("accepted sidecar from different repository")
	}
}

func TestExternalRepositoryValidation(t *testing.T) {
	for _, input := range []string{
		`{"` + testRepository + `":[]}`,
		`{"` + testRepository + `":[],"` + externalRepository + `":null}`,
		`[]`,
	} {
		if _, err := Build(externalMetadata(t), []byte(input), testRepository, nil); err == nil {
			t.Fatalf("accepted missing external release data: %s", input)
		}
	}
	published := fixtureRelease("community-props", "1.0.0")
	releases := map[string][]*release{testRepository: {}, externalRepository: {published}}
	if _, err := Build(externalMetadata(t), encodeReleases(t, releases), testRepository, nil); err == nil {
		t.Fatal("accepted archive from the catalog repository for an external package")
	}
	for _, change := range []func(*packageMetadata){
		func(p *packageMetadata) { p.Repository = "../outside" },
		func(p *packageMetadata) { p.Repository = "https://github.com/owner/repo" },
		func(p *packageMetadata) { p.Source = "../outside" },
		func(p *packageMetadata) { p.Readme = "../README.md" },
		func(p *packageMetadata) { p.Ref = "../main" },
		func(p *packageMetadata) { p.Preview = "https://example.com/image.png" },
	} {
		var input metadata
		if err := json.Unmarshal(externalMetadata(t), &input); err != nil {
			t.Fatal(err)
		}
		change(&input.Packages[len(input.Packages)-1])
		encoded, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Repositories(encoded, testRepository); err == nil {
			t.Fatalf("accepted invalid metadata: %s", encoded)
		}
	}
}

func TestExternalMonorepoPaths(t *testing.T) {
	input := metadata{FormatVersion: 1, Packages: []packageMetadata{
		{Name: "props", Repository: externalRepository, Description: "Props", Source: "kits/props", Preview: "kits/props/preview.png", Ref: "release/assets"},
		{Name: "audio", Repository: externalRepository, Description: "Audio", Source: "kits/audio"},
	}}
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	repositories, err := Repositories(encoded, testRepository)
	if err != nil || !reflect.DeepEqual(repositories, []string{externalRepository}) {
		t.Fatalf("repositories=%v, error=%v", repositories, err)
	}
	master, err := Build(encoded, encodeReleases(t, map[string][]*release{externalRepository: {}}), testRepository, nil)
	if err != nil {
		t.Fatal(err)
	}
	props := packageByName(t, master, "props")
	if props.Readme != "https://github.com/"+externalRepository+"/blob/release%2Fassets/kits/props/README.md" {
		t.Fatal(props.Readme)
	}
	if props.Latest != nil || len(props.Versions) != 0 {
		t.Fatal("invented external version")
	}
	input.Packages[0].Readme = "README.md"
	input.Packages[0].Preview = "screenshots/props.png"
	encoded, err = json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	master, err = Build(encoded, encodeReleases(t, map[string][]*release{externalRepository: {}}), testRepository, nil)
	if err != nil {
		t.Fatal(err)
	}
	props = packageByName(t, master, "props")
	if props.Readme != "https://github.com/"+externalRepository+"/blob/release%2Fassets/README.md" || props.Preview != "https://raw.githubusercontent.com/"+externalRepository+"/release%2Fassets/screenshots/props.png" {
		t.Fatalf("repo-root metadata paths: %+v", props)
	}
}

func TestRepositoryCasing(t *testing.T) {
	published := releaseInRepository("community-props", "1.0.0", "Community/Karty-Props")
	releases := map[string][]*release{testRepository: {}, externalRepository: {published}}
	master, err := Build(externalMetadata(t), encodeReleases(t, releases), testRepository, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := packageByName(t, master, "community-props").Versions[0].Archive; !strings.HasPrefix(got, "https://github.com/"+externalRepository+"/") {
		t.Fatal(got)
	}
	published.Assets[0].BrowserDownloadURL = strings.ReplaceAll(published.Assets[0].BrowserDownloadURL, "community-props-v", "COMMUNITY-PROPS-v")
	if _, err := Build(externalMetadata(t), encodeReleases(t, releases), testRepository, nil); err == nil {
		t.Fatal("accepted release tag with different casing")
	}
}
