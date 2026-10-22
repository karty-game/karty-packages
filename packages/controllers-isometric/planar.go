package controllers

import "math"

type Vec2 struct{ X, Y float64 }
type Vec3 struct{ X, Y, Z float64 }

// PlanarOptions selects camera-relative movement and fixed isometric projection.
type PlanarOptions struct {
	Speed, RunSpeed, FollowRate float64
	FollowOffset                Vec2
	CameraOffset                Vec3
	Yaw, Pitch                  float64
	Zoom, MinZoom, MaxZoom      float64
	TargetTolerance             float64
}

func DefaultIsometricOptions() PlanarOptions {
	return PlanarOptions{Speed: 4, RunSpeed: 5.2, FollowRate: .15, CameraOffset: Vec3{-18, -18, 14.696938},
		Yaw: math.Pi / 4, Pitch: -math.Pi / 6, Zoom: 13.2, MinZoom: 10, MaxZoom: 22, TargetTolerance: .01}
}

// Planar owns movement and following-camera state. Collision, navigation and
// animation remain game-owned. Step proposes movement; Follow uses accepted positions.
type Planar struct {
	Input
	options  PlanarOptions
	target   Vec2
	targeted bool
}

func NewIsometric(o PlanarOptions) (*Planar, error) { return newPlanar(o) }
func newPlanar(o PlanarOptions) (*Planar, error) {
	c := &Planar{}
	if err := c.Configure(o); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Planar) Configure(o PlanarOptions) error {
	for _, v := range [...]float64{o.Speed, o.RunSpeed, o.FollowRate, o.FollowOffset.X, o.FollowOffset.Y, o.CameraOffset.X, o.CameraOffset.Y, o.CameraOffset.Z, o.Yaw, o.Pitch, o.Zoom, o.MinZoom, o.MaxZoom, o.TargetTolerance} {
		if !finite(v) {
			return ErrOptions
		}
	}
	if o.Speed < 0 || o.Speed > 10000 || o.RunSpeed < 0 || o.RunSpeed > 10000 || o.FollowRate < 0 || o.FollowRate > 1 || o.MinZoom <= 0 ||
		o.MaxZoom < o.MinZoom ||
		o.Zoom < o.MinZoom ||
		o.Zoom > o.MaxZoom ||
		o.TargetTolerance < 0 ||
		(o.Pitch >= 0 || o.Pitch <= -math.Pi/2) {
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
		sine, cosine := math.Sin(c.options.Yaw), math.Cos(c.options.Yaw)
		d = Vec2{d.X*cosine - d.Y*sine, -d.X*sine - d.Y*cosine}
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

// CameraPosition is the 3D camera's fixed offset from its following focus.
func (c *Planar) CameraPosition(focus Vec2) Vec3 {
	o := c.options.CameraOffset
	return Vec3{focus.X + o.X, focus.Y + o.Y, o.Z}
}

// ZoomBy scales the orthographic height within the configured bounds.
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
	scale := c.options.Zoom / height
	dx *= scale
	dy *= -scale / math.Sin(-c.options.Pitch)
	sine, cosine := math.Sin(c.options.Yaw), math.Cos(c.options.Yaw)
	return Vec2{focus.X + dx*cosine + dy*sine, focus.Y - dx*sine + dy*cosine}, true
}
