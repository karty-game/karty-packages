package controllers

import "math"

type Vec2 struct{ X, Y float64 }

// PlanarOptions selects screen-axis movement and following-camera behavior.
type PlanarOptions struct {
	Speed, RunSpeed, FollowRate float64
	FollowOffset                Vec2
	Zoom, MinZoom, MaxZoom      float64
	TargetTolerance             float64
}

func DefaultTopDown2DOptions() PlanarOptions {
	return PlanarOptions{Speed: 108.8, RunSpeed: 176, FollowRate: .12, Zoom: 1, MinZoom: .25, MaxZoom: 4, TargetTolerance: .5}
}

// Planar owns movement and following-camera state. Collision, navigation and
// animation remain game-owned. Step proposes movement; Follow uses accepted positions.
type Planar struct {
	Input
	options  PlanarOptions
	target   Vec2
	targeted bool
}

func NewTopDown2D(o PlanarOptions) (*Planar, error) { return newPlanar(o) }
func newPlanar(o PlanarOptions) (*Planar, error) {
	c := &Planar{}
	if err := c.Configure(o); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Planar) Configure(o PlanarOptions) error {
	for _, v := range [...]float64{o.Speed, o.RunSpeed, o.FollowRate, o.FollowOffset.X, o.FollowOffset.Y, o.Zoom, o.MinZoom, o.MaxZoom, o.TargetTolerance} {
		if !finite(v) {
			return ErrOptions
		}
	}
	if o.Speed < 0 || o.Speed > 10000 || o.RunSpeed < 0 || o.RunSpeed > 10000 || o.FollowRate < 0 || o.FollowRate > 1 || o.MinZoom <= 0 ||
		o.MaxZoom < o.MinZoom ||
		o.Zoom < o.MinZoom ||
		o.Zoom > o.MaxZoom ||
		o.TargetTolerance < 0 {
		return ErrOptions
	}
	c.options = o
	c.radius, c.deadzone = 96, .15
	c.Reset()
	return nil
}

func (c *Planar) Options() PlanarOptions { return c.options }
func (c *Planar) SetTarget(p Vec2) {
	if finite(p.X) && finite(p.Y) {
		c.target, c.targeted = p, true
	}
}
func (c *Planar) CancelTarget()   { c.targeted = false }
func (c *Planar) HasTarget() bool { return c.targeted }
func (c *Planar) Reset()          { c.Input.Reset(); c.CancelTarget() }
func (c *Planar) Speed() float64 {
	if c.Running() {
		return c.options.RunSpeed
	}
	return c.options.Speed
}

func (c *Planar) Step(accepted Vec2) Vec2 {
	if !finite(accepted.X) || !finite(accepted.Y) {
		c.Reset()
		return Vec2{}
	}
	x, y := c.keyboardAxes()
	tx, ty := c.analog(0)
	d := Vec2{float64(x + tx), float64(y + ty)}
	length := math.Hypot(d.X, d.Y)
	step := c.Speed() / 60
	if length > 0 {
		c.CancelTarget()
		step *= min(1, length)
	} else if c.targeted {
		d = Vec2{c.target.X - accepted.X, c.target.Y - accepted.Y}
		length = math.Hypot(d.X, d.Y)
		if length <= c.options.TargetTolerance {
			c.CancelTarget()
			return Vec2{}
		}
		step = min(step, length)
	}
	if length == 0 {
		return Vec2{}
	}
	return Vec2{d.X / length * step, d.Y / length * step}
}

func (c *Planar) Follow(current, accepted Vec2, snap bool) Vec2 {
	goal := Vec2{accepted.X + c.options.FollowOffset.X, accepted.Y + c.options.FollowOffset.Y}
	if snap {
		return goal
	}
	return Vec2{current.X + (goal.X-current.X)*c.options.FollowRate, current.Y + (goal.Y-current.Y)*c.options.FollowRate}
}

// ZoomBy scales the 2D camera within the configured bounds.
func (c *Planar) ZoomBy(factor float64) float64 {
	if finite(factor) && factor > 0 {
		c.options.Zoom = max(c.options.MinZoom, min(c.options.MaxZoom, c.options.Zoom*factor))
	}
	return c.options.Zoom
}

// ScreenToGround inverts the preset camera at Z=0. Pass the current camera focus
// and logical viewport, not CSS display dimensions. False means invalid input.
func (c *Planar) ScreenToGround(screen, focus Vec2, width, height float64) (Vec2, bool) {
	if !finite(screen.X) || !finite(screen.Y) || !finite(focus.X) || !finite(focus.Y) || !finite(width) || !finite(height) || width <= 0 ||
		height <= 0 {
		return Vec2{}, false
	}
	dx, dy := screen.X-width/2, screen.Y-height/2
	return Vec2{focus.X + dx/c.options.Zoom, focus.Y + dy/c.options.Zoom}, true
}
