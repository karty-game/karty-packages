package kpkg

import (
	"bytes"
	"fmt"
	"image/png"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

var namePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,47}$`)

type manifest struct {
	FormatVersion int         `toml:"format_version"`
	Name          string      `toml:"name"`
	Version       string      `toml:"version"`
	SDK           string      `toml:"sdk,omitempty"`
	Prefabs       string      `toml:"prefabs,omitempty"`
	Textures      []texture   `toml:"textures,omitempty"`
	UI            uiExports   `toml:"ui,omitempty"`
	Processors    []processor `toml:"processors,omitempty"`
}

type texture struct {
	Name        string       `toml:"name"`
	Source      string       `toml:"source"`
	Profile     string       `toml:"profile"`
	Materialize *materialize `toml:"materialize,omitempty"`
}

type materialize struct {
	Preset         string   `toml:"preset"`
	NormalStrength *float64 `toml:"normal_strength"`
	HeightContrast *float64 `toml:"height_contrast"`
	MetallicScale  *float64 `toml:"metallic_scale"`
	AODepthScale   *float64 `toml:"ao_depth_scale"`
}

type uiExports struct {
	Components []uiComponent `toml:"components,omitempty"`
	Layouts    []string      `toml:"layouts,omitempty"`
	Images     []texture     `toml:"images,omitempty"`
	Fonts      []uiFont      `toml:"fonts,omitempty"`
	Theme      string        `toml:"theme,omitempty"`
}

type uiComponent struct {
	Name   string `toml:"name"`
	Source string `toml:"source"`
}

type uiFont struct {
	Role   string `toml:"role"`
	Source string `toml:"source"`
}

type processor struct {
	Name    string   `toml:"name"`
	Command []string `toml:"command"`
}

func parseManifest(files map[string][]byte) (manifest, error) {
	data := files[ManifestName]
	if len(data) == 0 || len(data) > MaxManifestBytes {
		return manifest{}, fmt.Errorf("%w: missing or oversized manifest", ErrPackage)
	}

	var definition manifest
	if err := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(&definition); err != nil {
		return definition, err
	}

	if definition.FormatVersion != 1 || !namePattern.MatchString(definition.Name) || !validPath(definition.Name) ||
		!validVersion(definition.Version) || (definition.SDK != "" && !validVersion(definition.SDK)) {
		return definition, fmt.Errorf("%w: unsupported format, identity or version", ErrPackage)
	}

	if definition.Prefabs != "" {
		if err := requireSource(files, definition.Prefabs); err != nil {
			return definition, err
		}
	}

	if err := validateTextures(files, definition.Textures); err != nil {
		return definition, err
	}

	if err := validateUI(files, definition.UI); err != nil {
		return definition, err
	}

	if err := validateProcessors(definition.Processors); err != nil {
		return definition, err
	}

	if data, exists := files["go.mod"]; exists {
		mod, err := modfile.Parse("go.mod", data, nil)
		if err != nil {
			return definition, err
		}

		if mod.Module == nil || len(mod.Replace) != 0 {
			return definition, fmt.Errorf("%w: go.mod requires a module and no replacements", ErrPackage)
		}

		if err := module.Check(mod.Module.Mod.Path, "v"+definition.Version); err != nil {
			return definition, err
		}
	} else if definition.Prefabs == "" && len(definition.Textures) == 0 && len(definition.Processors) == 0 &&
		len(definition.UI.Components)+len(definition.UI.Layouts)+len(definition.UI.Images)+len(definition.UI.Fonts) == 0 &&
		definition.UI.Theme == "" {
		return definition, fmt.Errorf("%w: no code or resource exports", ErrPackage)
	}

	return definition, nil
}

func validVersion(value string) bool {
	return len(value) <= 64 && semver.Canonical("v"+value) == "v"+value
}

func requireSource(files map[string][]byte, name string) error {
	if !validPath(name) || len(files[name]) == 0 {
		return fmt.Errorf("%w: missing or invalid declared source %q", ErrPackage, name)
	}

	return nil
}

func validateTextures(files map[string][]byte, textures []texture) error {
	if len(textures) > 256 {
		return ErrPackage
	}

	seen := map[string]bool{}
	for _, item := range textures {
		if !namePattern.MatchString(item.Name) || seen[item.Name] || !namePattern.MatchString(item.Profile) ||
			!strings.HasSuffix(strings.ToLower(item.Source), ".png") {
			return fmt.Errorf("%w: invalid or duplicate texture %q", ErrPackage, item.Name)
		}

		seen[item.Name] = true
		if err := requireSource(files, item.Source); err != nil {
			return err
		}

		if err := validatePNG(files[item.Source]); err != nil {
			return fmt.Errorf("texture %q: %w", item.Name, err)
		}

		if item.Materialize != nil {
			if item.Profile != "environment" {
				return fmt.Errorf("%w: materialize requires environment profile", ErrPackage)
			}

			if err := validateMaterialize(*item.Materialize); err != nil {
				return err
			}
		}
	}

	return nil
}

func validatePNG(data []byte) error {
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 16384 || config.Height > 16384 {
		return fmt.Errorf("%w: invalid or oversized PNG", ErrPackage)
	}

	pixelBytes := uint64(4)
	if data[24] > 8 {
		pixelBytes = 8
	}

	if uint64(config.Width)*uint64(config.Height)*pixelBytes > 64<<20 {
		return fmt.Errorf("%w: PNG exceeds decoded bounds", ErrPackage)
	}

	_, err = png.Decode(bytes.NewReader(data))

	return err
}

func validateMaterialize(options materialize) error {
	if !slices.Contains([]string{"", "default", "auto", "skin", "floor", "metal", "fabric", "wood", "stone",
		"concrete", "leather", "marble", "sand", "foliage", "plaster", "asphalt", "brick", "ice", "snow", "lava", "water"}, options.Preset) {
		return fmt.Errorf("%w: unknown Materialize preset %q", ErrPackage, options.Preset)
	}

	for _, value := range []*float64{options.NormalStrength, options.HeightContrast, options.MetallicScale, options.AODepthScale} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 || *value > 16) {
			return fmt.Errorf("%w: materialize override must be finite within [0,16]", ErrPackage)
		}
	}

	return nil
}

func validateUI(files map[string][]byte, exports uiExports) error {
	if len(exports.Components) > 128 || len(exports.Layouts) > 128 || len(exports.Fonts) > 128 {
		return ErrPackage
	}

	seen := map[string]bool{}
	for _, component := range exports.Components {
		if !namePattern.MatchString(component.Name) || seen[component.Name] || !strings.HasSuffix(component.Source, ".kui") {
			return fmt.Errorf("%w: invalid UI component %q", ErrPackage, component.Name)
		}

		seen[component.Name] = true
		if err := requireSource(files, component.Source); err != nil {
			return err
		}
	}

	sources := slices.Clone(exports.Layouts)
	for _, font := range exports.Fonts {
		if font.Role == "" || len(font.Role) > 64 {
			return ErrPackage
		}

		sources = append(sources, font.Source)
	}

	if exports.Theme != "" {
		sources = append(sources, exports.Theme)
	}

	for _, name := range sources {
		if err := requireSource(files, name); err != nil {
			return err
		}
	}

	return validateTextures(files, exports.Images)
}

func validateProcessors(processors []processor) error {
	if len(processors) > 16 {
		return ErrPackage
	}

	seen := map[string]bool{}
	for _, item := range processors {
		if !namePattern.MatchString(item.Name) || seen[item.Name] || len(item.Command) == 0 || len(item.Command) > 32 {
			return fmt.Errorf("%w: invalid processor %q", ErrPackage, item.Name)
		}

		seen[item.Name] = true
		for _, argument := range item.Command {
			if argument == "" || len(argument) > 4096 || strings.ContainsRune(argument, 0) {
				return ErrPackage
			}
		}
	}

	return nil
}
