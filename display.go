package loupedeck

import (
	"encoding/binary"
	"image"
	"log/slog"
	"maze.io/x/pixel/pixelcolor"
)

// Type Display is part of the Loupedeck protocol, used to identify
// which of the displays on the Loupedeck to write to.
type Display struct {
	loupedeck        *Loupedeck
	id               byte
	width, height    int
	offsetx, offsety int // used for mapping legacy left/center/right screens onto unified devices.
	Name             string
	bigEndian        bool
}

// Function GetDisplay returns a Display object with a given name if
// it exists, otherwise it returns nil.
//
// Traditional Loupedeck devices had 3 displays, Left, Center, and
// Right.  Newer devices make all 3 look like a single display, and
// it's impossible to know at compile-time what any given device will
// support, so we need to create them dynamically and then look them
// up.
//
// In addition, some devices (like the Loupedeck CT) have additional
// displays.
//
// For now, common display names are:
//
//   - left (on all devices, emulated on newer hardware)
//   - right (on all devices, emulated on newer hardware)
//   - main (on all devices, emulated on newer hardware)
//   - dial (Loupedeck CT only)
//   - main (only on newer hardware)
func (l *Loupedeck) GetDisplay(name string) *Display {
	return l.displays[name]
}

func (l *Loupedeck) addDisplay(name string, id byte, width, height, offsetx, offsety int) {
	d := &Display{
		loupedeck: l,
		Name:      name,
		id:        id,
		width:     width,
		height:    height,
		offsetx:   offsetx,
		offsety:   offsety,
		bigEndian: false,
	}
	l.displays[name] = d
}

func (l *Loupedeck) SetDisplays() {
	l.addDisplay("left", 'L', 60, 270, 0, 0)
	l.addDisplay("main", 'A', 360, 270, 0, 0)
	l.addDisplay("right", 'R', 60, 270, 0, 0)
	l.addDisplay("all", 'M', 480, 270, 0, 0)
	l.addDisplay("dial", 'W', 240, 240, 0, 0)
}

// Function Height returns the height (in pixels) of the Loupedeck's displays.
func (d *Display) Height() int {
	return d.height
}

func (d *Display) Width() int {
	return d.width
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
func (d *Display) Draw(im image.Image, xoff, yoff int) {
	slog.Info("Draw called", "Display", d.Name, "xoff", xoff, "yoff", yoff, "width", im.Bounds().Dx(), "height", im.Bounds().Dy())
	littleEndian := true

	// Call 'WriteFramebuff'
	data := make([]byte, 10)
	binary.BigEndian.PutUint16(data[0:], uint16(d.id))
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
		}
	}

	m := d.loupedeck.NewMessage(WriteFramebuff, data)
	d.loupedeck.Send(m)

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
	binary.BigEndian.PutUint16(data2[0:], uint16(d.id))
	m2 := d.loupedeck.NewMessage(Draw, data2)
	d.loupedeck.Send(m2)
}
