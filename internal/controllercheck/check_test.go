package controllercheck

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func archiveFixture(t *testing.T, entries ...string) (string, string) {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for _, name := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("source")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(t.TempDir(), "input.zip")
	if err := os.WriteFile(filename, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return filename, fmt.Sprintf("%x", sha256.Sum256(data.Bytes()))
}

func TestReadArchiveAuthenticatesAndConfinesPaths(t *testing.T) {
	filename, sum := archiveFixture(t, "bridge/templates/controller.go.tmpl")
	if _, err := readArchive(filename, sum); err != nil {
		t.Fatal(err)
	}
	for _, badSum := range []string{"", strings.Repeat("0", 64)} {
		if _, err := readArchive(filename, badSum); err == nil {
			t.Fatal("accepted missing or incorrect SHA")
		}
	}
	for _, name := range []string{"../outside", "/absolute", "a/../outside", `a\outside`, "C:/outside"} {
		t.Run(name, func(t *testing.T) {
			filename, sum := archiveFixture(t, name)
			if _, err := readArchive(filename, sum); err == nil {
				t.Fatal("accepted unsafe entry")
			}
		})
	}
	filename, sum = archiveFixture(t, "file.go", "FILE.go")
	if _, err := readArchive(filename, sum); err == nil {
		t.Fatal("accepted case collision")
	}
}

func TestCommandDecoderRejectsPartialAndInvalidWalking(t *testing.T) {
	g := &guest{config: configuration{Commands: map[string]byte{"velocity": 45}}, commands: make(map[string]uint64)}
	data := make([]byte, 47)
	copy(data, "KTYB")
	data[4], data[6] = 16, 2
	binary.LittleEndian.PutUint32(data[16:], 1)
	binary.LittleEndian.PutUint32(data[20:], 23)
	data[24] = 45
	binary.LittleEndian.PutUint16(data[25:], 20)
	binary.LittleEndian.PutUint32(data[27:], 1)
	binary.LittleEndian.PutUint32(data[35:], math.Float32bits(3.3))
	binary.LittleEndian.PutUint32(data[43:], 1)
	if err := g.submit(data); err != nil {
		t.Fatal(err)
	}
	if g.commands["velocity"] != 1 || g.velocity[1] != 3.3 {
		t.Fatal("walking command was not observed")
	}
	for index := range data {
		if err := g.submit(data[:index]); err == nil {
			t.Fatalf("accepted prefix %d", index)
		}
	}
	for _, index := range []int{27, 43} {
		invalid := bytes.Clone(data)
		invalid[index] = 0
		if err := g.submit(invalid); err == nil {
			t.Fatal("accepted wrong owner or vertical momentum flag")
		}
	}
	invalid := bytes.Clone(data)
	binary.LittleEndian.PutUint32(invalid[31:], math.Float32bits(float32(math.NaN())))
	if err := g.submit(invalid); err == nil {
		t.Fatal("accepted nonfinite motion")
	}
}

func TestStageRequiresBothArchivedImplementations(t *testing.T) {
	sdk := make(map[string][]byte)
	controller := make(map[string][]byte)
	for _, name := range []string{"game.go", "components.go", "protocol.go", "renderers.go"} {
		sdk["bindings/tinygo/engine/"+name] = []byte("package engine\n")
	}
	for _, name := range []string{"controller", "motion", "touch", "validation"} {
		controller["bridge/templates/"+name+".go.tmpl"] = []byte("package controller\n// {{.Module}}\n")
	}
	directory := t.TempDir()
	if err := stage(directory, sdk, controller); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(filepath.Join(directory, ".karty/processed/code/controllers-3d.sdk/controller/controller.go"))
	if err != nil || !bytes.Contains(output, []byte("example.com/controllercheck")) {
		t.Fatal("archived bridge template was not rendered")
	}
	delete(controller, "bridge/templates/motion.go.tmpl")
	if err := stage(t.TempDir(), sdk, controller); err == nil {
		t.Fatal("accepted incomplete package")
	}
	delete(sdk, "bindings/tinygo/engine/game.go")
	if err := stage(t.TempDir(), sdk, controller); err == nil {
		t.Fatal("accepted incomplete SDK")
	}
}

func TestFailedRunRemovesStaleSuccessReport(t *testing.T) {
	output := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(output, []byte(`{"cases":"stale success"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Run(t.Context(), Options{Output: output}); err == nil {
		t.Fatal("accepted absent verified archives")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("failed run retained a stale success report")
	}
}
