package controllers

import (
	"errors"
	"math"
	"testing"
)

func TestTopDownAliasesTargetsAndFollowing(t *testing.T) {
	o := DefaultTopDown2DOptions()
	o.FollowOffset = Vec2{Y: -110}
	c, err := NewTopDown2D(o)
	if err != nil {
		t.Fatal(err)
	}
	c.Key(Forward, true)
	c.Key(ForwardAlt, true)
	c.Key(Forward, false)
	d := c.Step(Vec2{})
	near(t, d.Y, -108.8/60)
	c.Key(RunLeft, true)
	c.Key(RunRight, true)
	c.Key(RunLeft, false)
	near(t, c.Step(Vec2{}).Y, -176.0/60)
	c.Reset()
	c.SetTarget(Vec2{X: .75})
	d = c.Step(Vec2{})
	near(t, d.X, .75)
	if c.Step(Vec2{X: .75}) != (Vec2{}) || c.HasTarget() {
		t.Fatal("target overshot or retained after arrival")
	}
	c.SetTarget(Vec2{X: 20})
	c.Key(Left, true)
	if c.Step(Vec2{}).X >= 0 || c.HasTarget() {
		t.Fatal("keyboard did not take over target")
	}
	c.Reset()
	if got := c.Follow(Vec2{}, Vec2{X: 100, Y: 200}, true); got != (Vec2{100, 90}) {
		t.Fatalf("snap camera: %+v", got)
	}
	got := c.Follow(Vec2{}, Vec2{X: 100, Y: 200}, false)
	near(t, got.X, 12)
	near(t, got.Y, 10.8)
	ground, ok := c.ScreenToGround(Vec2{640, 360}, Vec2{100, 90}, 1280, 720)
	if !ok || ground != (Vec2{100, 90}) {
		t.Fatalf("screen center: %+v", ground)
	}
}

func TestTopDownOptionsAreTransactional(t *testing.T) {
	c, err := NewTopDown2D(DefaultTopDown2DOptions())
	if err != nil {
		t.Fatal(err)
	}
	before := c.Options()
	for _, change := range []func(*PlanarOptions){
		func(o *PlanarOptions) { o.Speed = -1 }, func(o *PlanarOptions) { o.FollowRate = math.NaN() },
		func(o *PlanarOptions) { o.MinZoom = 0 }, func(o *PlanarOptions) { o.MaxZoom = o.MinZoom / 2 },
	} {
		o := before
		change(&o)
		if !errors.Is(c.Configure(o), ErrOptions) || c.Options() != before {
			t.Fatal("invalid configuration changed state")
		}
	}
}

// Keep serial: Go 1.27 rejects AllocsPerRun inside parallel tests.
func TestSteadyUpdatesDoNotAllocate(t *testing.T) {
	c, err := NewTopDown2D(DefaultTopDown2DOptions())
	if err != nil {
		t.Fatal(err)
	}
	c.Key(Right, true)
	if got := testing.AllocsPerRun(100, func() {
		c.Step(Vec2{})
		c.Follow(Vec2{}, Vec2{1, 2}, false)
		c.ScreenToGround(Vec2{640, 360}, Vec2{}, 1280, 720)
	}); got != 0 {
		t.Fatalf("steady update allocated: %v", got)
	}
}

func near(t *testing.T, got, want float64) {
	t.Helper()
	if !finite(got) || math.Abs(got-want) > .00001 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
