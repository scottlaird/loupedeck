package loupedeck

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log/slog"
	"net"
	"net/http"
	"time"
)

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
		displays:             map[string]*Display{},
	}
	err = l.SetDefaultFont()
	if err != nil {
		return nil, fmt.Errorf("Unable to set default font: %v", err)
	}

	slog.Info("Found Loupedeck", "vendor", l.Vendor, "product", l.Product)

	slog.Info("Sending reset.")
	data := make([]byte, 0)
	m := l.NewMessage(Reset, data)
	err = l.Send(m)
	if err != nil {
		return nil, fmt.Errorf("Unable to send: %v", err)
	}

	slog.Info("Setting default brightness.")
	data = []byte{9}
	m = l.NewMessage(SetBrightness, data)
	err = l.Send(m)
	if err != nil {
		return nil, fmt.Errorf("Unable to send: %v", err)
	}

	// Ask the device about itself.  The responses come back
	// asynchronously, so we need to provide a callback.  Since
	// `listen()` hasn't been called yet, we *have* to use
	// callbacks, blocking via 'sendAndWait' isn't going to work.
	m = l.NewMessage(Version, data)
	err = l.SendWithCallback(m, func(m *Message) {
		l.Version = fmt.Sprintf("%d.%d.%d", m.data[0], m.data[1], m.data[2])
		slog.Info("Received 'Version' response", "version", l.Version)
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to send: %v", err)
	}

	m = l.NewMessage(Serial, data)
	err = l.SendWithCallback(m, func(m *Message) {
		l.SerialNo = string(m.data)
		slog.Info("Received 'Serial' response", "serial", l.SerialNo)
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to send: %v", err)
	}

	return l, nil
}
