package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadMasterBounds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "master.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxMasterBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readMaster(path); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected bounded input error, got %v", err)
	}
}

func TestRunRejectsPositionalArguments(t *testing.T) {
	if err := run(context.Background(), []string{"unexpected"}); err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected argument error, got %v", err)
	}
}
