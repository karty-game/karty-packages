// Package controllercheck validates the released controller without engine source.
package controllercheck

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pelletier/go-toml/v2"
)

//go:embed testdata/main.go.txt testdata/cases.json
var fixtures embed.FS

// Options supplies immutable input artifacts and the selected SDK's compilers.
type Options struct {
	SDKArchive, SDKSHA256, PackageArchive, PackageSHA256 string
	Go, TinyGo, Output                                   string
}

type toolPins struct {
	Go     string `toml:"go"`
	TinyGo string `toml:"tinygo"`
}
type manifest struct {
	Version string   `toml:"version"`
	Tools   toolPins `toml:"tools"`
}
type report struct {
	SDKVersion    string       `json:"sdkVersion"`
	SDKSHA256     string       `json:"sdkSHA256"`
	PackageSHA256 string       `json:"packageSHA256"`
	Go            string       `json:"go"`
	TinyGo        string       `json:"tinygo"`
	Cases         []caseResult `json:"cases"`
}

// Run compiles verified archive sources and executes all controller allocation cases.
// A failed run removes any previous success report at the requested output path.
func Run(ctx context.Context, options Options) error {
	if options.Output == "" {
		return fmt.Errorf("output is required")
	}
	if err := os.Remove(options.Output); err != nil && !os.IsNotExist(err) {
		return err
	}
	sdk, err := readArchive(options.SDKArchive, options.SDKSHA256)
	if err != nil {
		return fmt.Errorf("SDK archive: %w", err)
	}
	controller, err := readArchive(options.PackageArchive, options.PackageSHA256)
	if err != nil {
		return fmt.Errorf("controller archive: %w", err)
	}
	var selected manifest
	if err := toml.Unmarshal(sdk["manifest.toml"], &selected); err != nil {
		return err
	}
	if err := checkCompilers(ctx, options, selected.Tools); err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "karty-controller-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	if err := stage(directory, sdk, controller); err != nil {
		return err
	}
	wasmPath := filepath.Join(directory, "controller.wasm")
	command := exec.CommandContext(ctx, options.TinyGo, "build", "-target=wasi", "-buildmode=c-shared", "-scheduler=none", "-o", wasmPath, "./app")
	command.Dir = directory
	command.Env = compilerEnv(options.Go)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("compile TinyGo controller: %w\n%s", err, output)
	}
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		return err
	}
	results, err := execute(ctx, wasm)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(report{selected.Version, strings.ToLower(options.SDKSHA256), strings.ToLower(options.PackageSHA256), selected.Tools.Go, selected.Tools.TinyGo, results}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(options.Output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(options.Output, append(data, '\n'), 0o644)
}

func readArchive(filename, expected string) (map[string][]byte, error) {
	sum, err := hex.DecodeString(expected)
	if err != nil || len(sum) != sha256.Size {
		return nil, fmt.Errorf("exact SHA-256 is required")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (32<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 32<<20 {
		return nil, fmt.Errorf("archive exceeds 32 MiB")
	}
	actual := sha256.Sum256(data)
	if !bytes.Equal(actual[:], sum) {
		return nil, fmt.Errorf("SHA-256 mismatch: got %x", actual)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	if len(reader.File) > 4096 {
		return nil, fmt.Errorf("too many archive entries")
	}
	result := make(map[string][]byte)
	seen := make(map[string]bool)
	var total uint64
	for _, entry := range reader.File {
		name := entry.Name
		if name == "" || strings.ContainsAny(name, "\\\x00") || path.IsAbs(name) || path.Clean(name) != strings.TrimSuffix(name, "/") || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, ":") {
			return nil, fmt.Errorf("unsafe archive path %q", name)
		}
		key := strings.ToLower(strings.TrimSuffix(name, "/"))
		if seen[key] {
			return nil, fmt.Errorf("duplicate archive path %q", name)
		}
		seen[key] = true
		if entry.FileInfo().IsDir() {
			continue
		}
		if !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("non-regular archive entry %q", name)
		}
		total += entry.UncompressedSize64
		if entry.UncompressedSize64 > 16<<20 || total > 64<<20 {
			return nil, fmt.Errorf("expanded archive exceeds bounds")
		}
		source, err := entry.Open()
		if err != nil {
			return nil, err
		}
		contents, readErr := io.ReadAll(io.LimitReader(source, (16<<20)+1))
		closeErr := source.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(contents) > 16<<20 {
			return nil, fmt.Errorf("entry exceeds bounds")
		}
		result[name] = contents
	}
	return result, nil
}

func stage(directory string, sdk, controller map[string][]byte) error {
	for _, name := range []string{"game.go", "components.go", "protocol.go", "renderers.go"} {
		data, ok := sdk["bindings/tinygo/engine/"+name]
		if !ok {
			return fmt.Errorf("SDK missing %s", name)
		}
		if err := write(directory, ".karty/engine/"+name, data); err != nil {
			return err
		}
	}
	for _, name := range []string{"controller", "motion", "touch", "validation"} {
		data, ok := controller["bridge/templates/"+name+".go.tmpl"]
		if !ok {
			return fmt.Errorf("controller archive missing %s template", name)
		}
		parsed, err := template.New(name).Option("missingkey=error").Parse(string(data))
		if err != nil {
			return err
		}
		var buffer bytes.Buffer
		if err := parsed.Execute(&buffer, struct{ Module string }{"example.com/controllercheck"}); err != nil {
			return err
		}
		formatted, err := format.Source(buffer.Bytes())
		if err != nil {
			return err
		}
		if err := write(directory, ".karty/processed/code/controllers-3d.sdk/controller/"+name+".go", formatted); err != nil {
			return err
		}
	}
	main, err := fixtures.ReadFile("testdata/main.go.txt")
	if err != nil {
		return err
	}
	if err := write(directory, "app/main.go", main); err != nil {
		return err
	}
	return write(directory, "go.mod", []byte("module example.com/controllercheck\n\ngo 1.27.0\n"))
}

func write(directory, name string, data []byte) error {
	target := filepath.Join(directory, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func checkCompilers(ctx context.Context, options Options, pins toolPins) error {
	valid := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	if !valid.MatchString(pins.Go) || !valid.MatchString(pins.TinyGo) {
		return fmt.Errorf("SDK lacks exact Go/TinyGo pins")
	}
	for _, compiler := range []struct{ path, prefix, version string }{{options.Go, "go version go", pins.Go}, {options.TinyGo, "tinygo version ", pins.TinyGo}} {
		command := exec.CommandContext(ctx, compiler.path, "version")
		command.Env = compilerEnv(options.Go)
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("compiler version: %w: %s", err, output)
		}
		if !strings.HasPrefix(string(output), compiler.prefix+compiler.version+" ") {
			return fmt.Errorf("compiler mismatch: expected %s%s, got %s", compiler.prefix, compiler.version, output)
		}
	}
	return nil
}

func compilerEnv(goExecutable string) []string {
	// TinyGo invokes Go from PATH. Select the same Go whose version was checked.
	environment := make([]string, 0, len(os.Environ())+2)
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "GOWORK=") || strings.HasPrefix(value, "PATH=") || strings.HasPrefix(value, "GOROOT=") || strings.HasPrefix(value, "GOFLAGS=") {
			continue
		}
		environment = append(environment, value)
	}
	executable, err := exec.LookPath(goExecutable)
	if err != nil {
		executable = goExecutable
	}
	return append(environment, "GOWORK=off", "GOFLAGS="+strings.TrimSpace(os.Getenv("GOFLAGS")+" -buildvcs=false"), "PATH="+filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
}
