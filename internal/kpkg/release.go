package kpkg

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Identity struct {
	Package string `json:"package"`
	Version string `json:"version"`
	Tag     string `json:"tag"`
	Archive string `json:"archive"`
}

type Matrix struct {
	Include []Identity `json:"include"`
}

type ArtifactPaths struct {
	Archive  string `json:"archive"`
	Checksum string `json:"checksum"`
	Metadata string `json:"metadata"`
}

type Release struct {
	Identity
	SHA256       string        `json:"sha256"`
	Size         int64         `json:"size"`
	SourceCommit string        `json:"source_commit,omitempty"`
	Files        ArtifactPaths `json:"files"`
}

func identity(definition manifest) Identity {
	return Identity{Package: definition.Name, Version: definition.Version,
		Tag: definition.Name + "-v" + definition.Version, Archive: definition.Name + "-" + definition.Version + ".kpkg"}
}

func inspect(ctx context.Context, directory string) (Identity, map[string][]byte, error) {
	info, err := os.Lstat(directory)
	if err != nil {
		return Identity{}, nil, err
	}

	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Identity{}, nil, fmt.Errorf("%w: package directory must be a real directory", ErrPackage)
	}

	files, err := collect(ctx, directory)
	if err != nil {
		return Identity{}, nil, err
	}

	definition, err := parseManifest(files)
	if err != nil {
		return Identity{}, nil, err
	}

	if filepath.Base(directory) != definition.Name {
		return Identity{}, nil, fmt.Errorf("%w: package directory must match manifest name %q", ErrPackage, definition.Name)
	}

	return identity(definition), files, nil
}

func Discover(ctx context.Context, repository string) (Matrix, error) {
	directory, err := packageNamespace(repository)
	if err != nil {
		return Matrix{}, err
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return Matrix{}, err
	}

	result := Matrix{Include: []Identity{}}
	seen := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}

		item, _, err := inspect(ctx, filepath.Join(directory, entry.Name()))
		if err != nil {
			return Matrix{}, err
		}

		folded := strings.ToLower(item.Package)
		if seen[folded] {
			return Matrix{}, fmt.Errorf("%w: case-colliding package identity", ErrPackage)
		}

		seen[folded] = true
		result.Include = append(result.Include, item)
	}

	if len(result.Include) == 0 {
		return result, fmt.Errorf("%w: no packages discovered", ErrPackage)
	}

	slices.SortFunc(result.Include, func(a, b Identity) int { return strings.Compare(a.Package, b.Package) })

	return result, nil
}

func resolve(ctx context.Context, repository, selector string) (Identity, map[string][]byte, error) {
	directory, err := packageNamespace(repository)
	if err != nil {
		return Identity{}, nil, err
	}

	name := selector
	if index := strings.LastIndex(selector, "-v"); index >= 0 && validVersion(selector[index+2:]) {
		name = selector[:index]
	}

	if !namePattern.MatchString(name) || !validPath(name) {
		return Identity{}, nil, fmt.Errorf("%w: invalid package selector %q", ErrPackage, selector)
	}

	item, files, err := inspect(ctx, filepath.Join(directory, name))
	if err != nil {
		return item, nil, err
	}

	if selector != item.Package && selector != item.Tag {
		return item, nil, fmt.Errorf("%w: tag %q does not match declared version %s", ErrPackage, selector, item.Version)
	}

	return item, files, nil
}

func packageNamespace(repository string) (string, error) {
	directory := filepath.Join(repository, "packages")
	info, err := os.Lstat(directory)
	if err != nil {
		return "", err
	}

	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%w: packages must be a real directory", ErrPackage)
	}

	return directory, nil
}

func Resolve(ctx context.Context, repository, selector string) (Identity, error) {
	item, _, err := resolve(ctx, repository, selector)

	return item, err
}

func VerifyTag(ctx context.Context, repository, tag string) (Identity, error) {
	item, err := Resolve(ctx, repository, tag)
	if err == nil && item.Tag != tag {
		return item, fmt.Errorf("%w: expected a package-specific release tag", ErrPackage)
	}

	return item, err
}

func Pack(ctx context.Context, repository, selector, output, commit string) (Release, error) {
	item, files, err := resolve(ctx, repository, selector)
	if err != nil {
		return Release{}, err
	}

	if commit != "" {
		decoded, err := hex.DecodeString(commit)
		if err != nil || len(decoded) != 20 || strings.ToLower(commit) != commit {
			return Release{}, fmt.Errorf("%w: source commit must be a lowercase 40-digit Git SHA", ErrPackage)
		}
	}

	output, err = filepath.Abs(output)
	if err != nil {
		return Release{}, err
	}

	packageRoot, err := filepath.Abs(filepath.Join(repository, "packages", item.Package))
	if err != nil {
		return Release{}, err
	}

	relative, err := filepath.Rel(packageRoot, output)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return Release{}, fmt.Errorf("%w: release output must be outside the package source", ErrPackage)
	}

	encoded, err := encodeArchive(ctx, files)
	if err != nil {
		return Release{}, err
	}

	result := Release{Identity: item, SHA256: Digest(encoded), Size: int64(len(encoded)), SourceCommit: commit,
		Files: ArtifactPaths{Archive: filepath.ToSlash(filepath.Join(output, item.Archive)),
			Checksum: filepath.ToSlash(filepath.Join(output, item.Archive+".sha256")),
			Metadata: filepath.ToSlash(filepath.Join(output, item.Package+"-"+item.Version+".json"))}}
	metadata, err := json.MarshalIndent(struct {
		Identity
		SHA256       string `json:"sha256"`
		Size         int64  `json:"size"`
		SourceCommit string `json:"source_commit,omitempty"`
	}{item, result.SHA256, result.Size, commit}, "", "  ")
	if err != nil {
		return Release{}, err
	}

	artifacts := map[string][]byte{item.Archive: encoded,
		item.Archive + ".sha256":                    []byte(result.SHA256 + "  " + item.Archive + "\n"),
		item.Package + "-" + item.Version + ".json": append(metadata, '\n')}
	if err := publishArtifacts(ctx, output, artifacts); err != nil {
		return Release{}, err
	}

	return result, nil
}

func publishArtifacts(ctx context.Context, directory string, artifacts map[string][]byte) error {
	if err := os.MkdirAll(directory, 0750); err != nil {
		return err
	}

	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()

	names := make([]string, 0, len(artifacts))
	for name, data := range artifacts {
		names = append(names, name)
		if _, err := root.Lstat(name); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}

		before, err := read(ctx, root, name, MaxArchiveBytes)
		if err != nil || !bytes.Equal(before, data) {
			return fmt.Errorf("%w: refusing to replace artifact %s", ErrPackage, name)
		}
	}

	slices.Sort(names)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}

		file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if os.IsExist(err) {
			before, err := read(ctx, root, name, MaxArchiveBytes)
			if err != nil || !bytes.Equal(before, artifacts[name]) {
				return fmt.Errorf("%w: concurrent artifact changed %s", ErrPackage, name)
			}

			continue
		}

		if err != nil {
			return err
		}

		_, writeErr := file.Write(artifacts[name])
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			_ = root.Remove(name)

			return fmt.Errorf("write artifact %s: %w", name, errors.Join(writeErr, closeErr))
		}
	}

	return nil
}
