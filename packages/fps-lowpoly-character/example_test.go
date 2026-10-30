package character_test

import (
	"fmt"

	character "github.com/karty-game/fps-lowpoly-character"
)

func ExampleController() {
	// Each NPC owns its controller. Feed accepted movement after the motor tick,
	// whether the desired motion came from player input or AI steering.
	var guard, scientist character.Controller
	walking := guard.Step(1.0/60, character.Input{
		Grounded: true, Forward: 1.4, Equipment: character.Pistol,
		Request: character.Reload,
	})
	idle := scientist.Step(1.0/60, character.Input{Grounded: true})
	fmt.Println(walking.Base, walking.Upper)
	fmt.Println(idle.Base, idle.Upper == "")
	// Output:
	// walk pistol-reload
	// idle true
}
