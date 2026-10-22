package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type gitStatus int

func (status gitStatus) Error() string { return "git lookup failed" }
func (status gitStatus) ExitCode() int { return int(status) }

func releaseFixture(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	packageDirectory := filepath.Join(directory, "packages", "demo")
	if err := os.MkdirAll(packageDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"karty-package.toml": "format_version = 1\nname = 'demo'\nversion = '0.1.0'\n",
		"go.mod":             "module example.com/demo\n\ngo 1.27.0\n",
	} {
		if err := os.WriteFile(filepath.Join(packageDirectory, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

func TestReleaseSelection(t *testing.T) {
	for _, test := range []struct {
		name, event, tagTarget string
		missing                gitStatus
		wantError              bool
	}{
		{name: "manual-new-tag", event: "workflow_dispatch", missing: 1},
		{name: "pushed-tag", event: "push", tagTarget: strings.Repeat("1", 40)},
		{name: "manual-existing-tag", event: "workflow_dispatch", tagTarget: strings.Repeat("1", 40)},
		{name: "moved-tag", event: "push", tagTarget: strings.Repeat("2", 40), wantError: true},
		{name: "manual-moved-tag", event: "workflow_dispatch", tagTarget: strings.Repeat("2", 40), wantError: true},
		{name: "push-missing-tag", event: "push", missing: 1, wantError: true},
		{name: "git-fatal", event: "workflow_dispatch", missing: 128, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := releaseFixture(t)
			env := map[string]string{"GITHUB_EVENT_NAME": test.event, "GITHUB_REF_TYPE": "tag", "GITHUB_REF_NAME": "demo-v0.1.0", "PACKAGE_NAME": "demo", "PACKAGE_VERSION": "0.1.0"}
			git := func(_ context.Context, args ...string) (string, error) {
				if args[len(args)-1] == "HEAD" {
					return strings.Repeat("1", 40), nil
				}
				if test.missing != 0 {
					return "", test.missing
				}
				return test.tagTarget, nil
			}
			result, err := selectRelease(t.Context(), root, env, git)
			if (err != nil) != test.wantError {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if err == nil && (result.Tag != "demo-v0.1.0" || result.Commit != strings.Repeat("1", 40) || result.Prerelease) {
				t.Fatalf("wrong release: %+v", result)
			}
		})
	}
}

func TestReleaseSelectionRejectsInvalidInputBeforeGit(t *testing.T) {
	for _, env := range []map[string]string{
		{"GITHUB_EVENT_NAME": "push", "GITHUB_REF_TYPE": "branch", "GITHUB_REF_NAME": "demo-v0.1.0"},
		{"GITHUB_EVENT_NAME": "workflow_dispatch", "PACKAGE_NAME": "demo", "PACKAGE_VERSION": "0.2.0"},
		{"GITHUB_EVENT_NAME": "workflow_dispatch", "PACKAGE_NAME": "../demo", "PACKAGE_VERSION": "0.1.0"},
		{"GITHUB_EVENT_NAME": "pull_request"},
	} {
		_, err := selectRelease(t.Context(), releaseFixture(t), env, func(context.Context, ...string) (string, error) {
			t.Fatal("invalid release reached Git")
			return "", errors.New("unexpected Git invocation")
		})
		if err == nil {
			t.Fatal("invalid release accepted")
		}
	}
}
