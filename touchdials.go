/*
   Copyright 2021 Google LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       https://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package loupedeck

import (
	"github.com/golang/freetype"
	"image"
	"image/color"
	"image/draw"
	"strconv"
)

// Type TouchDial implements a "smart" bank of dials for the Loupedeck
// Live.  If displayid is DisplayLeft then the TouchDial will display
// knobs 1-3 on the left display, otherwise it will show knobs 4-6 on
// the right display.
//
// The display will show the current value of the WatchedInt for each
// knob.  Turning the knob will increment/decrement each value as
// expected.  Clicking the knob will zero the value.  Sliding up or
// down on the LCD display will increase or decrease all 3 knob values
// at once.
type TouchDial struct {
	loupedeck              *Loupedeck
	display                Display
	w1, w2, w3             *WatchedInt
	dragv1, dragv2, dragv3 int
	Knob1, Knob2, Knob3    *IntKnob
	touchdivisor           int
	dragstart              uint16
}

func (l *Loupedeck) NewTouchDial(displayid Display, w1, w2, w3 *WatchedInt, min, max int) *TouchDial {
	touch := TouchLeft
	var knob1, knob2, knob3 Knob
	knob1 = Knob1
	knob2 = Knob2
	knob3 = Knob3

	if displayid == DisplayRight {
		touch = TouchRight
		knob1 = Knob4
		knob2 = Knob5
		knob3 = Knob6
	}

	touchdial := &TouchDial{
		loupedeck: l,
		display:   displayid,
		w1:        w1,
		w2:        w2,
		w3:        w3,
	}

	touchdial.touchdivisor = int(float64(l.Height()) / float64(max-min))

	touchdial.Knob1 = l.IntKnob(knob1, min, max, w1)
	touchdial.Knob2 = l.IntKnob(knob2, min, max, w2)
	touchdial.Knob3 = l.IntKnob(knob3, min, max, w3)

	touchdial.dragstart = 65535

	l.BindTouch(touch, func(t TouchButton, s ButtonStatus, x, y uint16) {
		if touchdial.dragstart == 65535 {
			touchdial.dragv1 = w1.Get()
			touchdial.dragv2 = w2.Get()
			touchdial.dragv3 = w3.Get()
			touchdial.dragstart = y
		} else {
			delta := (int(touchdial.dragstart) - int(y)) / touchdial.touchdivisor
			touchdial.Knob1.Set(int(touchdial.dragv1) + delta)
			touchdial.Knob2.Set(int(touchdial.dragv2) + delta)
			touchdial.Knob3.Set(int(touchdial.dragv3) + delta)
		}
	})
	l.BindTouchUp(touch, func(t TouchButton, s ButtonStatus, x, y uint16) {
		touchdial.dragstart = 65535
	})

	touchdial.Draw()
	touchdial.w1.AddWatcher(func(i int) { touchdial.Draw() })
	touchdial.w2.AddWatcher(func(i int) { touchdial.Draw() })
	touchdial.w3.AddWatcher(func(i int) { touchdial.Draw() })
	return touchdial
}

// Function Draw updates the display for a TouchDial.
func (t *TouchDial) Draw() {
	im := image.NewRGBA(image.Rect(0, 0, 60, 270))
	bg := color.RGBA{0, 0, 0, 255}
	draw.Draw(im, im.Bounds(), &image.Uniform{bg}, image.ZP, draw.Src)

	fd := t.loupedeck.FontDrawer()
	fd.Dst = im

	fd.Dot = freetype.Pt(10, 30)
	fd.DrawString(strconv.Itoa(t.w1.Get()))

	fd.Dot = freetype.Pt(10, 150)
	fd.DrawString(strconv.Itoa(t.w2.Get()))

	fd.Dot = freetype.Pt(10, 260)
	fd.DrawString(strconv.Itoa(t.w3.Get()))

	t.loupedeck.Draw(t.display, im, 0, 0)
}
