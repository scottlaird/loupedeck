package loupedeck

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/jphsd/graphics2d"
)

var (
	colorActive     = color.RGBA{192, 192, 192, 255}
	colorInActive   = color.RGBA{64, 64, 64, 255}
	colorBackground = color.RGBA{0, 0, 0, 255}
)

// DisplayKnob is an abstraction over the Loupedeck CT's large knob
// with a display.  This is basically the IntKnob code, but with the
// Knob parameter removed, since there's only one DisplayKnob today,
// and it uses different messages, so shoehorning it into the IntKnob
// code would be messy.
type DisplayKnob struct {
	watchedint *WatchedInt
	min        int
	max        int
}

// Get returns the current value of the DisplayKnob.
func (k *DisplayKnob) Get() int {
	return k.watchedint.Get()
}

// Set sets the current value of the DisplayKnob, triggering any
// callbacks set on the WatchedInt that underlies the DisplayKnob.
func (k *DisplayKnob) Set(v int) {
	if v < k.min {
		v = k.min
	}
	if v > k.max {
		v = k.max
	}
	k.watchedint.Set(v)
}

// Inc incremements (or decrements) the current value of the
// IntKnob by a specified amount.  This triggers a callback on the
// WatchedInt that underlies the DisplayKnob.
func (k *DisplayKnob) Inc(v int) {
	x := k.watchedint.Get()
	x += v
	if x < k.min {
		x = k.min
	}
	if x > k.max {
		x = k.max
	}
	k.watchedint.Set(x)
}

// DisplayKnob implements a generic dial knob for the big knob in the
// Loupedeck CT (the one with a display in the middle, hence the name
// "DisplayKnob"). It binds the dial function of the knob to
// increase/decrease the DisplayKnob's value.  This is very similar to
// the IntKnob, except it (a) doesn't support click-to-reset (the big
// knob doesn't click) and (b) it uses different messages under the
// hood when talking to the Loupedeck.
func (l *Loupedeck) DisplayKnob(min int, max int, watchedint *WatchedInt) *DisplayKnob {
	k := &DisplayKnob{
		watchedint: watchedint,
		min:        min,
		max:        max,
	}
	l.BindKnob(CTKnob, func(_ Knob, v int) {
		k.Inc(v)
	})
	return k
}

// DragDisplayKnobFunc is a callback for handling drag events from the
// touchscreen in the middle of the Loupedeck CT's big knob.
type DragDisplayKnobFunc func(event DragEvent, x, y int)

// Quick hack to decide if a touch is a click or a drag.  In an ideal
// world, we'd also support double-click, but that either requires
// knowing the future or delaying click messages until after a
// specified time has passed without a second click, and there's no
// room in the code for either today.
func isClick(duration time.Duration, x, y int) bool {
	if duration > 500*time.Millisecond {
		return false
	}

	if x > 20 || x < -20 {
		return false
	}

	if y > 20 || y < -20 {
		return false
	}

	return true
}

// RegisterDragDisplayKnobWatcher registers a callback function for managing
// click and drag events for the touchscreen in the middle of the
// Loupedeck CT's big dial.  Only one watcher can be registered at a
// time; if it is called a second time then the previous function will
// be silently replaced.
//
// The function provided will be called with a DragEvent and a set of
// X and Y values.  If DragEvent is loupedeck.DragClick, then a click
// occured, and the X and Y parameters are the location where the
// click started.  If the DragEvent is loupedeck.DragDone, then the X
// and Y values are the *delta* that was dragged; the upper left is
// negative and the lower right is positive.  This is intended to be
// used to decide between a left and a right swipe, and that's about
// it for now.
func (l *Loupedeck) RegisterDragDisplayKnobWatcher(f DragDisplayKnobFunc) {
	l.dragDKBinding = f
	l.BindTouchCT(func(b ButtonStatus, x, y uint16) {
		if !l.dragDKStarted {
			// Not dragging yet
			if b == ButtonDown {
				// Starting dragging
				l.dragDKStarted = true
				l.dragDKStartX = x
				l.dragDKStartY = y
				l.dragDKStartTime = time.Now()
			} else if b == ButtonUp {
				// Where did *that* come from?
				slog.Warn("Received CT ButtonUp event while not dragging")
			} else {
				slog.Warn("Received unknown CT button event while not dragging", "event", b)
			}
		} else {
			// Already started dragging
			if b == ButtonDown {
				// Already dragging, can probably just ignore
			} else if b == ButtonUp {
				// Drag completed, let's see what happened...
				duration := time.Since(l.dragDKStartTime)
				dx := int(x) - int(l.dragDKStartX)
				dy := int(y) - int(l.dragDKStartY)
				fmt.Printf("Drag event: dX: %d dY: %d  elapsed: %v\n", dx, dy, duration)
				l.dragDKStarted = false

				if isClick(duration, dx, dy) {
					l.dragDKBinding(DragClick, int(x), int(y)) // use x/y, not dx/dy
				} else {
					l.dragDKBinding(DragDone, dx, dy) // Show the distance moved, not the location.
				}

			} else {
				slog.Warn("Received unknown CT button event while dragging", "event", b)
			}
		}
	})
}

