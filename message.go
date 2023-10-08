package loupedeck

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log/slog"
	"time"
)

// Type MessageType is a uint16 used to identify various commands and
// actions needed for the Loupedeck protocol.
type MessageType byte

// See 'COMMANDS' in https://github.com/foxxyz/loupedeck/blob/master/constants.js
const (
	ButtonPress      MessageType = 0x00
	KnobRotate                   = 0x01
	SetColor                     = 0x02
	Serial                       = 0x03
	Reset                        = 0x06
	Version                      = 0x07
	SetBrightness                = 0x09
	MCU                          = 0x0d
	WriteFramebuff               = 0x10
	Draw                         = 0x0f
	ConfirmFramebuff             = 0x10
	SetVibration                 = 0x1b
	Touch                        = 0x4d
	TouchEnd                     = 0x6d
	TouchEndCT                   = 0x72
)

// Type Message defines a message for communicating with the Loupedeck
// over USB.  All communication with the Loupedeck occurs via
// Messages, but most application software can use higher-level
// functions in this library and never touch messages directly.
//
// The exception would be wanting to use a feature like vibration that
// isn't currently supported in this library.
type Message struct {
	transactionID byte
	messageType   MessageType
	length        byte
	data          []byte
}

// Function NewMessage creates a new low-level Loupedeck message with
// a specified type and data.  This isn't generally needed for
// end-use.
func (l *Loupedeck) NewMessage(messageType MessageType, data []byte) *Message {
	length := len(data) + 3
	if length > 255 {
		length = 255
	}

	m := Message{
		transactionID: l.newTransactionID(),
		messageType:   messageType,
		length:        byte(length),
		data:          data,
	}

	return &m
}

// Function ParseMessage creates a Loupedeck Message from a block of
// bytes.  This is used to decode incoming messages from a Loupedeck,
// and shouldn't generally be needed outside of this library.
func (l *Loupedeck) ParseMessage(b []byte) (*Message, error) {
	m := Message{
		length:        b[0],
		messageType:   MessageType(b[1]),
		transactionID: b[2],
		data:          b[3:],
	}
	return &m, nil
}

// function asBytes() returns the wire-format form of the message.
func (m *Message) asBytes() []byte {
	b := make([]byte, 3)
	b[0] = m.length
	b[1] = byte(m.messageType)
	b[2] = m.transactionID
	b = append(b, m.data...)

	return b
}

// function String() returns a human-readable form of the message for
// debugging use.
func (m *Message) String() string {
	d := m.data

	if len(d) > 8 {
		d = d[0:8]
	}

	return fmt.Sprintf("{len: %d, type: %02x, txn: %02x, data: %v}", m.length, m.messageType, m.transactionID, d)
}

// Function newTransactionId picks the next 8-bit transaction ID
// number.  This is used as part of the Loupedeck protocol and used to
// match results with specific queries.  The transaction ID
// incrememnts per call and rolls over back to 1 (not 0).
func (l *Loupedeck) newTransactionID() uint8 {
	l.transactionMutex.Lock()
	t := l.transactionID
	t++
	if t == 0 {
		t = 1
	}
	l.transactionID = t
	l.transactionMutex.Unlock()

	return t
}

// Function Send sends a message to the specified device.
func (l *Loupedeck) Send(m *Message) error {
	slog.Info("Sending", "message", m.String())
	b := m.asBytes()
	l.conn.WriteMessage(websocket.BinaryMessage, b)

	return nil
}

// Function SendWithCallback sends a message to the specified device
// and registers a callback.  When (or if) the Loupedeck sends a
// response to the message, the callback function will be called and
// provided with the response message.
func (l *Loupedeck) SendWithCallback(m *Message, c transactionCallback) error {
	slog.Info("Setting callback", "message", m.String())
	l.transactionCallbacks[m.transactionID] = c

	return l.Send(m)
}

// function SendAndWait sends a message and then waits for a response, returning the response message.
func (l *Loupedeck) sendAndWait(m *Message, timeout time.Duration) (resp *Message, err error) {
	ch := make(chan *Message)
	defer close(ch)
	// TODO(scottlaird): actually implement the timeout.
	l.SendWithCallback(m, func(m2 *Message) {
		slog.Info("sendAndWait callback received, sending to channel")
		ch <- m2
	})

	select {
	case resp = <-ch:
		slog.Info("sendAndWait received ok")
		return resp, nil
	case <-time.After(timeout):
		slog.Warn("sendAndWait timeout")
		return nil, fmt.Errorf("Timeout waiting for response")
	}
}