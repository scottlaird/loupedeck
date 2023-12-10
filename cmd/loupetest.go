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
	"image"
	"image/color"
	"time"
)

func main() {
	// Right now, the Loupedeck doesn't always respond if the
	// previous run didn't shut down correctly.  Unsure why yet.
	// Re-running fixes the problem.
	fmt.Printf("Trying to connect to Loupedeck.  If this fails, try hitting ctrl-C and re-running.\n")
	l, err := loupedeck.ConnectAuto()
	if err != nil {
		panic(err)
	}
	defer l.Close()

	light1 := loupedeck.NewWatchedInt(0)
	light1.AddWatcher(func(i int) { fmt.Printf("DMX 1->%d\n", i) })
	light2 := loupedeck.NewWatchedInt(0)
	light2.AddWatcher(func(i int) { fmt.Printf("DMX 3->%d\n", i) })
	light3 := loupedeck.NewWatchedInt(0)
	light3.AddWatcher(func(i int) { fmt.Printf("DMX 5->%d\n", i) })
	light4 := loupedeck.NewWatchedInt(0)
	light4.AddWatcher(func(i int) { fmt.Printf("DMX 7->%d\n", i) })
	light5 := loupedeck.NewWatchedInt(0)
	light5.AddWatcher(func(i int) { fmt.Printf("DMX 9->%d\n", i) })
	light6 := loupedeck.NewWatchedInt(0)
	light6.AddWatcher(func(i int) { fmt.Printf("DMX 11->%d\n", i) })

	_ = l.NewTouchDial(l.GetDisplay("left"), light1, light2, light3, 0, 100)
	_ = l.NewTouchDial(l.GetDisplay("right"), light4, light5, light6, 0, 10)

	go func() {
		time.Sleep(2 * time.Second)
		_ = l.SetButtonColor(loupedeck.Circle, color.RGBA{255, 0, 0, 255})      // This doesn't seem to "stick".  Not sure why.
		_ = l.SetButtonColor(loupedeck.Button1, color.RGBA{8, 8, 8, 255})       // This doesn't seem to "stick".  Not sure why.
		_ = l.SetButtonColor(loupedeck.Button2, color.RGBA{16, 16, 16, 255})    // This doesn't seem to "stick".  Not sure why.
		_ = l.SetButtonColor(loupedeck.Button3, color.RGBA{32, 32, 32, 255})    // This doesn't seem to "stick".  Not sure why.
		_ = l.SetButtonColor(loupedeck.Button4, color.RGBA{64, 64, 64, 255})    // This doesn't seem to "stick".  Not sure why.
		_ = l.SetButtonColor(loupedeck.Button5, color.RGBA{128, 128, 128, 255}) // This doesn't seem to "stick".  Not sure why.
		_ = l.SetButtonColor(loupedeck.Button6, color.RGBA{255, 255, 255, 255}) // This doesn't seem to "stick".  Not sure why.
	}()

	w1 := loupedeck.NewWatchedInt(0)
	w2 := loupedeck.NewWatchedInt(0)
	w3 := loupedeck.NewWatchedInt(0)
	w4 := loupedeck.NewWatchedInt(0)
	w5 := loupedeck.NewWatchedInt(0)
	w6 := loupedeck.NewWatchedInt(0)
	_ = AeonColorTempButton(l, w1, 2, loupedeck.Touch1)
	_ = AeonColorTempButton(l, w2, 4, loupedeck.Touch5)
	_ = AeonColorTempButton(l, w3, 6, loupedeck.Touch9)
	_ = AeonColorTempButton(l, w4, 8, loupedeck.Touch4)
	_ = AeonColorTempButton(l, w5, 10, loupedeck.Touch8)
	_ = AeonColorTempButton(l, w6, 12, loupedeck.Touch12)

	// Define the 'Circle' button (bottom left) to function as an "off" button.
	l.BindButton(loupedeck.Circle, func(b loupedeck.Button, s loupedeck.ButtonStatus) {
		l.SetButtonColor(loupedeck.Circle, color.RGBA{255, 0, 0, 255})
		light1.Set(0)
		light2.Set(0)
		light3.Set(0)
		light4.Set(0)
		light5.Set(0)
		light6.Set(0)
		w1.Set(0)
		w2.Set(0)
		w3.Set(0)
		w4.Set(0)
		w5.Set(0)
		w6.Set(0)
	})

	l.BindButton(loupedeck.Button1, func(b loupedeck.Button, s loupedeck.ButtonStatus) {
		light1.Set(15)
		light2.Set(3)
		light3.Set(5)
		light4.Set(2)
		light5.Set(3)
		light6.Set(4)
	})

	l.Listen()
}

func AeonColorTempButton(l *loupedeck.Loupedeck, w *loupedeck.WatchedInt, dmxid int, button loupedeck.TouchButton) *loupedeck.MultiButton {
	fmt.Printf("Watchedint starts as %p=%#v\n", w, *w)
	ims := make([]image.Image, len(ColorTemps))
	for i, t := range ColorTemps {
		im, err := l.TextInBox(90, 90, t.Name, color.Black, t.Color)
		if err != nil {
			panic(err)
		}
		ims[i] = im
	}

	watchfunc := func(i int) { fmt.Printf("dmx%d -> %d\n", dmxid, i) }
	w.AddWatcher(watchfunc)
	m := l.NewMultiButton(w, button, ims[0], ColorTemps[0].Value)
	for i := 1; i < len(ims); i++ {
		m.Add(ims[i], ColorTemps[i].Value)
	}
	fmt.Printf("Created new button: %#v\n", m)
	fmt.Printf("Watchedint is %#v\n", *w)

	return m
}

type ColorTemp struct {
	Name  string
	Color *color.RGBA
	Value int
}

// Using RotoLight Aeon 2s, color temp 3150-6300K.  Using 5 values,
// assuming that there's roughly a linear relationship between Value
// and degrees K.  Using 3100-6300, that's a span of 3200K.  Let's
// have 5 points, spaced out every 800K.  So 3100, 3900, 4700, 5500,
// 6300.
//
// But those don't look far enough apart on the streamdeck, so let's
// push them apart further.  Let's leave 4700K the same but add an
// extra 800K to the RGB numbers.
var ColorTemps = []ColorTemp{
	{
		// Around 3100K, but use 1500K for visualization
		Name:  "3100K",
		Color: &color.RGBA{255, 109, 0, 255},
		Value: 0,
	},
	{
		// Around 3900K, but use 3100K for visualization
		Name:  "3900K",
		Color: &color.RGBA{255, 184, 114, 255},
		Value: 64,
	},
	{
		// Around 4700K
		Name:  "4700K",
		Color: &color.RGBA{255, 223, 194, 255},
		Value: 128,
	},
	{
		// Around 5500K, but use 7900K
		Name:  "5500K",
		Color: &color.RGBA{228, 234, 255, 255},
		Value: 192,
	},
	{
		// Around 6300K, but use 9500K
		Name:  "6300K",
		Color: &color.RGBA{208, 222, 255, 255},
		Value: 255,
	},
}