// So, what I *really* want here is a set of widgets that I can
// display on the knob display, each of which control a different
// thing, possibly with slightly different UIs.  For instance, I could
// have a set of 4 widget "tabs":
//
// - Camera gain control (analog, 0%-200%)
// - Camera white balance (analog, 3000K-9000K)
// - Background video selector (discrete selections, spin to select)
// - Motorized curtain control (up/down, spin to raise/lower)
//
// When the app starts, it shows the camera gain widget.  Swiping
// right takes you to the white balance widget, then the background
// widget, then the curtain widget.  Presumably we'd show a set of 4
// grey/white blips at the bottom, and swiping would move the blip (as
// well as redrawing the display, possibly with a transition effect).
//
// Then, for any specific widget, the dial behavior (and possibly
// up/down swipes) would control something widget-specific.  Examples:
//
// - Analog base widget. Spinning the dial changes numeric value
//   between `min` and `max`.  The widget draws a partial ring around
//   the outside of the widget to show the current setting, and the
//   widget can draw numeric values as needed.
//
// - A discrete number widget.  Think camera iris control.  Works just
//   like the analog widget, but only specific values are allowed.
//   For iris, think "f/2.0", "f/2.2", "f/2.5", "f/2.8", etc.
//
// - A boolean widget.  Similar to the discrete widget, but only
//   true/false (allow names to be specified--on/off, true/false, etc).
//
// - Fully custom widgets.  Imagine a background selector, which let
//   you choose between various background images or videos.  *Might*
//   be overlap here with the discrete widget.  Requirement: be able
//   to add/remove options on the fly.  Applies to the iris widget as
//   well; changing lenses changes the allowed set of iris settings.
//
// So, the possibly-odd thing here is the dial binding, as right now,
// it's only bound once globally.  When changing widgets, do we unbind
// the previous one and just set up it from scratch again?  I think
// that'd actually work fine, by accident.
//
// To make room for widget-holder graphics, let's limit widgets to
// 240x220, saving the bottom 20px for the holder.

// DKWidget is an interface that describes a generic widget for use
// with the Loupedeck CT's knob.
type DKWidget interface {
	Activate(*Loupedeck)
	Deactivate(*Loupedeck)

	// Do we need Draw() or similar?  Shouldn't need (or want) Get/Set
}

// DKAnalogWidget is a widget for use with the Loupedeck CT's larget
// display knob.  It controls a single analog variable; turning the
// knob one direction decreases the value, turning it the other way
// increases the value.
type DKAnalogWidget struct {
	Min, Max                 int
	MinDegrees, TotalDegrees float64 // 0 is straight up
	Value                    *WatchedInt
	Name                     string
	active                   bool
}

// NewDKAnalogWidget creates a new DKAnalogWidget
func NewDKAnalogWidget(min, max int, value *WatchedInt, name string) *DKAnalogWidget {
	w := &DKAnalogWidget{
		Min:          min,
		Max:          max,
		Value:        value,
		Name:         name,
		MinDegrees:   135,
		TotalDegrees: 270,
	}

	return w
}

// Activate is called when the widget gains focus.  It needs to re-set
// the CT knob so that it updates this widget's controls.
func (w *DKAnalogWidget) Activate(l *Loupedeck) {
	w.active = true
	_ = l.DisplayKnob(w.Min, w.Max, w.Value)
	w.Draw(l)
}

// Deactivate is called when the widget loses focus.
func (w *DKAnalogWidget) Deactivate(l *Loupedeck) {
	w.active = false
}

func d2r(d float64) float64 {
	return d * math.Pi / 180
}

