package bridge

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderUsesOwnedImplementationAndProjectSDK(t *testing.T) {
	t.Parallel()
	first, err := Render("example.com/controller-fixture")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate([]byte("module example.com/controller-fixture\n\ngo 1.27.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 4 || !bytes.Equal(encode(first), encode(second)) {
		t.Fatal("adapter generation is not deterministic")
	}
	var source strings.Builder
	for _, data := range first {
		source.Write(data)
	}
	for _, required := range []string{"func AttachFirstPerson(", "func DefaultFirstPersonOptions(", ".RegisterComponent(", ".CanAttachBody(", ".TeleportPose(", "func (c *FirstPersonController) pointer(", "example.com/controller-fixture/.karty/engine"} {
		if !strings.Contains(source.String(), required) {
			t.Fatalf("adapter lost %s", required)
		}
	}
	for _, forbidden := range []string{"engine.FirstPerson", "simulationBodies", "simulationWorldSettings", "karty-engine", ".camera.settings"} {
		if strings.Contains(source.String(), forbidden) {
			t.Fatalf("adapter depends on SDK policy/private state: %s", forbidden)
		}
	}
	for _, module := range []string{"", "bad module", "example.com/../game", "example.com/\"injection"} {
		if _, err := Render(module); err == nil {
			t.Fatalf("accepted invalid module %q", module)
		}
	}
}

func TestProcessorRejectsInvalidRequestsWithoutOutput(t *testing.T) {
	t.Parallel()
	valid := encode(map[string][]byte{"go.mod": []byte("module example.com/game\n\ngo 1.27.0\n")})
	for name, data := range map[string][]byte{"empty": nil, "truncated": valid[:len(valid)-1], "trailing": append(append([]byte(nil), valid...), 0), "missing module": encode(map[string][]byte{"go.mod": []byte("go 1.27.0\n")}), "extra inputs": encode(map[string][]byte{"go.mod": []byte("module example.com/game\n"), "private.txt": nil}), "path escape": encode(map[string][]byte{"../go.mod": nil})} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			if err := Process(bytes.NewReader(data), &output); err == nil || output.Len() != 0 {
				t.Fatalf("invalid processor input produced output: %v", err)
			}
		})
	}
	var first, second bytes.Buffer
	if err := Process(bytes.NewReader(valid), &first); err != nil {
		t.Fatal(err)
	}
	if err := Process(bytes.NewReader(valid), &second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("processor response is not deterministic")
	}
}

// The child tests retain the controller's native policy assertions against a
// small public SDK API double; real SDK, WASM and browser checks live in consumers.
func TestOwnedControllerBehavior(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	files, err := Render("example.com/controller-fixture")
	if err != nil {
		t.Fatal(err)
	}
	for name, contents := range files {
		writeTestFile(t, directory, strings.TrimPrefix(name, "go/"), contents)
	}
	for _, fixture := range []struct{ source, target string }{{"testdata/engine.go.txt", ".karty/engine/engine.go"}, {"testdata/controller_test.go.txt", "controller/controller_test.go"}} {
		contents, err := os.ReadFile(fixture.source)
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, directory, fixture.target, contents)
	}
	writeTestFile(t, directory, "go.mod", []byte("module example.com/controller-fixture\n\ngo 1.27.0\n"))
	command := exec.CommandContext(t.Context(), "go", "test", "-buildvcs=false", "-count=1", "-timeout=9s", "./controller")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("package-owned controller behavior: %v\n%s", err, output)
	}
}

func writeTestFile(t *testing.T, root, name string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
