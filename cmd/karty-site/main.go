package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/karty-game/karty-packages/website"
)

const maxMasterBytes = 16 << 20

func readMaster(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, maxMasterBytes+1))
	if err != nil {
		return nil, err
	}
	if len(contents) > maxMasterBytes {
		return nil, fmt.Errorf("master.json exceeds %d bytes", maxMasterBytes)
	}
	return contents, nil
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("karty-site", flag.ContinueOnError)
	master := flags.String("master", "dist/catalog/master.json", "Input package catalog JSON")
	output := flags.String("output", "dist/catalog", "Output directory for HTML, Markdown and master.json")
	serve := flags.String("serve", "", "Preview on a loopback address, for example 127.0.0.1:8082")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: karty-site [--master master.json] [--output directory] [--serve 127.0.0.1:8082]")
	}
	contents, err := readMaster(*master)
	if err != nil {
		return err
	}
	if err := website.Generate(contents, *output); err != nil {
		return err
	}
	fmt.Printf("Generated package website in %s\n", *output)
	if *serve != "" {
		return website.Serve(ctx, *output, *serve)
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
