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

// Package loupedeck provides a Go interface for talking to a
// Loupedeck Live control surface.
//
// The Loupedeck Live with firmware 1.x appeared as a USB network
// device that we talked to via HTTP+websockets, but newer firmware
// looks like a serial device that talks a mutant version of the
// Websocket protocol.
//
// See https://github.com/foxxyz/loupedeck for Javascript code for
// talking to the Loupedeck Live; it supports more of the Loupedeck's
// functionality.
package loupedeck

import (
	"encoding/binary"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"maze.io/x/pixel/pixelcolor"
	"net"
	"net/http"
	"sync"
	"time"
)

// Type Header is a uint16 used to identify various commands and
// actions needed for the Loupedeck protocol.
type Header uint16

const (
	Confirm          Header = 0x0302
	Tick                    = 0x0400
	SetBrightness           = 0x0409
	ConfirmFramebuff        = 0x0410
	SetVibration            = 0x041b
	ButtonPress             = 0x0500
	KnobRotate              = 0x0501
	Reset                   = 0x0506
	Draw                    = 0x050f
	SetColor                = 0x0702
	Touch                   = 0x094d
	TouchEnd                = 0x096d
	Version                 = 0x0c07
	MCU                     = 0x180d
	Serial                  = 0x1f03
	WriteFramebuff          = 0xff10
)

// Type Display is part of the Loupedeck protocol, used to identify
// which of the displays on the Loupedeck to write to.
type Display uint16

const (
	DisplayMain  Display = 'A'
	DisplayLeft          = 'L'
	DisplayRight         = 'R'
)

// Type Loupedeck describes a Loupedeck device.
type Loupedeck struct {
	font             *opentype.Font
	face             font.Face
	fontdrawer       *font.Drawer
	serial           *SerialWebSockConn
	conn             *websocket.Conn
	buttonBindings   map[Button]ButtonFunc
	buttonUpBindings map[Button]ButtonFunc
	knobBindings     map[Knob]KnobFunc
	touchBindings    map[TouchButton]TouchFunc
	touchUpBindings  map[TouchButton]TouchFunc
	transactionID    uint8
	transactionMutex sync.Mutex
}

// Function ConnectAuto connects to a Loupedeck Live by automatically
// locating the first USB Loupedeck device in the system.  If you have
// more than one device and want to connect to a specific one, then
// use ConnectPath().
func ConnectAuto() (*Loupedeck, error) {
	c, err := ConnectSerialAuto()
	if err != nil {
		return nil, err
	}

	return tryConnect(c)
}

// Function ConnectPath connects to a Loupedeck Live via a specified serial device.  If successful it returns a new Loupedeck.
func ConnectPath(serialPath string) (*Loupedeck, error) {
	c, err := ConnectSerialPath(serialPath)
	if err != nil {
		return nil, err
	}

	return tryConnect(c)
}

type connectResult struct {
	l   *Loupedeck
	err error
}

// function tryConnect helps make connections to USB devices more
// reliable by adding timeout and retry logic.
//
// Without this, 50% of the time my LoupeDeck fails to connect the
// HTTP link for the websocket.  We send the HTTP headers to request a
// websocket connection, but the LoupeDeck never returns.
//
// This is a painful workaround for that.  It uses the generic Go
// pattern for implementing a timeout (do the "real work" in a
// goroutine, feeding answers to a channel, and then add a timeout on
// select).  If the timeout triggers, then it tries a second time to
// connect.  This has a 100% success rate for me.
//
// The actual connection logic is all in doConnect(), below.
func tryConnect(c *SerialWebSockConn) (*Loupedeck, error) {
	result := make(chan connectResult, 1)
	go func() {
		r := connectResult{}
		r.l, r.err = doConnect(c)
		result <- r
	}()

	select {
	case <-time.After(2 * time.Second):
		// timeout
		slog.Info("Timeout! Trying again without timeout.")
		return doConnect(c)

	case result := <-result:
		return result.l, result.err
	}
}

