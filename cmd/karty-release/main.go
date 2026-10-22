package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/karty-game/karty-packages/internal/kpkg"
	"golang.org/x/mod/semver"
)

type selection struct {
	kpkg.Identity
	Commit     string `json:"commit"`
	Prerelease bool   `json:"prerelease"`
}

type gitRunner func(context.Context, ...string) (string, error)

func selectRelease(ctx context.Context, repository string, environment map[string]string, git gitRunner) (selection, error) {
	selector := environment["PACKAGE_NAME"] + "-v" + environment["PACKAGE_VERSION"]
	pushed := environment["GITHUB_EVENT_NAME"] == "push"
	if pushed {
		if environment["GITHUB_REF_TYPE"] != "tag" {
			return selection{}, errors.New("package releases require a tag push")
		}
		selector = environment["GITHUB_REF_NAME"]
	} else if environment["GITHUB_EVENT_NAME"] != "workflow_dispatch" {
		return selection{}, errors.New("expected a tag push or manual workflow dispatch")
	}
	identity, err := kpkg.VerifyTag(ctx, repository, selector)
	if err != nil {
		return selection{}, err
	}
	commit, err := git(ctx, "rev-parse", "HEAD")
	if err != nil {
		return selection{}, err
	}
	commit = strings.TrimSpace(commit)
	if len(commit) != 40 || strings.IndexFunc(commit, func(char rune) bool {
		return !strings.ContainsRune("0123456789abcdef", char)
	}) >= 0 {
		return selection{}, errors.New("expected a full source commit SHA")
	}
	_, err = git(ctx, "show-ref", "--verify", "--quiet", "refs/tags/"+identity.Tag)
	if err != nil {
		var status interface{ ExitCode() int }
		if pushed || !errors.As(err, &status) || status.ExitCode() != 1 {
			return selection{}, err
		}
	} else {
		target, err := git(ctx, "rev-parse", "refs/tags/"+identity.Tag+"^{commit}")
		if err != nil {
			return selection{}, err
		}
		if strings.TrimSpace(target) != commit {
			return selection{}, errors.New("package tag points to a different source commit")
		}
	}
	return selection{Identity: identity, Commit: commit, Prerelease: semver.Prerelease("v"+identity.Version) != ""}, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	repository, err := os.Getwd()
	if err != nil {
		return err
	}
	environment := map[string]string{}
	for _, name := range []string{"GITHUB_EVENT_NAME", "GITHUB_REF_TYPE", "GITHUB_REF_NAME", "PACKAGE_NAME", "PACKAGE_VERSION"} {
		environment[name] = os.Getenv(name)
	}
	git := func(ctx context.Context, args ...string) (string, error) {
		command := exec.CommandContext(ctx, "git", args...)
		command.Dir = repository
		data, err := command.Output()
		return string(data), err
	}
	result, err := selectRelease(context.Background(), repository, environment, git)
	if err != nil {
		return err
	}
	if output := os.Getenv("GITHUB_OUTPUT"); output != "" {
		file, err := os.OpenFile(output, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, writeErr := fmt.Fprintf(file, "package=%s\nversion=%s\ntag=%s\narchive=%s\ncommit=%s\nprerelease=%t\n",
			result.Package, result.Version, result.Tag, result.Archive, result.Commit, result.Prerelease)
		closeErr := file.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
