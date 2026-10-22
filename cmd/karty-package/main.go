package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/karty-game/karty-packages/internal/kpkg"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: karty-package list|resolve|verify-tag|check|pack")
	}

	repository, err := os.Getwd()
	if err != nil {
		return err
	}

	var result any
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return errors.New("usage: karty-package list")
		}

		result, err = kpkg.Discover(ctx, repository)
	case "resolve", "check", "verify-tag":
		if len(args) != 2 {
			return fmt.Errorf("usage: karty-package %s <package-or-tag>", args[0])
		}

		if args[0] == "verify-tag" {
			result, err = kpkg.VerifyTag(ctx, repository, args[1])
		} else {
			result, err = kpkg.Resolve(ctx, repository, args[1])
		}
	case "pack":
		flags := flag.NewFlagSet("pack", flag.ContinueOnError)
		directory := flags.String("output", "dist/releases", "artifact output directory")
		commit := flags.String("commit", os.Getenv("GITHUB_SHA"), "optional source commit SHA")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}

		if flags.NArg() != 1 {
			return errors.New("usage: karty-package pack [--output dist/releases] [--commit SHA] <package-or-tag>")
		}

		result, err = kpkg.Pack(ctx, repository, flags.Arg(0), *directory, *commit)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}

	if err != nil {
		return err
	}

	return json.NewEncoder(output).Encode(result)
}
