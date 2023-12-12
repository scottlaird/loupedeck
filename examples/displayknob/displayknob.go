package main

// Demonstration code for using github.com/scottlaird/loupedeck in Go.
//
// This creates several buttons and does some back-end logic to use
// the Loupedeck Live as a sort of minimal smart DMX controller, for
// controlling my desktop video conferencing lights.  This is only a
// partial implementation; it's intended as an example rather than a
// full-blown DMX controller.  Specifically, this doesn't actually
// talk to any DMX hardware.  The actual controller will live in its
// own Github repo, link TBD.
//

import (
	"fmt"

	"github.com/scottlaird/loupedeck"
)

func main() {
	l, err := loupedeck.ConnectAuto()
	if err != nil {
		panic(err)
	}
	defer l.Close()

	go l.Listen()
	l.SetDisplays()

	// Create widgets
	x1 := loupedeck.NewWatchedInt(50)
	w1 := loupedeck.NewDKAnalogWidget(0, 100, x1, "AnalogTest")

	x1.AddWatcher(func(i int) {
		fmt.Printf("Knob (1) set to %d\n", i)
		w1.Draw(l)
	})

	x2 := loupedeck.NewWatchedInt(10)
	w2 := loupedeck.NewDKAnalogWidget(0, 30, x2, "Test 2")

	x2.AddWatcher(func(i int) {
		fmt.Printf("Knob (2) set to %d\n", i)
		w2.Draw(l)
	})

	x3 := loupedeck.NewWatchedInt(70)
	w3 := loupedeck.NewDKAnalogWidget(0, 100, x3, "Test 3")

	x3.AddWatcher(func(i int) {
		fmt.Printf("Knob (3) set to %d\n", i)
		w3.Draw(l)
	})

	l.WidgetHolder([]loupedeck.DKWidget{w1, w2, w3})

	select {} // Wait forever

}
