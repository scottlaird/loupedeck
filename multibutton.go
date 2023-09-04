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
	"fmt"
	"image"
)

// Type MultiButton implements a multi-image touch button for the
// Loupedeck Live that rotates between a set of images for each touch,
// changing its value (and image) for each touch.  Once the last image
// is touched, it loops back to the first image in the set.
type MultiButton struct {
	loupedeck *Loupedeck
	display   Display
	images    []image.Image
	values    []int
	value     *WatchedInt
	x, y      int
}

// Function TouchToXY turns a specific TouchButton into a set of x,y +
// Display addresses, for use with the Draw function.
func TouchToXY(b TouchButton) (int, int, Display) {
	switch b {
	case Touch1:
		return 0, 0, DisplayMain
	case Touch2:
		return 90, 0, DisplayMain
	case Touch3:
		return 180, 0, DisplayMain
	case Touch4:
		return 270, 0, DisplayMain
	case Touch5:
		return 0, 90, DisplayMain
	case Touch6:
		return 90, 90, DisplayMain
	case Touch7:
		return 180, 90, DisplayMain
	case Touch8:
		return 270, 90, DisplayMain
	case Touch9:
		return 0, 180, DisplayMain
	case Touch10:
		return 90, 180, DisplayMain
	case Touch11:
		return 180, 180, DisplayMain
	case Touch12:
		return 270, 180, DisplayMain
	default:
		return 0, 0, DisplayMain
	}
}

// Function NewMultiButton creates a new MultiButton, bound to an
// existing WatchedInt.  One image.Image and value must be provided;
// this is the first image (and default value) for the MultiButton.
// Additional images and values can be added via the Add function.
func (l *Loupedeck) NewMultiButton(watchedint *WatchedInt, b TouchButton, im image.Image, val int) *MultiButton {
	x, y, display := TouchToXY(b)

	m := &MultiButton{
		loupedeck: l,
		images:    []image.Image{im},
		values:    []int{val},
		value:     watchedint,
		x:         x,
		y:         y,
		display:   display,
	}

	watchedint.AddWatcher(func(i int) {
		m.Draw()
	})

	l.BindTouch(b, func(a TouchButton, b ButtonStatus, c uint16, d uint16) {
		m.Advance()
	})

	watchedint.Set(val)

	return m
}

// Function Add adds an additional image+value to a MultiButton.
func (m *MultiButton) Add(im image.Image, value int) {
	m.images = append(m.images, im)
	m.values = append(m.values, value)
}

// Function Draw redraws the MultiButton on the Loupedeck live.
func (m *MultiButton) Draw() {
	m.loupedeck.Draw(m.display, m.images[m.GetCur()], m.x, m.y)
}

// Function GetCur gets the current value of the MultiButton.  The
// value returned will match one of the values from either
// NewMultiButton or multibutton.Add, depending on which image is
// currently displayed.
func (m *MultiButton) GetCur() int {
	c := m.value.Get()
	for i, v := range m.values {
		if v == c {
			return i
		}
	}
	fmt.Printf("Could not find value, returning 0!")
	return 0
}

// Function Advance moves to the next value of the MultiButton,
// updating the display and underlying WatchedInt.
func (m *MultiButton) Advance() {
	c := m.GetCur() + 1
	if c >= len(m.images) {
		c = 0
	}
	m.value.Set(m.values[c])
}
