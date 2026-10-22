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
	releasesPath := flag.String("releases", "", "Offline releases JSON: repository map, or array for the default repository")
	output := flag.String("output", "", "Output directory")
	catalogPath := flag.String("catalog", "catalog.json", "Curated catalog metadata")
	flag.Parse()
	if *repository == "" || *output == "" || flag.NArg() != 0 {
		return fmt.Errorf("usage: karty-catalog --repository owner/repo --output directory [--catalog catalog.json] [--releases releases.json]")
	}
	metadata, err := os.ReadFile(*catalogPath)
	if err != nil {
		return err
	}
	repositories, err := catalog.Repositories(metadata, *repository)
	if err != nil {
		return err
	}
	var releases []byte
	if *releasesPath != "" {
		releases, err = os.ReadFile(*releasesPath)
	} else {
		token := os.Getenv("GH_TOKEN")
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}
		releases, err = catalog.FetchReleases(repositories, token)
	}
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
