// Command check-controller measures the archived controller in an actual TinyGo guest.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/karty-game/karty-packages/internal/controllercheck"
)

func main() {
	var options controllercheck.Options
	flag.StringVar(&options.SDKArchive, "sdk-archive", "", "immutable SDK zip")
	flag.StringVar(&options.SDKSHA256, "sdk-sha256", "", "expected SDK SHA-256")
	flag.StringVar(&options.PackageArchive, "package-archive", "", "immutable controllers-3d kpkg")
	flag.StringVar(&options.PackageSHA256, "package-sha256", "", "expected package SHA-256")
	flag.StringVar(&options.Go, "go", "go", "selected SDK's Go executable")
	flag.StringVar(&options.TinyGo, "tinygo", "tinygo", "selected SDK's TinyGo executable")
	flag.StringVar(&options.Output, "output", "dist/validation/controller-allocations.json", "successful provenance and measurement report")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := controllercheck.Run(ctx, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("controller allocation gate passed (14 actual TinyGo cases): " + options.Output)
}
