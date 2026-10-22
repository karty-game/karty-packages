package kpkg

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(name, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func fixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	directory := filepath.Join(root, "packages/sample-kit")
	writeFixture(t, filepath.Join(directory, ManifestName), []byte("format_version=1\nname='sample-kit'\nversion='0.1.0'\n"))
	writeFixture(t, filepath.Join(directory, "go.mod"), []byte("module example.com/sample-kit\n\ngo 1.27.0\n"))
	writeFixture(t, filepath.Join(directory, "main.go"), []byte("package sample\n"))
	writeFixture(t, filepath.Join(directory, "README.md"), []byte("Sample source package.\n"))

	return root, directory
}

func TestArchiveMatchesDeterministicInventoryContract(t *testing.T) {
	t.Parallel()
	root, directory := fixture(t)
	writeFixture(t, filepath.Join(directory, "dist/ignored"), []byte("generated output"))
	writeFixture(t, filepath.Join(directory, ".git/ignored"), []byte("private metadata"))
	commit := strings.Repeat("a", 40)
	first, err := Pack(t.Context(), root, "sample-kit", filepath.Join(root, "first"), commit)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(filepath.Join(directory, "main.go"), 0700); err != nil {
		t.Fatal(err)
	}

	second, err := Pack(t.Context(), root, "sample-kit-v0.1.0", filepath.Join(root, "second"), commit)
	if err != nil || first.SHA256 != second.SHA256 || !bytes.Equal(readFixture(t, first.Files.Archive), readFixture(t, second.Files.Archive)) {
		t.Fatalf("archive depends on output location or source permissions: %v", err)
	}

	encoded := readFixture(t, first.Files.Archive)
	if first.SHA256 != Digest(encoded) || string(readFixture(t, first.Files.Checksum)) != first.SHA256+"  "+first.Archive+"\n" {
		t.Fatal("archive checksum or sidecar format differs")
	}

	var metadata map[string]any
	if err := json.Unmarshal(readFixture(t, first.Files.Metadata), &metadata); err != nil {
		t.Fatal(err)
	}

	if metadata["source_commit"] != commit || metadata["sha256"] != first.SHA256 || metadata["tag"] != "sample-kit-v0.1.0" {
		t.Fatalf("release metadata lost source provenance: %+v", metadata)
	}

	assertInventory(t, encoded)
	if _, err := Pack(t.Context(), root, "sample-kit", filepath.Join(root, "first"), commit); err != nil {
		t.Fatalf("identical artifact retry failed: %v", err)
	}

	writeFixture(t, first.Files.Checksum, []byte("corrupt"))
	if _, err := Pack(t.Context(), root, "sample-kit", filepath.Join(root, "first"), commit); err == nil {
		t.Fatal("replaced a different existing release artifact")
	}
}

func assertInventory(t *testing.T, encoded []byte) {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(encoded), int64(len(encoded)))
	if err != nil {
		t.Fatal(err)
	}

	files, names := map[string][]byte{}, []string{}
	for _, entry := range reader.File {
		if entry.Mode().Perm() != 0644 || entry.Method != zip.Deflate || entry.Modified.Year() != 1980 {
			t.Fatal("ZIP framing differs from the format 1 contract")
		}

		stream, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}

		data, err := io.ReadAll(stream)
		closeErr := stream.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("ZIP payload failed CRC/read checks: %v %v", err, closeErr)
		}

		names = append(names, entry.Name)
		files[entry.Name] = data
	}

	if !slices.IsSorted(names) || len(files) != 5 || files["dist/ignored"] != nil || files[".git/ignored"] != nil {
		t.Fatalf("ZIP order, count or exclusions differ: %v", names)
	}

	var index inventory
	if err := json.Unmarshal(files[InventoryName], &index); err != nil {
		t.Fatal(err)
	}

	if index.FormatVersion != 1 || len(index.Files) != len(files)-1 {
		t.Fatal("inventory format/count differs")
	}

	for _, record := range index.Files {
		if record.Path == InventoryName || record.SHA256 != Digest(files[record.Path]) || record.Size != int64(len(files[record.Path])) {
			t.Fatalf("inventory digest or size mismatch: %+v", record)
		}
	}
}

func TestPackageSpecificIdentityAndSelectors(t *testing.T) {
	t.Parallel()
	root, _ := fixture(t)
	matrix, err := Discover(t.Context(), root)
	if err != nil || len(matrix.Include) != 1 || matrix.Include[0].Tag != "sample-kit-v0.1.0" {
		t.Fatalf("matrix discovery failed: %+v %v", matrix, err)
	}

	for _, selector := range []string{"sample-kit", "sample-kit-v0.1.0"} {
		if _, err := Resolve(t.Context(), root, selector); err != nil {
			t.Fatal(err)
		}
	}

	for _, selector := range []string{"sample-kit", "sample-kit-v0.2.0", "v0.1.0", "../sample-kit", "sample-kit-vlatest"} {
		if _, err := VerifyTag(t.Context(), root, selector); err == nil {
			t.Fatalf("invalid release tag accepted: %q", selector)
		}
	}

	if _, err := Pack(t.Context(), root, "sample-kit", filepath.Join(root, "out"), "invalid"); err == nil {
		t.Fatal("invalid source commit accepted")
	}
}