// Draw draws the widget on the display if the widget is currently active.
//
// Note that this is kind of expensive as it sends a lot of bits to
// the Loupedeck, and it's possible to lag several seconds behind when
// the user spins the dial quickly.  We'll probably want to break this
// out into its own thread, and only allow ~1 draw at a time to be
// queued.  Then, when the user spins the dial, we do *one* draw, then
// fetch new events, which try to queue up a zillion new draw events,
// but we only allow one to be queued at a time, so we end up doing 1
// draw per event block.  I think.
//
// Since none of the code is really thread-safe yet, this will take a
// bit more work.  At a minimum, we'll need to add locks around USB
// communication.
//
// The alternative would be to do some sort of "future draw" thing,
// where draws are scheduled for (say) 100ms in the future, and then
// we drop duplicates.  Then we inject the "draw event" back into the
// main loop, so we don't need to worry about locking.  Might be
// easier, and likely it'd have better performance.
func (w *DKAnalogWidget) Draw(l *Loupedeck) {
	// Only draw if we have the focus
	if !w.active {
		return
	}

	display := l.GetDisplay("dial")

	fmt.Printf("Should draw widget %q here.\n", w.Name)

	// TODO: actually draw something.

	im := image.NewRGBA(image.Rect(0, 0, 240, 210))
	bg := colorBackground
	draw.Draw(im, im.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	startRadian := d2r(w.MinDegrees + 180)
	startX := math.Cos(startRadian)*110 + 120
	startY := math.Sin(startRadian)*110 + 120

	radians := d2r(w.TotalDegrees)
	stopRadian := radians

	fmt.Printf("X: %f, Y: %f, startRadian: %f, radians: %f, stopRadian: %f\n", startX, startY, startRadian, radians, stopRadian)

	pen := graphics2d.NewPen(colorInActive, 1)
	// I'm officially mystified by the graphics2d coordinate
	// system, but this seems to draw correct arcs.  I started
	// with angle=0 being to the east and (x, y) coordinates, but
	// that didn't come even close.  This seems draws correctly;
	// I'll look at it again another day.
	graphics2d.DrawArc(im, []float64{startY, startX}, []float64{120, 120}, stopRadian, pen)

	stopRadian = radians * (float64(w.Value.Get()) / float64(w.Max))
	pen = graphics2d.NewPen(colorActive, 4)
	graphics2d.DrawArc(im, []float64{startY, startX}, []float64{120, 120}, stopRadian, pen)

	fd := l.FontDrawer()
	fd.Dst = im

	drawCenteredStringAt(fd, w.Name, 120, 80)
	drawCenteredStringAt(fd, strconv.Itoa(w.Value.Get()), 120, 160)

	display.Draw(im, 0, 0)
}

// WidgetHolder is a container that can hold multiple DKWidgets and
// allows the user to select between them by swiping right/left on the
// CT's display.
func (l *Loupedeck) WidgetHolder(widgets []DKWidget) {
	active := 0
	count := len(widgets)
	widgets[0].Activate(l)

	l.RegisterDragDisplayKnobWatcher(func(b DragEvent, x, y int) {
		if b == DragClick {
			fmt.Printf("Click at %d, %d\n", x, y)
		} else if b == DragDone {
			fmt.Printf("Drag, direction is %d, %d\n", x, y)

			if x < -20 {
				widgets[active].Deactivate(l)
				active++
				if active >= count {
					active = 0
				}
				widgets[active].Activate(l)
				l.drawWidgetHolderNavBar(active, count)
			} else if x > 20 {
				widgets[active].Deactivate(l)
				active--
				if active < 0 {
					active = count - 1
				}
				widgets[active].Activate(l)
				l.drawWidgetHolderNavBar(active, count)
			}
		}
	})
	l.drawWidgetHolderNavBar(0, count)
}

// drawWidgetHolderNavBar draws in the bottom 20x240 of the knob to
// show which swipable tab the user is currently on and give context.
// Since the knob is circular, most of those pixels aren't actually
// visible, but it's probably still enough room for a few dots.  We
// might need to change that to 30x or 40x if we need more room.
func (l *Loupedeck) drawWidgetHolderNavBar(position int, tabCount int) {
	// Let's just draw "blips" per tab at the bottom of the screen
	// for each tab, and highlight the current tab's blip in a
	// brighter color.  That's fine for up to ~10 tabs, after that
	// the blips start to get pretty small and we'll probably want
	// to scroll or something.  OTOH, that's a lot of tabs, so
	// maybe we don't care.
	//
	//
	// A bit of math; we have a circular display with r=120px.  If
	// we lop off S pixels at the bottom, then the circular area
	// is 2*sqrt(2*s*r-s^s) pixels wide.  Calculating the angle is
	// left as an exercise for the reader, but when S=30 we're
	// looking at about 70 degrees total.

	if tabCount > 10 {
		panic("We can't draw more than 10 tabs right now.  So either create fewer tabs or fix the logic in displayknob.go")
	}

	// We have about 70 degrees available, but we don't want to
	// spread out *too* much.  Plus, at the edges will end up
	// cropping the dots on the top edge of the 30-pixel boundry.
	// So let's use 60 degrees total, and then cap the spread to
	// 20 degrees per tab.
	degreesPerTab := 60 / float64(tabCount)
	if degreesPerTab > 20 {
		degreesPerTab = 20
	}
	anglePerTab := d2r(degreesPerTab)
	totalAngle := anglePerTab * float64(tabCount-1)
	leftMostAngle := -(totalAngle / 2)

	display := l.GetDisplay("dial")

	im := image.NewRGBA(image.Rect(0, 0, 240, 30))
	bg := colorBackground
	draw.Draw(im, im.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	penActive := graphics2d.NewPen(colorActive, 10)
	penInActive := graphics2d.NewPen(colorInActive, 8)
	var pen *graphics2d.Pen

	for i := 0; i < tabCount; i++ {
		angle := leftMostAngle + anglePerTab*float64(i)
		// I'm making 0 degrees straight down here, so sin/cos
		// go to the wrong vars.  Sue me.
		x := math.Sin(angle)*110 + 120
		y := math.Cos(angle)*110 - 90 // We're only drawing into the bottom 30px of the image, so this is +120-210 = -90.

		if i == position {
			pen = penActive
		} else {
			pen = penInActive
		}

		graphics2d.DrawPoint(im, []float64{x, y}, pen)
	}
	display.Draw(im, 0, 210)
}
