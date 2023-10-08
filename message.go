package loupedeck

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log/slog"
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
)

type Message struct {
	transactionID byte
	messageType   MessageType
	length        byte
	data          []byte
}

func (l *Loupedeck) newMessage(messageType MessageType, data []byte) *Message {
	length := len(data) + 3
	if length>255 {
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

func (l *Loupedeck) parseMessage(b []byte) (*Message, error) {
	m := Message{
		length: b[0],
		messageType: MessageType(b[1]),
		transactionID: b[2],
		data: b[3:],
	}
	return &m, nil
}

// function AsBytes() returns the wire-format form of the message
func (m *Message) AsBytes() []byte {
	b := make([]byte, 3)
	b[0] = m.length
	b[1] = byte(m.messageType)
	b[2] = m.transactionID
	b = append(b, m.data...)

	return b
}

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

func (l *Loupedeck) send(m *Message) error {
	slog.Info("Sending","message", m.String())
	b := m.AsBytes()
	l.conn.WriteMessage(websocket.BinaryMessage, b)

	return nil
}

// Function sendMessage sends a formatted message to the Loupedeck.
func (l *Loupedeck) sendMessage(h MessageType, data []byte) error {
	transactionID := l.newTransactionID()
	b := make([]byte, 3) // should probably add len(data) to make append() cheaper.

	// The Loupedeck protocol only uses a single byte for lengths,
	// but big images, etc, are larger than that.  Since the
	// length field is only 8 bits, it uses 255 to mean "255 or
	// larger".  Given that, I'm not sure why it has a length
	// field at all, but whatever.
	length := 3 + len(data)
	if length > 255 {
		length = 255
	}

	b[0] = byte(length)
	b[1] = byte(h)
	b[2] = byte(transactionID)
	b = append(b, data...)

	if len(b) > 32 {
		slog.Info("Sendmessage", "header type", h, "len", len(b), "data", fmt.Sprintf("%v", b[0:32]))
	} else {
		slog.Info("Sendmessage", "header type", h, "len", len(b), "data", fmt.Sprintf("%v", b))
	}

	l.conn.WriteMessage(websocket.BinaryMessage, b)
	//l.serial.Write(b)
	return nil
}
