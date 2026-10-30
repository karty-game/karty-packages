package character

import (
	"math"
	"testing"
)

func step(c *Controller, in Input) Output { return c.Step(1.0/60, in) }

func TestMovementAndPosture(t *testing.T) {
	for _, test := range []struct {
		name string
		in   Input
		clip string
		rate float64
	}{
		{"idle", Input{Grounded: true}, "idle", 1},
		{"walk", Input{Grounded: true, Forward: 1.2}, "walk", 1},
		{"run", Input{Grounded: true, Forward: 4.8}, "run", 1},
		{"backward", Input{Grounded: true, Forward: -2.4}, "walk", -1},
		{"left", Input{Grounded: true, Right: -2.4}, "strafe-left", 1},
		{"right", Input{Grounded: true, Right: 2.4}, "strafe-right", 1},
		{"diagonal", Input{Grounded: true, Right: .8, Forward: .8}, "walk", 1},
		{"crouch", Input{Grounded: true, Stance: Crouching}, "crouch", 1},
		{"crouch-walk", Input{Grounded: true, Stance: Crouching, Forward: 1}, "crouch-walk", 1},
		{"crawl", Input{Grounded: true, Stance: Prone, Forward: .5}, "crawl", 1},
		{"crawl-hold", Input{Grounded: true, Stance: Prone}, "crawl", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			var c Controller
			o := step(&c, test.in)
			if o.Base != test.clip || math.Signbit(o.Rate) != math.Signbit(test.rate) || (o.Rate == 0) != (test.rate == 0) {
				t.Fatalf("unexpected: %+v", o)
			}
		})
	}
}

func TestAirborneAndLanding(t *testing.T) {
	var c Controller
	step(&c, Input{Grounded: true})
	o := step(&c, Input{Vertical: 4.3})
	if o.State != "takeoff" {
		t.Fatal(o)
	}
	for range 10 {
		o = step(&c, Input{Vertical: 2})
	}
	if o.State != "rising" || o.Base != "jump-air" {
		t.Fatal(o)
	}
	o = step(&c, Input{Vertical: -1})
	if o.State != "falling" || o.Base != "jump-air" {
		t.Fatal(o)
	}
	o = step(&c, Input{Grounded: true})
	if o.State != "landing" {
		t.Fatal(o)
	}
	o = step(&c, Input{Grounded: true, Forward: 1.2})
	if o.State != "walk" {
		t.Fatalf("movement must interrupt landing: %+v", o)
	}
	// Stepping off a ledge must enter falling without playing takeoff.
	o = step(&c, Input{Vertical: -2})
	if o.State != "falling" {
		t.Fatal(o)
	}
	step(&c, Input{Grounded: true})
	for range 30 {
		o = step(&c, Input{Grounded: true})
	}
	if o.State != "idle" {
		t.Fatal(o)
	}
}

func TestPistolActionCompletionAndInterruption(t *testing.T) {
	var c Controller
	in := Input{Grounded: true, Equipment: Pistol, Forward: 1.2}
	o := step(&c, in)
	if o.Base != "walk" || o.Upper != "pistol-idle" {
		t.Fatal(o)
	}
	in.Request = Reload
	o = step(&c, in)
	if o.Upper != "pistol-reload" {
		t.Fatal(o)
	}
	// Firing during reload is rejected, rather than queued for a surprise shot.
	in.Request = Fire
	o = step(&c, in)
	if o.Upper != "pistol-reload" || o.Base != "walk" {
		t.Fatal(o)
	}
	in.Request = None
	for range 130 {
		o = step(&c, in)
	}
	if o.Upper != "pistol-idle" {
		t.Fatal(o)
	}
	in.Request = Fire
	o = step(&c, in)
	if o.Upper != "pistol-fire" {
		t.Fatal(o)
	}
	in.Request = None
	in.Stance = Crouching
	o = step(&c, in)
	if o.Base != "crouch-walk" || o.Upper != "pistol-fire" {
		t.Fatal(o)
	}
	in.Stance = Prone
	o = step(&c, in)
	if o.Upper != "" {
		t.Fatal(o)
	}
	in.Stance = Standing
	in.Request = Reload
	step(&c, in)
	in.Grounded = false
	in.Vertical = 2
	in.Request = None
	o = step(&c, in)
	if o.Upper != "" {
		t.Fatal(o)
	}
	// Equipment removal cancels actions and cannot leave a weapon pose latched.
	in.Grounded = true
	in.Vertical = 0
	in.Request = Reload
	step(&c, in)
	in.Equipment = Empty
	in.Request = None
	o = step(&c, in)
	if o.Upper != "" {
		t.Fatal(o)
	}
}

