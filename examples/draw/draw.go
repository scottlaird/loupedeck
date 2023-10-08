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
	"github.com/scottlaird/loupedeck"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"time"
)

func main() {
	l, err := loupedeck.ConnectAuto()
	if err != nil {
		panic(err)
	}
	defer l.Close()

	go l.Listen()
	l.SetDisplays()

	d := l.GetDisplay("dial")

	time.Sleep(1 * time.Second)

	slog.Info("Drawing 1x1 white rect at 10,10")

	im := image.NewRGBA(image.Rect(0, 0, 1, 1))
	draw.Draw(im, im.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	d.Draw(im, 10, 10)

	time.Sleep(1 * time.Second)

	slog.Info("Drawing 5x5 blue rect at 20,20")
	im = image.NewRGBA(image.Rect(0, 0, 5, 5))
	draw.Draw(im, im.Bounds(), &image.Uniform{&color.RGBA{0, 0, 255, 255}}, image.ZP, draw.Src)
	d.Draw(im, 20, 20)

	time.Sleep(1 * time.Second)

	slog.Info("Drawing 10x10 red rect at 30,30")
	im = image.NewRGBA(image.Rect(0, 0, 10, 10))
	draw.Draw(im, im.Bounds(), &image.Uniform{&color.RGBA{255, 0, 0, 255}}, image.ZP, draw.Src)
	d.Draw(im, 30, 30)

	time.Sleep(1 * time.Second)

	slog.Info("Drawing 20x20 green rect at 50,50")
	im = image.NewRGBA(image.Rect(0, 0, 20, 20))
	draw.Draw(im, im.Bounds(), &image.Uniform{&color.RGBA{0, 0, 255, 255}}, image.ZP, draw.Src)
	d.Draw(im, 50, 50)

	time.Sleep(1 * time.Second)

	slog.Info("Drawing 20x20 rect at 70,70")
	im = image.NewRGBA(image.Rect(0, 0, 20, 20))
	draw.Draw(im, im.Bounds(), &image.Uniform{&color.RGBA{255, 0, 255, 255}}, image.ZP, draw.Src)
	d.Draw(im, 70, 70)

	time.Sleep(1 * time.Second)
}
