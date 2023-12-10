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

// Button represents a physical button on the Loupedeck Live.  This
// includes the 8 buttons at the bottom of the device as well as the
// 'click' function of the 6 dials.
type Button uint16

const (
	// KnobPress1 is sent when the first knob (upper left) is clicked.
	KnobPress1 Button = 1
	// KnobPress2 is sent when the second knob (middle left) is clicked.
	KnobPress2        = 2
	// KnobPress3 is sent when the third knob (bottom left) is clicked.
	KnobPress3        = 3
	// KnobPress4 is sent when the fourth knob (upper right) is clicked.
	KnobPress4        = 4
	// KnobPress5 is sent when the fifth knob (middle right) is clicked.
	KnobPress5        = 5
	// KnobPress6 is sent when the sixth knob (middle right) is clicked.
	KnobPress6        = 6
	// Circle is sent when the left-most hardware button under the
	// display is clicked.  This has a circle icon on the
	// Loupedeck Live, but is unfortunately labeled "1" on the
	// Loupedeck CT.
	Circle            = 7
	// Button1 is sent when the button to the right of the circle
	// button is clicked.  This is labeled "1" on the Loupdeck
	// live and "2" on the Loupedeck CT.
	Button1           = 8
	// Button2 is sent when the third button is clicked.
	Button2           = 9
	// Button3 is sent when the fourth button is clicked.
	Button3           = 10
	// Button4 is sent when the fifth button is clicked.
	Button4           = 11
	// Button5 is sent when the sixth button is clicked.
	Button5           = 12
	// Button6 is sent when the seventh button is clicked.
	Button6           = 13
	// Button7 is sent when the eighth (rightmost) button is clicked.
	Button7           = 14
)

// ButtonStatus represents the state of Buttons.
type ButtonStatus uint8

const (
	// ButtonDown indicates that a button has just been pressed.
	ButtonDown ButtonStatus = 0
	// ButtonUp indicates that a button was just released.
	ButtonUp                = 1
)

// ButtonFunc is a function signature used for callbacks on Button
// events.  When a specified event happens, the ButtonFunc is called
// with parameters specifying which button was pushed and what its
// current state is.
type ButtonFunc func(Button, ButtonStatus)

// Knob represents the 6 knobs on the Loupedeck Live.
type Knob uint16

const (
	// Knob1 is the upper left knob.
	Knob1 Knob = 1
	// Knob2 is the middle left knob.
	Knob2      = 2
	// Knob3 is the bottom left knob.
	Knob3      = 3
	// Knob4 is the upper right knob.
	Knob4      = 4
	// Knob5 is the middle right knob.
	Knob5      = 5
	// Knob6 is the bottom right knob.
	Knob6      = 6
)

// KnobFunc is a function signature used for callbacks on Knob events,
// similar to ButtonFunc's use with Button events.  The exact use of
// the second parameter depends on the use; in some cases it's simply
// +1/-1 (for right/left button turns) and in other cases it's the
// current value of the dial.
type KnobFunc func(Knob, int)

// TouchButton represents the regions of the touchpad on the Loupedeck Live.
type TouchButton uint16

const (
	// TouchLeft indicates that the left touchscreen area, near the leftmost knobs has been touched.
	TouchLeft  TouchButton = 1
	// TouchRight indicates that hte right touchscreen area, near the rightmost knobs has been touched.
	TouchRight             = 2
	Touch1                 = 3
	Touch2                 = 4
	Touch3                 = 5
	Touch4                 = 6
	Touch5                 = 7
	Touch6                 = 8
	Touch7                 = 9
	Touch8                 = 10
	Touch9                 = 11
	Touch10                = 12
	Touch11                = 13
	Touch12                = 14
)

// TouchFunc is a function signature used for callbacks on TouchButton
// events, similar to ButtonFunc and KnobFunc.  The parameters are:
//
//   - The TouchButton touched
//   - The ButtonStatus (down/up)
//   - The X location touched (relative to the whole display)
//   - The Y location touched (relative to the whole display)
type TouchFunc func(TouchButton, ButtonStatus, uint16, uint16)

// touchCoordToButton translates an x,y coordinate on the
// touchscreen to a TouchButton.
func touchCoordToButton(x, y uint16) TouchButton {
	switch {
	case x < 60:
		return TouchLeft
	case x >= 420:
		return TouchRight
	}

	x -= 60
	x /= 90
	y /= 90

	return TouchButton(uint16(Touch1) + x + 4*y)
}

// BindButton sets a callback for actions on a specific
// button.  When the Button is pushed down, then the provided
// ButtonFunc is called.
func (l *Loupedeck) BindButton(b Button, f ButtonFunc) {
	l.buttonBindings[b] = f
}

// BindButtonUp sets a callback for actions on a specific
// button.  When the Button is released, then the provided
// ButtonFunc is called.
func (l *Loupedeck) BindButtonUp(b Button, f ButtonFunc) {
	l.buttonUpBindings[b] = f
}

// BindKnob sets a callback for actions on a specific
// knob.  When the Knob is turned then the provided
// KnobFunc is called.
func (l *Loupedeck) BindKnob(k Knob, f KnobFunc) {
	l.knobBindings[k] = f
}

// BindTouch sets a callback for actions on a specific
// TouchButton.  When the TouchButton is pushed down, then the
// provided TouchFunc is called.
func (l *Loupedeck) BindTouch(b TouchButton, f TouchFunc) {
	l.touchBindings[b] = f
}

// BindTouchUp sets a callback for actions on a specific
// TouchButton.  When the TouchButton is released, then the
// provided TouchFunc is called.
func (l *Loupedeck) BindTouchUp(b TouchButton, f TouchFunc) {
	l.touchUpBindings[b] = f
}
