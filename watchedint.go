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

import ()

// Type WatchFunc is used for callbacks for changes to a WatchedInt.
type WatchFunc func(int)

// Type WatchedInt wraps an int with zero or more callback watchers;
// whenever the value of the int changes (via Set), all of the
// callbacks will be called.  This is used to implement a sane model
// for Loupedeck Live knobs, etc.  Calling 'myknob.Set(3)' will update
// any impacted displays and should trigger any required underlying
// behaviour.
type WatchedInt struct {
	value     int
	notifiers []WatchFunc
}

// Function NewWatchedInt creates a new WatchedInt with the specified initial value.
func NewWatchedInt(value int) *WatchedInt {
	return &WatchedInt{
		value:     value,
		notifiers: make([]WatchFunc, 0),
	}
}

// Function Get returns the current value of the WatchedInt.
func (w *WatchedInt) Get() int {
	return w.value
}

// Function Set updates the current value of the WatchedInt and calls all callback functions added via AddWatcher.
func (w *WatchedInt) Set(value int) {
	w.value = value
	for _, f := range w.notifiers {
		f(value)
	}
}

// Function AddWatcher adds a callback function for this WatchedInt.  The callback will be called whenever Set is called.
func (w *WatchedInt) AddWatcher(f WatchFunc) {
	w.notifiers = append(w.notifiers, f)
}