func doConnect(c *SerialWebSockConn) (*Loupedeck, error) {
	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			slog.Info("Dialing...")
			return c, nil
		},
		HandshakeTimeout: 1 * time.Second,
	}

	header := http.Header{}

	slog.Info("Attempting to open websocket connection")
	conn, resp, err := dialer.Dial("ws://fake", header)

	if err != nil {
		slog.Warn("dial failed", "err", err)
		return nil, err
	}

	slog.Info("Connect successful", "resp", resp)

	l := &Loupedeck{
		conn:             conn,
		buttonBindings:   make(map[Button]ButtonFunc),
		buttonUpBindings: make(map[Button]ButtonFunc),
		knobBindings:     make(map[Knob]KnobFunc),
		touchBindings:    make(map[TouchButton]TouchFunc),
		touchUpBindings:  make(map[TouchButton]TouchFunc),
	}
	l.SetDefaultFont()

	return l, nil
}

func (l *Loupedeck) Close() {
	l.conn.Close()
	l.serial.Close()
}

// Function Height returns the height (in pixels) of the Loupedeck's displays.
func (l *Loupedeck) Height() int {
	return 270
}

// Function FontDrawer returns a font.Drawer object configured to
// writing text onto the Loupedeck's graphical buttons.
func (l *Loupedeck) FontDrawer() font.Drawer {
	return font.Drawer{
		Src:  l.fontdrawer.Src,
		Face: l.face,
	}
}

// Function Face returns the current font.Face in use for writing text
// onto the Loupedeck's graphical buttons.
func (l *Loupedeck) Face() font.Face {
	return l.face
}

// Function TextInBox writes a specified string into a x,y pixel
// image.Image, using the specified foreground and background colors.
// The font size used will be chosen to maximize the size of the text.
func (l *Loupedeck) TextInBox(x, y int, s string, fg, bg color.Color) (image.Image, error) {
	im := image.NewRGBA(image.Rect(0, 0, x, y))
	draw.Draw(im, im.Bounds(), &image.Uniform{bg}, image.ZP, draw.Src)

	fd := l.FontDrawer()
	fd.Src = &image.Uniform{fg}
	fd.Dst = im

	size := 12.0
	x26 := fixed.I(x)
	y26 := fixed.I(y)

	mx26 := fixed.I(int(float64(x) * 0.85))
	my26 := fixed.I(int(float64(y) * 0.85))

	for {
		face, err := opentype.NewFace(l.font, &opentype.FaceOptions{
			Size: size,
			DPI:  150,
		})
		if err != nil {
			return nil, err
		}

		fd.Face = face

		bounds, _ := fd.BoundString(s)
		fmt.Printf("Measured %q at %+v\n", s, bounds)
		width := bounds.Max.X - bounds.Min.X
		height := bounds.Max.Y - bounds.Min.Y

		if width > mx26 || height > my26 {
			size = size * 0.8
			fmt.Printf("Reducing font size to %f\n", size)
			continue
		}

		centerx := (x26 - width) / 2
		centery := (y26-height)/2 - bounds.Min.Y

		fmt.Printf("H: %v  H: %v  Center: %v/%v\n", height, width, centerx, centery)

		fd.Dot = fixed.Point26_6{centerx, centery}
		fd.DrawString(s)
		return im, nil
	}

}

// Function SetDefaultFont sets the default font for drawing onto buttons.
//
// TODO(laird): Actually make it easy to override this default.
func (l *Loupedeck) SetDefaultFont() error {
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return err
	}
	l.font = f

	l.face, err = opentype.NewFace(l.font, &opentype.FaceOptions{
		Size: 12,
		DPI:  150,
	})
	if err != nil {
		return err
	}

	l.fontdrawer = &font.Drawer{
		Src:  &image.Uniform{color.RGBA{255, 255, 255, 255}},
		Face: l.face,
	}
	return nil
}

