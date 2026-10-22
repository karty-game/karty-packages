//go:build ignore

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerationPreservesManifestVersion(t *testing.T) {
	props, err := os.ReadFile("props.yaml")
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	for _, directory := range []string{"packages/fps-kit", "scripts/fps-kit"} {
		if err := os.MkdirAll(filepath.Join(fixture, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(fixture, "scripts/fps-kit/props.yaml"), props, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(fixture)
	manifest := filepath.Join(root, "karty-package.toml")
	if err := os.WriteFile(manifest, []byte("format_version = 1\nname = 'fps-kit'\nversion = '0.0.1'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	generate(false)
	version, err := packageVersion(manifest)
	if err != nil || version != "0.0.1" {
		t.Fatalf("generation changed package version: %q, %v", version, err)
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `version = "0.0.1"`, `version = "2.4.6-rc.1"`, 1))
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	// Changing only the release version must not invalidate any generated asset.
	generate(true)
	version, err = packageVersion(manifest)
	if err != nil || version != "2.4.6-rc.1" {
		t.Fatalf("checking assets changed package version: %q, %v", version, err)
	}
}

func TestPackageVersionRejectsInvalidMetadata(t *testing.T) {
	for _, document := range []string{
		"name = 'fps-kit'\n", "name = 'other'\nversion = '0.0.1'\n",
		"name = 'fps-kit'\nversion = '0.1'\n", "name = 'fps-kit'\nversion = '../0.0.1'\n",
		"name = 'fps-kit'\nversion = '01.0.0'\n", "invalid TOML",
	} {
		path := filepath.Join(t.TempDir(), "karty-package.toml")
		if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := packageVersion(path); err == nil {
			t.Fatalf("accepted invalid metadata %q", document)
		}
	}
}