func TestInteractionAndDeathPriority(t *testing.T) {
	var c Controller
	in := Input{Grounded: true, Request: Use}
	o := step(&c, in)
	if o.Base != "use" {
		t.Fatal(o)
	}
	in.Request = None
	in.Forward = 1.2
	o = step(&c, in)
	if o.Base != "walk" {
		t.Fatal(o)
	}
	in.Forward = 0
	in.Request = Use
	step(&c, in)
	in.Request = None
	for range 160 {
		o = step(&c, in)
	}
	if o.Base != "idle" {
		t.Fatalf("interact must complete: %+v", o)
	}
	for death := uint8(1); death <= 3; death++ {
		c.Reset()
		o = step(&c, Input{Grounded: true, Death: death, Equipment: Pistol, Request: Reload})
		want := []string{"death-a", "death-c", "death-d"}[death-1]
		if o.Base != want || !o.Dead || o.Upper != "" {
			t.Fatal(o)
		}
		for range 600 {
			o = step(&c, Input{Grounded: true, Forward: 4, Request: Fire})
		}
		if o.Base != want || o.BaseTime != duration(want) {
			t.Fatalf("death must hold: %+v", o)
		}
		serial := o.Transition
		c.Reset()
		o = step(&c, Input{Grounded: true})
		if o.Dead || o.Base != "idle" || o.Transition <= serial {
			t.Fatal(o)
		}
	}
}

func TestStablePhasesBoundsAndAllocations(t *testing.T) {
	var c Controller
	in := Input{Grounded: true, Forward: 1.2, Equipment: Pistol}
	first := step(&c, in)
	for range 100 {
		o := step(&c, in)
		if o.Transition != first.Transition || o.BaseTime <= first.BaseTime {
			t.Fatal(o)
		}
	}
	previous := c.Out
	if got := c.Step(math.NaN(), in); got != previous {
		t.Fatal(got)
	}
	in.Right = math.Inf(1)
	if got := step(&c, in); got != previous {
		t.Fatal(got)
	}
	in.Right = 0
	if n := testing.AllocsPerRun(100, func() { step(&c, in) }); n != 0 {
		t.Fatalf("steady step allocations: %v", n)
	}
}

func TestCadenceFollowsDistanceWithoutRetimingReload(t *testing.T) {
	var c Controller
	in := Input{Grounded: true, Forward: 1.2, Equipment: Pistol, Request: Reload}
	step(&c, in)
	in.Request = None
	for range 60 {
		step(&c, in)
	}
	if math.Abs(c.Out.BaseTime*.7345470488-1.2*61/60) > 1e-7 {
		t.Fatalf("walk phase not matched to travelled distance: %+v", c.Out)
	}
	actionBefore := c.Out.UpperTime
	in.Forward = 4.8
	for range 30 {
		step(&c, in)
	}
	if c.Out.Base != "run" || c.Out.Upper != "pistol-reload" {
		t.Fatal(c.Out)
	}
	if math.Abs(c.Out.UpperTime-actionBefore-.5) > 1e-8 {
		t.Fatal("locomotion speed changed reload timing")
	}
	// Analog half-speed movement must advance half as much through the cycle.
	var slow, fast Controller
	for range 60 {
		step(&slow, Input{Grounded: true, Forward: .4})
		step(&fast, Input{Grounded: true, Forward: .8})
	}
	if math.Abs(fast.Out.BaseTime/slow.Out.BaseTime-2) > 1e-8 {
		t.Fatal("cadence does not follow analog speed")
	}
	phase := fast.Out.BaseTime
	step(&fast, Input{Grounded: true})
	if fast.Out.Base != "idle" {
		t.Fatal("stopped pawn keeps stepping", phase)
	}
}

// In normal play the walking pace is 1.4m/s and sprint is 4.2m/s. The old
// 2.4m/s jog selection ran at 0.58x and stretched the flight poses.
func TestJogNeverSelectedAtWalkingSpeed(t *testing.T) {
	for _, speed := range []float64{.3, 1.4, 2.4, 3.59, 3.6, 4.2} {
		var c Controller
		o := step(&c, Input{Grounded: true, Forward: speed})
		if speed < 3.6 && o.Base != "walk" || speed >= 3.6 && (o.Base != "run" || o.Rate < .87) {
			t.Fatalf("speed %v: %+v", speed, o)
		}
	}
}
