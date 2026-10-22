package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/karty-game/karty-packages/internal/catalog"
)

func run() error {
	repository := flag.String("repository", "", "GitHub owner/repository")
	releasesPath := flag.String("releases", "", "GitHub releases JSON array")
	output := flag.String("output", "", "Output directory")
	catalogPath := flag.String("catalog", "catalog.json", "Curated catalog metadata")
	flag.Parse()
	if *repository == "" || *releasesPath == "" || *output == "" || flag.NArg() != 0 {
		return fmt.Errorf("usage: karty-catalog --repository owner/repo --releases releases.json --output directory [--catalog catalog.json]")
	}
	metadata, err := os.ReadFile(*catalogPath)
	if err != nil {
		return err
	}
	releases, err := os.ReadFile(*releasesPath)
	if err != nil {
		return err
	}
	master, err := catalog.Build(metadata, releases, *repository, nil)
	if err != nil {
		return err
	}
	contents, err := json.MarshalIndent(master, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*output, "master.json"), append(contents, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("Generated master.json for %d curated packages\n", len(master.Packages))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
