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

// Type Display is part of the Loupedeck protocol, used to identify
// which of the displays on the Loupedeck to write to.
type Display uint16

const (
	DisplayMain   Display = 'A'
	DisplayLeft           = 'L'
	DisplayRight          = 'R'
	DisplayCTDial         = 'W'
)

type transactionCallback func(m *Message)

// Type Loupedeck describes a Loupedeck device.
type Loupedeck struct {
	Vendor               string
	Product              string
	Model                string
	Version              string
	SerialNo             string
	font                 *opentype.Font
	face                 font.Face
	fontdrawer           *font.Drawer
	serial               *SerialWebSockConn
	conn                 *websocket.Conn
	buttonBindings       map[Button]ButtonFunc
	buttonUpBindings     map[Button]ButtonFunc
	knobBindings         map[Knob]KnobFunc
	touchBindings        map[TouchButton]TouchFunc
	touchUpBindings      map[TouchButton]TouchFunc
	transactionID        uint8
	transactionMutex     sync.Mutex
	transactionCallbacks map[byte]transactionCallback
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
		conn:                 conn,
		serial:               c,
		buttonBindings:       make(map[Button]ButtonFunc),
		buttonUpBindings:     make(map[Button]ButtonFunc),
		knobBindings:         make(map[Knob]KnobFunc),
		touchBindings:        make(map[TouchButton]TouchFunc),
		touchUpBindings:      make(map[TouchButton]TouchFunc),
		Vendor:               c.Vendor,
		Product:              c.Product,
		Model:                "foo",
		transactionCallbacks: map[byte]transactionCallback{},
	}
	l.SetDefaultFont()

	slog.Info("Found Loupedeck", "vendor", l.Vendor, "product", l.Product)

	slog.Info("Sending reset.")
	data := make([]byte, 0)
	m := l.newMessage(Reset, data)
	l.send(m)

	// Ask the device about itself.  The responses come back
	// asynchronously, so we need to provide a callback.  Since
	// `listen()` hasn't been called yet, we *have* to use
	// callbacks, blocking via 'sendAndWait' isn't going to work.
	m = l.newMessage(Version, data)
	l.sendWithCallback(m, func(m *Message) {
		l.Version = fmt.Sprintf("%d.%d.%d", m.data[0], m.data[1], m.data[2])
		slog.Info("Received 'Version' response", "version", l.Version)
	})

	m = l.newMessage(Serial, data)
	l.sendWithCallback(m, func(m *Message) {
		l.SerialNo = string(m.data)
		slog.Info("Received 'Serial' response", "serial", l.SerialNo)
	})

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
		//fmt.Printf("Measured %q at %+v\n", s, bounds)
		width := bounds.Max.X - bounds.Min.X
		height := bounds.Max.Y - bounds.Min.Y

		if width > mx26 || height > my26 {
			size = size * 0.8
			//fmt.Printf("Reducing font size to %f\n", size)
			continue
		}

		centerx := (x26 - width) / 2
		centery := (y26-height)/2 - bounds.Min.Y

		//fmt.Printf("H: %v  H: %v  Center: %v/%v\n", height, width, centerx, centery)

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

// Function SetBrightness sets the overall brightness of the Loupedeck display.
func (l *Loupedeck) SetBrightness(b int) error {
	data := make([]byte, 1)
	data[0] = byte(b)
	m := l.newMessage(SetBrightness, data)
	return l.send(m)
}

// Function SetButtonColor sets the color of a specific Button.  The
// Loupedeck Live allows the 8 buttons below the display to be set to
// specific colors, however the 'Circle' button's colors may be
// overridden to show the status of the Loupedeck Live's connection to
// the host.
func (l *Loupedeck) SetButtonColor(b Button, c color.RGBA) error {
	data := make([]byte, 4)
	data[0] = byte(b)
	data[1] = c.R
	data[2] = c.G
	data[3] = c.B
	m := l.newMessage(SetColor, data)
	return l.send(m)
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

	m := l.newMessage(WriteFramebuff, data)
	l.send(m)

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
	m2 := l.newMessage(Draw, data2)
	l.send(m2)
}
