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

package loupedeck_test

import (
	"fmt"
	"github.com/scottlaird/loupedeck"
)

func Example() {
	l, err := loupedeck.Connect("ws://100.127.5.1")
	if err != nil {
		panic(err)
	}

	light1 := loupedeck.NewWatchedInt(0)
	light1.AddWatcher(func(i int) { fmt.Printf("DMX 1->%d\n", i) })
	light2 := loupedeck.NewWatchedInt(0)
	light2.AddWatcher(func(i int) { fmt.Printf("DMX 3->%d\n", i) })
	light3 := loupedeck.NewWatchedInt(0)
	light3.AddWatcher(func(i int) { fmt.Printf("DMX 5->%d\n", i) })

	l.NewTouchDial(loupedeck.DisplayLeft, light1, light2, light3, 0, 100)

	// Define the 'Circle' button (bottom left) to function as an "off" button.
	l.BindButton(loupedeck.Circle, func(b loupedeck.Button, s loupedeck.ButtonStatus) {
		light1.Set(0)
		light2.Set(0)
		light3.Set(0)
	})

	l.Listen()
}
