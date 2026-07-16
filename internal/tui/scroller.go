package tui

import (
	"math"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
)

// scroller wraps a viewport with spring-eased vertical scrolling. The viewport
// still owns the content and clamps offsets; only the *displayed* offset is
// animated toward the target the viewport computed. When reduce is set (tests /
// reduced motion) scrolling is instant and no frames are scheduled.
type scroller struct {
	vp        viewport.Model
	spring    harmonica.Spring
	pos, vel  float64
	target    float64
	animating bool
	reduce    bool
}

func newScroller(reduce bool) scroller {
	return scroller{
		vp:     viewport.New(0, 0),
		spring: harmonica.NewSpring(harmonica.FPS(60), 8.0, 1.0),
		reduce: reduce,
	}
}

func (s *scroller) setSize(w, h int)    { s.vp.Width, s.vp.Height = w, h }
func (s *scroller) setContent(c string) { s.vp.SetContent(c) }
func (s *scroller) view() string        { return s.vp.View() }

func frameTick() tea.Cmd {
	return tea.Tick(time.Second/60, func(time.Time) tea.Msg { return frameMsg{} })
}

// onKey applies a scroll key. It lets the viewport compute the clamped target
// offset, then springs the displayed offset toward it (or jumps, if reduced).
func (s *scroller) onKey(msg tea.KeyMsg) tea.Cmd {
	if s.reduce {
		s.vp, _ = s.vp.Update(msg)
		s.pos, s.target = float64(s.vp.YOffset), float64(s.vp.YOffset)
		return nil
	}
	before := s.vp.YOffset
	s.vp, _ = s.vp.Update(msg)
	s.target = float64(s.vp.YOffset)
	s.vp.SetYOffset(before) // hold the display; the spring will move it
	s.pos = float64(before)
	if s.target == s.pos || s.animating {
		return nil
	}
	s.animating = true
	return frameTick()
}

// scrollBy scrolls by delta lines through the same spring as onKey (used by
// the mouse wheel; a synthetic KeyMsg would move one line per event and couple
// wheel semantics to the viewport keymap).
func (s *scroller) scrollBy(delta int) tea.Cmd {
	if s.reduce {
		s.vp.SetYOffset(s.vp.YOffset + delta) // SetYOffset clamps
		s.pos, s.target = float64(s.vp.YOffset), float64(s.vp.YOffset)
		return nil
	}
	before := s.vp.YOffset
	s.vp.SetYOffset(before + delta) // let the viewport clamp the target
	s.target = float64(s.vp.YOffset)
	s.vp.SetYOffset(before) // hold the display; the spring will move it
	s.pos = float64(before)
	if s.target == s.pos || s.animating {
		return nil
	}
	s.animating = true
	return frameTick()
}

// onFrame advances the spring one step and returns the next tick until settled.
func (s *scroller) onFrame() tea.Cmd {
	s.pos, s.vel = s.spring.Update(s.pos, s.vel, s.target)
	if math.Abs(s.pos-s.target) < 0.5 && math.Abs(s.vel) < 0.5 {
		s.pos, s.vel, s.animating = s.target, 0, false
		s.vp.SetYOffset(int(math.Round(s.target)))
		return nil
	}
	s.vp.SetYOffset(int(math.Round(s.pos)))
	return frameTick()
}
