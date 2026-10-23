// controller-sdk emits the package-owned typed adapter through KPR1.
package main

import (
	"fmt"
	"os"

	"github.com/karty-game/controllers-3d/bridge"
)

func main() {
	if err := bridge.Process(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
