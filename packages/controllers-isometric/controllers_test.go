package controllers

import (
	"errors"
	"math"
	"testing"
)

func TestIsometricBasisProjectionAndZoom(t *testing.T) {
	c, err := NewIsometric(DefaultIsometricOptions())
	if err != nil {
		t.Fatal(err)
	}
	c.Key(Forward, true)
	d := c.Step(Vec2{})
	near(t, d.X, 4.0/60/math.Sqrt2)
	near(t, d.Y, d.X)
	c.Key(Right, true)
	d = c.Step(Vec2{})
	near(t, d.X, 4.0/60)
	near(t, d.Y, 0)
	for _, factor := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		near(t, c.ZoomBy(factor), 13.2)
	}
	near(t, c.ZoomBy(100), 22)
	near(t, c.ZoomBy(.001), 10)
	for _, size := range []Vec2{{1280, 720}, {720, 1280}, {2560, 720}} {
		focus := Vec2{12, 4}
		ground, ok := c.ScreenToGround(Vec2{size.X/2 + size.Y/10, size.Y/2 - size.Y/20}, focus, size.X, size.Y)
		if !ok {
			t.Fatal("valid viewport rejected")
		}
		near(t, ground.X, focus.X+math.Sqrt2)
		near(t, ground.Y, focus.Y)
	}
	if _, ok := c.ScreenToGround(Vec2{}, Vec2{}, 0, 720); ok {
		t.Fatal("zero viewport accepted")
	}
	position := c.CameraPosition(Vec2{12, 4})
	if position != (Vec3{-6, -14, 14.696938}) {
		t.Fatalf("camera offset: %+v", position)
	}
	o := c.Options()
	o.Pitch = 0
	if !errors.Is(c.Configure(o), ErrOptions) || c.Options().Pitch == 0 {
		t.Fatal("singular projection accepted")
	}
}

// Keep this serial: Go 1.27 rejects AllocsPerRun inside parallel tests.
// Keep serial: Go 1.27 rejects AllocsPerRun inside parallel tests.
func TestSteadyUpdatesDoNotAllocate(t *testing.T) {
	c, err := NewIsometric(DefaultIsometricOptions())
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
