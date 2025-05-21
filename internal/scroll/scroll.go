// SPDX-License-Identifier: Unlicense OR MIT

// Most of the code in this package and the fling package are from Gio,
// with minor modifications to change to behaviour of the scroller.
package scroll

import (
	"math"
	"runtime"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/oligo/gvcode/internal/scroll/fling"
)

// Scroll detects scroll gestures and reduces them to
// scroll distances. Scroll recognizes mouse wheel
// movements as well as drag and fling touch gestures.
//
// This is a modified version of the original [gesture.Scroll] in Gio.
// The most importantly change is that scrolling axis is detected, not
// passed by user.
type Scroll struct {
	dragging  bool
	estimator fling.Extrapolation
	flinger   fling.Animation
	pid       pointer.ID
	last      int
	lastAxis  Axis
	// Leftover scroll.
	scroll float32
}

type ScrollState uint8

type Axis uint8

const (
	Horizontal Axis = iota
	Vertical
)

const (
	// StateIdle is the default scroll state.
	StateIdle ScrollState = iota
	// StateDragging is reported during drag gestures.
	StateDragging
	// StateFlinging is reported when a fling is
	// in progress.
	StateFlinging
)

const touchSlop = unit.Dp(3)

// Add the handler to the operation list to receive scroll events.
// The bounds variable refers to the scrolling boundaries
// as defined in [pointer.Filter].
func (s *Scroll) Add(ops *op.Ops) {
	event.Op(ops, s)
}

// Stop any remaining fling movement.
func (s *Scroll) Stop() {
	s.flinger = fling.Animation{}
}

// Direction returns the last scrolling axis detected by Update.
func (s *Scroll) Direction() Axis {
	return s.lastAxis
}

// Update state and report the scroll distance along axis.
func (s *Scroll) Update(cfg unit.Metric, q input.Source, t time.Time, scrollx, scrolly pointer.ScrollRange) int {
	total := 0
	s.lastAxis = Vertical
	f := pointer.Filter{
		Target:  s,
		Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Scroll | pointer.Cancel,
		ScrollX: scrollx,
		ScrollY: scrolly,
	}
	for {
		evt, ok := q.Event(f)
		if !ok {
			break
		}
		e, ok := evt.(pointer.Event)
		if !ok {
			continue
		}

		if e.Modifiers.Contain(key.ModShift) {
			s.lastAxis = Horizontal
		} else if e.Scroll.X != 0.0 {
			s.lastAxis = Horizontal
		}

		//slog.Info("scrolling started!!!", "eventKind", e.Kind, "position", e.Position, "axis", s.lastAxis)

		switch e.Kind {
		case pointer.Press:
			if s.dragging {
				break
			}
			// Only scroll on touch drags, or on Android where mice
			// drags also scroll by convention.
			if e.Source != pointer.Touch && runtime.GOOS != "android" {
				break
			}
			s.Stop()
			s.estimator = fling.Extrapolation{}
			v := s.val(s.lastAxis, e.Position)
			s.last = int(math.Round(float64(v)))
			s.estimator.Sample(e.Time, v)
			s.dragging = true
			s.pid = e.PointerID
		case pointer.Release:
			if s.pid != e.PointerID {
				break
			}
			fling := s.estimator.Estimate()
			if slop, d := float32(cfg.Dp(touchSlop)), fling.Distance; d < -slop || d > slop {
				s.flinger.Start(cfg, t, fling.Velocity)
			}
			fallthrough
		case pointer.Cancel:
			s.dragging = false
		case pointer.Scroll:
			switch s.lastAxis {
			case Horizontal:
				s.scroll += e.Scroll.X
			case Vertical:
				s.scroll += e.Scroll.Y
			}
			iscroll := int(s.scroll)
			s.scroll -= float32(iscroll)
			total += iscroll
		case pointer.Drag:
			if !s.dragging || s.pid != e.PointerID {
				continue
			}
			val := s.val(s.lastAxis, e.Position)
			s.estimator.Sample(e.Time, val)
			v := int(math.Round(float64(val)))
			dist := s.last - v
			if e.Priority < pointer.Grabbed {
				slop := cfg.Dp(touchSlop)
				if dist := dist; dist >= slop || -slop >= dist {
					q.Execute(pointer.GrabCmd{Tag: s, ID: e.PointerID})
				}
			} else {
				s.last = v
				total += dist
			}
		}
	}
	total += s.flinger.Tick(t)
	if s.flinger.Active() {
		q.Execute(op.InvalidateCmd{})
	}
	return total
}

func (s *Scroll) val(axis Axis, p f32.Point) float32 {
	switch axis {
	case Horizontal:
		return p.X
	case Vertical:
		return p.Y
	default:
		return 0
	}
}

func (a Axis) String() string {
	switch a {
	case Horizontal:
		return "Horizontal"
	case Vertical:
		return "Vertical"
	default:
		panic("invalid Axis")
	}
}

// State reports the scroll state.
func (s *Scroll) State() ScrollState {
	switch {
	case s.flinger.Active():
		return StateFlinging
	case s.dragging:
		return StateDragging
	default:
		return StateIdle
	}
}

func (s ScrollState) String() string {
	switch s {
	case StateIdle:
		return "StateIdle"
	case StateDragging:
		return "StateDragging"
	case StateFlinging:
		return "StateFlinging"
	default:
		panic("unreachable")
	}
}
