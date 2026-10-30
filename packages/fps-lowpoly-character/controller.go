// Package character selects animation from gameplay facts. It has no SDK,
// renderer or browser dependency. Times are seconds on the original source bank.
package character

import "math"

type Stance uint8

const (
	Standing Stance = iota
	Crouching
	Prone
)

type Equipment uint8

const (
	Empty Equipment = iota
	Pistol
)

type Action uint8

const (
	None Action = iota
	Fire
	Reload
	Use
)

// Input movement is accepted velocity in the character's local space, not held
// keys. Requests are edge-triggered; death overrides everything until Reset.
type Input struct {
	Right, Forward, Vertical float64
	Grounded                 bool
	Stance                   Stance
	Equipment                Equipment
	Request                  Action
	Death                    uint8
}

type Output struct {
	State, Base, Upper  string
	BaseTime, UpperTime float64
	Rate                float64
	Dead                bool
	Transition          uint32
}

// Controller retains animation state for one actor. Its zero value is ready.
// Step selects clips only; a game supplies physics and a renderer samples poses.
type Controller struct {
	Out           Output
	action        Action
	actionTime    float64
	phase         string
	phaseTime     float64
	wasGrounded   bool
	initialized   bool
	previousEquip Equipment
}

// Reset respawns this animation state and invalidates any previous transition.
func (c *Controller) Reset() {
	serial := c.Out.Transition + 1
	*c = Controller{}
	c.Out.Transition = serial
}

// Step advances by seconds. Invalid values leave all controller state unchanged.
// Requests are edge-triggered. Death is zero (alive) or source death 1, 2 or 3.
func (c *Controller) Step(dt float64, in Input) Output {
	if math.IsNaN(dt) || math.IsInf(dt, 0) || dt <= 0 || dt > .1 {
		return c.Out
	}
	if !finite(in.Right) || !finite(in.Forward) || !finite(in.Vertical) || in.Stance > Prone || in.Equipment > Pistol {
		return c.Out
	}
	if in.Request > Use || in.Death > 3 {
		return c.Out
	}
	if c.Out.Dead {
		c.Out.BaseTime = min(c.Out.BaseTime+dt, duration(c.Out.Base))
		return c.Out
	}
	if in.Death > 0 {
		clip := "death-a"
		if in.Death == 2 {
			clip = "death-c"
		}
		if in.Death >= 3 {
			clip = "death-d"
		}
		c.selectClips("dead", clip, "", 1)
		c.Out.Dead = true
		return c.Out
	}
	moving := math.Hypot(in.Right, in.Forward) > .12
	if !in.Grounded {
		if c.phase != "rising" && c.phase != "falling" {
			c.phaseTime = 0
		}
		c.phase = "falling"
		if in.Vertical > .15 {
			c.phase = "rising"
		}
	} else if c.initialized && !c.wasGrounded {
		c.phase, c.phaseTime = "landing", 0
	} else if c.phase != "landing" || c.phaseTime >= .35 || moving {
		c.phase, c.phaseTime = "grounded", 0
	}
	c.wasGrounded, c.initialized = in.Grounded, true
	c.phaseTime += dt
	state, base, rate := "idle", "idle", 1.0
	switch {
	case c.phase == "rising":
		state, base = "rising", "jump-air"
		if c.phaseTime <= .12 {
			state, base = "takeoff", "jump-start"
		}
	case c.phase == "falling":
		state, base = "falling", "jump-air"
	case c.phase == "landing":
		state, base = "landing", "jump-land"
	case in.Stance == Prone:
		state, base = "crawl-idle", "crawl"
		if moving {
			state = "crawl"
		} else {
			rate = 0
		}
	case in.Stance == Crouching:
		state, base = "crouch", "crouch"
		if moving {
			state, base = "crouch-walk", "crouch-walk"
		}
	case moving:
		state, base = "walk", "walk"
		if math.Abs(in.Right) > math.Abs(in.Forward)*1.35 {
			state, base = "strafe-right", "strafe-right"
			if in.Right < 0 {
				state, base = "strafe-left", "strafe-left"
			}
			// The source jog has an airborne phase. Do not stretch it to walking
			// speeds: use it only near its measured native travel speed.
		} else if math.Hypot(in.Right, in.Forward) >= 3.6 && in.Forward > 0 {
			state, base = "run", "run"
		}
	}
	if reference := strideSpeed(base); reference > 0 && moving {
		rate = math.Min(math.Hypot(in.Right, in.Forward)/reference, 8)
	}
	if moving && (base == "walk" || base == "crouch-walk" || base == "crawl") && in.Forward < -.12 {
		rate = -rate
	}
	// Actions have independent time. Changing equipment cancels the old action;
	// airborne/prone posture cancels weapon use. Movement interrupts interact.
	weaponOK := in.Grounded && in.Stance != Prone && in.Equipment == Pistol
	if in.Equipment != c.previousEquip || !in.Grounded || in.Stance == Prone || c.action == Use && moving {
		c.action = None
	}
	c.previousEquip = in.Equipment
	if c.action != None {
		c.actionTime += dt
		if c.actionTime >= duration(actionClip(c.action)) {
			c.action = None
		}
	}
	if c.action == None && in.Grounded && in.Stance != Prone {
		if in.Request == Use && !moving || weaponOK && (in.Request == Fire || in.Request == Reload) {
			c.action, c.actionTime = in.Request, 0
		}
	}
	upper := ""
	if weaponOK {
		upper = "pistol-idle"
	}
	if c.action == Fire || c.action == Reload {
		upper = actionClip(c.action)
	}
	if c.action == Use {
		state, base, upper, rate = "interact", "use", "", 1
	}
	c.selectClips(state, base, upper, rate)
	if c.action == Use {
		c.Out.BaseTime = c.actionTime
	} else {
		c.Out.BaseTime += dt * math.Abs(rate)
	}
	if c.action != None {
		c.Out.UpperTime = c.actionTime
	} else {
		c.Out.UpperTime += dt
	}
	return c.Out
}

func (c *Controller) selectClips(state, base, upper string, rate float64) {
	if c.Out.State != state || c.Out.Base != base || c.Out.Upper != upper {
		c.Out.Transition++
	}
	if c.Out.Base != base {
		phase := 0.0
		if strideSpeed(c.Out.Base) > 0 && strideSpeed(base) > 0 {
			phase = math.Mod(c.Out.BaseTime/duration(c.Out.Base), 1)
		}
		c.Out.BaseTime = phase * duration(base)
	} else if c.Out.Rate*rate < 0 && strideSpeed(base) > 0 {
		// Reverse the sampling direction without jumping to a different pose.
		c.Out.BaseTime = duration(base) - math.Mod(c.Out.BaseTime, duration(base))
	}
	if c.Out.Upper != upper {
		c.Out.UpperTime = 0
	}
	c.Out.State, c.Out.Base, c.Out.Upper, c.Out.Rate = state, base, upper, rate
}

func actionClip(action Action) string {
	switch action {
	case Fire:
		return "pistol-fire"
	case Reload:
		return "pistol-reload"
	case Use:
		return "use"
	}
	return ""
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Historical calibration at 1.8m guard height. The inherited strafe calibration
// is known to be incorrect; see README.md before using lateral movement.
// Crawl uses supporting hands because its feet are not planted contacts.
func strideSpeed(alias string) float64 {
	switch alias {
	case "walk":
		return .7345470488
	case "run":
		return 4.109203331
	case "crouch-walk":
		return .5583784408
	case "crawl":
		return .3371791466
	case "strafe-left":
		return .4964817167
	case "strafe-right":
		return .4962987808
	}
	return 0
}
