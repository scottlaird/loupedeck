package loupedeck

import (
	"encoding/binary"
	"image"
	"log/slog"
	"maze.io/x/pixel/pixelcolor"
)

// Type Display is part of the Loupedeck protocol, used to identify
// which of the displays on the Loupedeck to write to.
type Display uint16

// These vary on different Loupedecks; newer models (and even some
// older models with new firmware) combine the Main/Left/Right screens
// into a single logical display called 'M'.  So, to have a stable API
// against varying hardware, we're going to need to add an extra
// mapping layer here that dynamically switches between configs based
// on which device is detected, and probably includes offsets so that
// we can put "virtual" left/right screens on top of single-screened
// devices.
const (
	DisplayMain   Display = 'A'
	DisplayLeft           = 'L'
	DisplayRight          = 'R'
	DisplayCTDial         = 'W'
	DisplaySingle         = 'M' // Razor and newer Loupedeck devices only
)

// Function Height returns the height (in pixels) of the Loupedeck's displays.
func (l *Loupedeck) Height() int {
	return 270
}

// Function Draw draws an image onto a specific display of the
// Loupedeck Live.  The device has 3 seperate displays, the left
// segment (by knobs 1-3), the right segment (by knobs 4-6) and the
// main/center segment (underneath the 4x3 array of touch buttons).
// Drawing subsets of a display is explicitly allowed; writing a 90x90
// block of pixels to the main display will only overwrite one
// button's worth of image, and will not touch other pixels.
//
// Most Loupedeck screens are little-endian, except for the knob
// screen on the Loupedeck CT, which is big-endian.  This does not
// deal with this case correctly yet.
func (l *Loupedeck) Draw(displayid Display, im image.Image, xoff, yoff int) {
	slog.Info("Draw called", "Display", string(displayid), "xoff", xoff, "yoff", yoff, "width", im.Bounds().Dx(), "height", im.Bounds().Dy())
	littleEndian := true

	// Call 'WriteFramebuff'
	data := make([]byte, 10)
	binary.BigEndian.PutUint16(data[0:], uint16(displayid))
	binary.BigEndian.PutUint16(data[2:], uint16(xoff))
	binary.BigEndian.PutUint16(data[4:], uint16(yoff))
	binary.BigEndian.PutUint16(data[6:], uint16(im.Bounds().Dx()))
	binary.BigEndian.PutUint16(data[8:], uint16(im.Bounds().Dy()))

	b := im.Bounds()

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			pixel := pixelcolor.ToRGB565(im.At(x, y))
			lowByte := byte(pixel & 0xff)
			highByte := byte(pixel >> 8)

			if littleEndian {
				data = append(data, lowByte, highByte)
			} else {
				data = append(data, highByte, lowByte)
			}
			//			data = append(data, byte(pixel&0xff))
			//			data = append(data, byte(pixel>>8))
		}
	}

	m := l.NewMessage(WriteFramebuff, data)
	l.Send(m)

	//resp, err := l.sendAndWait(m, 1*time.Second)
	//if err != nil {
	//slog.Warn("Received error on draw", "message", resp)
	//}

	// Call 'Draw'.  The screen isn't actually updated until
	// 'draw' arrives.  Unclear if we should wait for the previous
	// Framebuffer transaction to complete first, but adding a
	// giant sleep here doesn't seem to change anything.
	//
	// Ideally, we'd batch these and only call Draw when we're
	// doing with multiple FB updates.
	data2 := make([]byte, 2)
	binary.BigEndian.PutUint16(data2[0:], uint16(displayid))
	m2 := l.NewMessage(Draw, data2)
	l.Send(m2)
}
