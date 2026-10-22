// Package kpkg validates and exports portable Karty package source archives.
package kpkg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"unicode/utf8"
)

const (
	ManifestName     = "karty-package.toml"
	InventoryName    = "karty-package.files.json"
	MaxFiles         = 8192
	MaxFileBytes     = 64 << 20
	MaxTotalBytes    = 512 << 20
	MaxArchiveBytes  = 256 << 20
	MaxManifestBytes = 1 << 20
)

var ErrPackage = errors.New("invalid Karty package")

func Digest(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func validPath(name string) bool {
	if !fs.ValidPath(name) || name == "." || len(name) > 1024 || strings.Count(name, "/") > 32 ||
		strings.ContainsAny(name, `\:<>"|?*`) || !utf8.ValidString(name) {
		return false
	}

	for _, char := range name {
		if char < 32 || char == 127 {
			return false
		}
	}

	for part := range strings.SplitSeq(name, "/") {
		if strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return false
		}

		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" ||
			(len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9') {
			return false
		}
	}

	return true
}

func excluded(name string) bool {
	switch name {
	case ".git", ".karty", "dist", "node_modules", ".codex", ".agents":
		return true
	}

	return false
}

func read(ctx context.Context, root *os.Root, name string, limit int64) ([]byte, error) {
	if !validPath(name) {
		return nil, fmt.Errorf("%w: invalid source path %q", ErrPackage, name)
	}

	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}

	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("%w: oversized or nonregular source %s", ErrPackage, name)
	}

	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buffer strings.Builder
	stream := io.LimitReader(file, limit+1)
	chunk := make([]byte, 32<<10)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		count, err := stream.Read(chunk)
		if int64(buffer.Len()+count) > limit {
			return nil, ErrPackage
		}

		_, _ = buffer.Write(chunk[:count])
		if errors.Is(err, io.EOF) {
			return []byte(buffer.String()), nil
		}

		if err != nil {
			return nil, err
		}
	}
}

func collect(ctx context.Context, directory string) (map[string][]byte, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	files := map[string][]byte{}
	var total int64

	err = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if name == "." {
			return nil
		}

		if excluded(entry.Name()) {
			if entry.IsDir() {
				return fs.SkipDir
			}

			return nil
		}

		if !validPath(name) || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: unsupported source path %s", ErrPackage, name)
		}

		if entry.IsDir() {
			return nil
		}

		if name == InventoryName || len(files) >= MaxFiles-1 {
			return ErrPackage
		}

		limit := int64(MaxFileBytes)
		if name == ManifestName || name == "go.mod" {
			limit = MaxManifestBytes
		}

		data, err := read(ctx, root, name, limit)
		if err != nil {
			return err
		}

		total += int64(len(data))
		if total > MaxTotalBytes {
			return ErrPackage
		}

		files[name] = data

		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := validateNames(files); err != nil {
		return nil, err
	}

	return files, nil
}

func validateNames(files map[string][]byte) error {
	paths, components := map[string]bool{}, map[string]string{}
	for name := range files {
		key := strings.ToLower(name)
		if paths[key] {
			return fmt.Errorf("%w: case-colliding source %s", ErrPackage, name)
		}

		paths[key] = true
		parts := strings.Split(name, "/")
		for index := range parts {
			prefix := strings.Join(parts[:index+1], "/")
			lower := strings.ToLower(prefix)
			if prior, exists := components[lower]; exists && prior != prefix {
				return fmt.Errorf("%w: case-colliding source component %s", ErrPackage, prefix)
			}

			components[lower] = prefix
		}
	}

	return nil
}
