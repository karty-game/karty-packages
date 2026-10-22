package kpkg

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

type inventory struct {
	FormatVersion int          `json:"formatVersion"`
	Files         []fileRecord `json:"files"`
}

type fileRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func encodeArchive(ctx context.Context, files map[string][]byte) ([]byte, error) {
	names := make([]string, 0, len(files)+1)
	for name := range files {
		names = append(names, name)
	}

	slices.Sort(names)
	index := inventory{FormatVersion: 1, Files: make([]fileRecord, 0, len(files))}
	for _, name := range names {
		index.Files = append(index.Files, fileRecord{name, Digest(files[name]), int64(len(files[name]))})
	}

	data, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}

	if len(data) > MaxManifestBytes {
		return nil, fmt.Errorf("%w: inventory exceeds bounds", ErrPackage)
	}

	names = append(names, InventoryName)
	slices.Sort(names)
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.Modified = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
		header.SetMode(0644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}

		contents := files[name]
		if name == InventoryName {
			contents = data
		}

		if _, err := entry.Write(contents); err != nil {
			return nil, err
		}

		if buffer.Len() > MaxArchiveBytes {
			return nil, ErrPackage
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	if buffer.Len() > MaxArchiveBytes {
		return nil, ErrPackage
	}

	return buffer.Bytes(), nil
}
