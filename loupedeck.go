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
	"github.com/gorilla/websocket"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
//	"log/slog"
	"sync"
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


func (l *Loupedeck) Close() {
	l.conn.Close()
	l.serial.Close()
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
	m := l.NewMessage(SetBrightness, data)
	return l.Send(m)
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
	m := l.NewMessage(SetColor, data)
	return l.Send(m)
}
