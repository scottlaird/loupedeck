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

// Type IntKnob is an abstraction over the Loupedeck Live's Knobs.
// The IntKnob turns left/right dial actions into incrememnting and
// decrementing an integer within a specified range.  In addition, the
// 'click' action of the knob resets the IntKnob's value to 0.
type IntKnob struct {
	knob       Knob
	watchedint *WatchedInt
	min        int
	max        int
}

// Function Get returns the current value of the IntKnob.
func (k *IntKnob) Get() int {
	return k.watchedint.Get()
}

// Function Set sets the current value of the IntKnob, triggering any
// callbacks set on the WatchedInt that underlies the IntKnob.
func (k *IntKnob) Set(v int) {
	if v < k.min {
		v = k.min
	}
	if v > k.max {
		v = k.max
	}
	k.watchedint.Set(v)
}

// Function Inc incremements (or decrements) the current value of the
// IntKnob by a specified amount.  This triggers a callback on the
// WatchedInt that underlies the IntKnob.
func (k *IntKnob) Inc(v int) {
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

// Function NewIntKnob returns a new IntKnob object, already bound to the
// specified Knob and ready to use.
//
// IntKnob implements a generic dial knob using the specified
// Loupedeck Knob.  It binds the dial function of the knob to
// increase/decrease the IntKnob's value and binds the button function
// of the knob to reset the value to 0.  Basically, spin the dial and
// it changes, and click and it resets.
func (l *Loupedeck) IntKnob(k Knob, min int, max int, watchedint *WatchedInt) *IntKnob {
	i8k := &IntKnob{
		knob:       k,
		watchedint: watchedint,
		min:        min,
		max:        max,
	}
	l.BindKnob(k, func(k Knob, v int) {
		i8k.Inc(v)
	})
	l.BindButton(Button(k), func(b Button, s ButtonStatus) {
		if s == ButtonDown {
			i8k.Set(0)
		}
	})
	return i8k
}
