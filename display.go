package loupedeck

import (
	"encoding/binary"
	"image"
	"log/slog"
	"maze.io/x/pixel/pixelcolor"
	// "time"
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

func (l *Loupedeck) addDisplay(name string, id byte, width, height, offsetx, offsety int, bigEndian bool) {
	d := &Display{
		loupedeck: l,
		Name:      name,
		id:        id,
		width:     width,
		height:    height,
		offsetx:   offsetx,
		offsety:   offsety,
		bigEndian: bigEndian,
	}
	l.displays[name] = d
}

func (l *Loupedeck) SetDisplays() {
	switch l.Product {
	case "0003":
		slog.Info("Using Loupedeck CT v1 display settings.")
		l.addDisplay("left", 'L', 60, 270, 0, 0, false)
		l.addDisplay("main", 'A', 360, 270, 60, 0, false)
		l.addDisplay("right", 'R', 60, 270, 420, 0, false)
		l.addDisplay("dial", 'W', 240, 240, 0, 0, true)
	case "0007":
		slog.Info("Using Loupedeck CT v2 display settings.")
		l.addDisplay("left", 'M', 60, 270, 0, 0, false)
		l.addDisplay("main", 'M', 360, 270, 60, 0, false)
		l.addDisplay("right", 'M', 60, 270, 420, 0, false)
		l.addDisplay("all", 'M', 480, 270, 0, 0, false) // Same as left+main+right
		l.addDisplay("dial", 'W', 240, 240, 0, 0, true)
	case "0004":
		slog.Info("Using Loupedeck Live display settings.")
		l.addDisplay("left", 'L', 60, 270, 0, 0, false)
		l.addDisplay("main", 'A', 360, 270, 0, 0, false)
		l.addDisplay("right", 'R', 60, 270, 0, 0, false)
	case "0006", "0d06":
		slog.Info("Using Loupedeck Live S/Razor Stream Controller display settings.")
		l.addDisplay("left", 'M', 60, 270, 0, 0, false)
		l.addDisplay("main", 'M', 360, 270, 60, 0, false)
		l.addDisplay("right", 'M', 60, 270, 420, 0, false)
		l.addDisplay("all", 'M', 480, 270, 0, 0, false) // Same as left+main+right
	default:
		panic("Unknown device type: " + l.Product)
	}
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

	x := xoff + d.offsetx
	y := yoff + d.offsety
	width := im.Bounds().Dx()
	height := im.Bounds().Dy()
	slog.Info("Draw parameters", "x", x, "y", y, "width", width, "height", height)

	// Call 'WriteFramebuff'
	data := make([]byte, 10)
	binary.BigEndian.PutUint16(data[0:], uint16(d.id))
	binary.BigEndian.PutUint16(data[2:], uint16(x))
	binary.BigEndian.PutUint16(data[4:], uint16(y))
	binary.BigEndian.PutUint16(data[6:], uint16(width))
	binary.BigEndian.PutUint16(data[8:], uint16(height))

	b := im.Bounds()

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			pixel := pixelcolor.ToRGB565(im.At(x, y))
			lowByte := byte(pixel & 0xff)
			highByte := byte(pixel >> 8)

			// The Loupedeck CT's center knob screen wants
			// images fed to it big endian; all other
			// displays are little endian.
			if d.bigEndian {
				data = append(data, highByte, lowByte)
			} else {
				data = append(data, lowByte, highByte)
			}
		}
	}

	m := d.loupedeck.NewMessage(WriteFramebuff, data)
	err :=d.loupedeck.Send(m)
	if err != nil {
		slog.Warn("Send failed", "err", err)
	}

	// I'd love to watch the return code for WriteFramebuff, but
	// it doesn't seem to come back until after Draw, below.

	//resp, err := d.loupedeck.SendAndWait(m, 50*time.Millisecond)
	//if err != nil {
	//	slog.Warn("Received error on draw", "message", resp)
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
	err = d.loupedeck.Send(m2)
	if err != nil {
		slog.Warn("Send failed", "err", err)
	}
}
