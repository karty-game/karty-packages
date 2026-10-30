package character

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"math"
	"os"
	"strings"
	"testing"
)

type assetManifest struct {
	FormatVersion int    `json:"format_version"`
	SourceSHA256  string `json:"source_sha256"`
	AnimationBank string `json:"animation_bank"`
	Models        []struct {
		Role, File        string
		Triangles, Joints int
	}
	Clips []struct {
		ID, Clip string
		Seconds  float64
		Loop     bool
	}
	Files []struct {
		Path, SHA256 string
		Bytes        int
	}
}

func readJSON(t *testing.T, name string, target any) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func TestPackagedAssetInventory(t *testing.T) {
	var manifest assetManifest
	readJSON(t, "assets.json", &manifest)
	if manifest.FormatVersion != 1 || manifest.AnimationBank != AnimationBank ||
		manifest.SourceSHA256 != "30b7d48fdee566fadff47a1cc4fdea1bc35eac30a1a05f784661fa28738f66e5" {
		t.Fatal("unexpected canonical asset input")
	}
	seen := map[string]bool{}
	for _, file := range manifest.Files {
		if !fs.ValidPath(file.Path) || seen[file.Path] {
			t.Fatal("invalid inventory path", file.Path)
		}
		seen[file.Path] = true
		data, err := os.ReadFile(file.Path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if len(data) != file.Bytes || hex.EncodeToString(sum[:]) != file.SHA256 {
			t.Fatal("asset identity changed without updating inventory", file.Path)
		}
	}
	for _, root := range []string{"models", "animations", "source"} {
		err := filepathWalk(root, func(path string, entry fs.DirEntry) error {
			if entry.Type()&os.ModeSymlink != 0 {
				t.Fatal("asset symlink", path)
			}
			if !entry.IsDir() && !seen[path] {
				t.Fatal("untracked asset", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(manifest.Models) != len(models) || len(manifest.Clips) != len(clips) {
		t.Fatal("incomplete cast or clip table")
	}
	for i, model := range manifest.Models {
		if model.Role != models[i].Role || model.File != models[i].File ||
			model.Triangles != models[i].Triangles || model.Joints != models[i].Joints || !seen[model.File] {
			t.Fatal("model catalog differs from assets", model.Role)
		}
	}
	for i, clip := range manifest.Clips {
		if clip.ID != clips[i].Alias || clip.Clip != clips[i].Source ||
			clip.Seconds != clips[i].Seconds || clip.Loop != clips[i].Loop {
			t.Fatal("controller profile differs from asset manifest", clip.ID)
		}
	}
	// Retain the latest surface correction.
	corrected, err := os.ReadFile("models/scientist.glb")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(corrected)
	if hex.EncodeToString(sum[:]) != "2d0310db9d1f032778c595e34d734b7f0861448ae1a80107e7c1c27364be35f1" {
		t.Fatal("scientist surface-correction baseline changed; review before replacing")
	}
}

func filepathWalk(root string, visit func(string, fs.DirEntry) error) error {
	return fs.WalkDir(os.DirFS("."), root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return visit(path, entry)
	})
}

type glbDocument struct {
	Nodes []struct {
		Name     string
		Children []int
	}
	Skins     []struct{ Joints []int }
	Buffers   []struct{ URI string }
	Images    []struct{ URI string }
	Meshes    []struct{ Primitives []struct{ Indices int } }
	Accessors []struct {
		Count int
		Max   []float64
	}
	Animations []struct {
		Name     string
		Channels []struct{ Target struct{ Node int } }
		Samplers []struct{ Input int }
	}
}

// This inspects embedded GLB structure only. Meshopt decoding and GPU skinning
// remain renderer validation; hashes above preserve their previously checked bytes.
func readGLB(t *testing.T, name string) glbDocument {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 28 || string(data[:4]) != "glTF" || binary.LittleEndian.Uint32(data[4:]) != 2 ||
		int(binary.LittleEndian.Uint32(data[8:])) != len(data) || string(data[16:20]) != "JSON" {
		t.Fatal("invalid GLB header", name)
	}
	end := 20 + int(binary.LittleEndian.Uint32(data[12:]))
	if end > len(data)-8 {
		t.Fatal("truncated GLB", name)
	}
	var doc glbDocument
	if err := json.Unmarshal(data[20:end], &doc); err != nil {
		t.Fatal(err)
	}
	for _, b := range doc.Buffers {
		if b.URI != "" {
			t.Fatal("external GLB buffer", name, b.URI)
		}
	}
	for _, im := range doc.Images {
		if im.URI != "" {
			t.Fatal("external GLB texture", name, im.URI)
		}
	}
	return doc
}

func TestModelsBindToTheSharedAnimationBank(t *testing.T) {
	bank := readGLB(t, AnimationBank)
	if len(bank.Animations) != len(clips) || len(bank.Meshes) != 0 {
		t.Fatal("bank must contain all clips without model geometry")
	}
	for _, model := range Models() {
		doc := readGLB(t, model.File)
		if len(doc.Animations) != 0 || len(doc.Skins) != 1 || len(doc.Skins[0].Joints) != model.Joints {
			t.Fatal("unexpected model rig/duplicated animation", model.Role)
		}
		names := map[string]bool{}
		for _, node := range doc.Nodes {
			if node.Name != "" {
				names[node.Name] = true
			}
		}
		count := 0
		for _, mesh := range doc.Meshes {
			for _, primitive := range mesh.Primitives {
				count += doc.Accessors[primitive.Indices].Count
			}
		}
		if count != model.Triangles*3 {
			t.Fatal("triangle count differs", model.Role, count)
		}
		for _, clip := range bank.Animations {
			id := -1
			for i, item := range clips {
				if item.Source == clip.Name {
					id = i
				}
			}
			if id < 0 {
				t.Fatal("unexpected source clip", clip.Name)
			}
			last := 0.0
			for _, sampler := range clip.Samplers {
				a := bank.Accessors[sampler.Input]
				if len(a.Max) != 1 {
					t.Fatal("missing clip time bounds", clip.Name)
				}
				last = max(last, a.Max[0])
			}
			if math.Abs(last-clips[id].Seconds) > 1e-5 {
				t.Fatal("clip duration drift", clip.Name, last)
			}
			for _, channel := range clip.Channels {
				name := bank.Nodes[channel.Target.Node].Name
				if name == "" || !names[name] {
					t.Fatal("clip target missing from model", model.Role, name)
				}
			}
		}
	}
}

func TestIndependentControllersAndInvalidRequests(t *testing.T) {
	var a, b Controller
	step(&a, Input{Grounded: true, Death: 1})
	if out := step(&b, Input{Grounded: true, Forward: 1.4}); out.Dead || out.Base != "walk" {
		t.Fatal(out)
	}
	before := a
	for _, in := range []Input{{Request: Use + 1}, {Death: 4}, {Stance: Prone + 1}, {Equipment: Pistol + 1}} {
		step(&a, in)
		if a != before {
			t.Fatal("invalid input changed private state")
		}
	}
	metadata := Clips()
	metadata[0].Alias = "bad"
	if ClipID("idle") != 0 || Duration("unknown") != 0 || ClipID("unknown") != -1 {
		t.Fatal("metadata mutation or unknown lookup")
	}
	for _, clip := range Clips() {
		if strings.TrimSpace(clip.Alias) == "" || clip.Seconds <= 0 {
			t.Fatal(clip)
		}
	}
}
