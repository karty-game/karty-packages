// Package bridge generates the package-owned first-person SDK adapter.
package bridge

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"strings"
	"text/template"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

//go:embed templates/*.go.tmpl
var templates embed.FS

// Generate renders ordinary Go source against the consuming project's SDK.
// The module input is authored source; generated files never import engine internals.
func Generate(goMod []byte) (map[string][]byte, error) {
	definition, err := modfile.Parse("go.mod", goMod, nil)
	if err != nil {
		return nil, err
	}
	if definition.Module == nil {
		return nil, fmt.Errorf("controllers-3d.sdk requires a project module directive")
	}
	return Render(definition.Module.Mod.Path)
}

// Render adapts the package-owned implementation to a project's generated SDK.
func Render(name string) (map[string][]byte, error) {
	if err := module.CheckImportPath(name); err != nil {
		return nil, fmt.Errorf("invalid project module: %w", err)
	}
	entries, err := fs.ReadDir(templates, "templates")
	if err != nil {
		return nil, err
	}
	output := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		path := "templates/" + entry.Name()
		data, err := templates.ReadFile(path)
		if err != nil {
			return nil, err
		}
		parsed, err := template.New(entry.Name()).Option("missingkey=error").Parse(string(data))
		if err != nil {
			return nil, err
		}
		var rendered bytes.Buffer
		if err := parsed.Execute(&rendered, struct{ Module string }{name}); err != nil {
			return nil, err
		}
		source, err := format.Source(rendered.Bytes())
		if err != nil {
			return nil, fmt.Errorf("format %s: %w", entry.Name(), err)
		}
		output["go/controller/"+strings.TrimSuffix(entry.Name(), ".tmpl")] = source
	}
	return output, nil
}