// Function Listen waits for events from the Loupedeck and calls
// callbacks as configured.
func (l *Loupedeck) Listen() {
	for {
		_, message, err := l.conn.ReadMessage()

		if err != nil {
			slog.Warn("Read error", "error", err)
		}
		slog.Debug("Read", "message", message)

		header := Header(binary.BigEndian.Uint16(message[0:]))

		switch header {
		case Confirm:
		case SetBrightness:
		case SetVibration:
		case Draw:
		case WriteFramebuff:
		case ConfirmFramebuff:
		case 0x40f: // Undefined
		case 0x1c73: // Undefined
		case 0x1f73: // Undefined
			// nothing
		case ButtonPress:
			button := Button(binary.BigEndian.Uint16(message[2:]))
			upDown := ButtonStatus(message[4])
			if upDown == ButtonDown && l.buttonBindings[button] != nil {
				l.buttonBindings[button](button, upDown)
			} else if upDown == ButtonUp && l.buttonUpBindings[button] != nil {
				l.buttonUpBindings[button](button, upDown)
			} else {
				slog.Info("Received uncaught button press message", "button", button, "upDown", upDown, "message", message)
			}
		case KnobRotate:
			knob := Knob(binary.BigEndian.Uint16(message[2:]))
			value := int(message[4])
			if l.knobBindings[knob] != nil {
				v := value
				if value == 255 {
					v = -1
				}
				l.knobBindings[knob](knob, v)
			} else {
				slog.Debug("Received knob rotate message", "knob", knob, "value", value, "message", message)
			}
		case Touch:
			x := binary.BigEndian.Uint16(message[4:])
			y := binary.BigEndian.Uint16(message[6:])
			id := message[8] // Not sure what this is for
			b := touchCoordToButton(x, y)

			if l.touchBindings[b] != nil {
				l.touchBindings[b](b, ButtonDown, x, y)
			} else {
				slog.Debug("Received touch message", "x", x, "y", y, "id", id, "b", b, "message", message)
			}
		case TouchEnd:
			x := binary.BigEndian.Uint16(message[4:])
			y := binary.BigEndian.Uint16(message[6:])
			id := message[8] // Not sure what this is for
			b := touchCoordToButton(x, y)

			if l.touchUpBindings[b] != nil {
				l.touchUpBindings[b](b, ButtonUp, x, y)
			} else {
				slog.Debug("Received touch end message", "x", x, "y", y, "id", id, "b", b, "message", message)
			}
		default:
			slog.Info("Received unknown message", "header", header, "message", message)
		}
	}
}

// Function newTransactionId picks the next 8-bit transaction ID
// number.  This is used as part of the Loupedeck protocol and used to
// match results with specific queries.  The transaction ID
// incrememnts per call and rolls over back to 1 (not 0).
func (l *Loupedeck) newTransactionId() uint8 {
	l.transactionMutex.Lock()
	t := l.transactionID
	t++
	if t == 0 {
		t++
	}
	l.transactionID = t
	l.transactionMutex.Unlock()

	return t
}

// Function sendMessage sends a formatted message to the Loupedeck.
func (l *Loupedeck) sendMessage(h Header, data []byte) error {
	t := l.newTransactionId()
	b := make([]byte, 3)
	binary.BigEndian.PutUint16(b[0:], uint16(h))
	b[2] = byte(t)
	b = append(b, data...)

	/*
		if len(b) > 32 {
			log.Printf("Sendmessage (%d) %#v...\n", len(b), b[0:32])
		} else {
			log.Printf("Sendmessage (%d) %#v\n", len(b), b)
		}
	*/

	l.conn.WriteMessage(websocket.BinaryMessage, b)

	return nil
}

// Function SetBrightness sets the overall brightness of the Loupedeck display.
func (l *Loupedeck) SetBrightness(b int) {
	data := make([]byte, 1)
	data[0] = byte(b)
	l.sendMessage(SetBrightness, data)
}

// Function SetButtonColor sets the color of a specific Button.  The
// Loupedeck Live allows the 8 buttons below the display to be set to
// specific colors, however the 'Circle' button's colors may be
// overridden to show the status of the Loupedeck Live's connection to
// the host.
func (l *Loupedeck) SetButtonColor(b Button, c color.RGBA) {
	data := make([]byte, 4)
	data[0] = byte(b)
	data[1] = c.R
	data[2] = c.G
	data[3] = c.B
	l.sendMessage(SetColor, data)
}

// Function Draw draws an image onto a specific display of the
// Loupedeck Live.  The device has 3 seperate displays, the left
// segment (by knobs 1-3), the right segment (by knobs 4-6) and the
// main/center segment (underneath the 4x3 array of touch buttons).
// Drawing subsets of a display is explicitly allowed; writing a 90x90
// block of pixels to the main display will only overwrite one
// button's worth of image, and will not touch other pixels.
func (l *Loupedeck) Draw(displayid Display, im image.Image, xoff, yoff int) {
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
			data = append(data, byte(pixel&0xff))
			data = append(data, byte(pixel>>8))
		}
	}

	l.sendMessage(WriteFramebuff, data)

	// Call 'Draw'
	data2 := make([]byte, 2)
	binary.BigEndian.PutUint16(data2[0:], uint16(displayid))
	l.sendMessage(Draw, data2)
}
