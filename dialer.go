/*
 */
package loupedeck

import (
	//"github.com/tarm/serial"
	"fmt"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
	"log/slog"
	"net"
	"time"
)

// The Gorilla websockets library can use an external dialer
// interface, which means that we can use it *mostly* unmodified to
// talk to a serial device instead of a network device.  We just need
// to provide something that matches the net.Conn interface.  Here's a
// minimal implementation.
type SerialWebSockConn struct {
	Name string
	Port serial.Port
}

func (l *SerialWebSockConn) Read(b []byte) (n int, err error) {
	slog.Info("Reading", "limit_bytes", len(b))
	n, err = l.Port.Read(b)
	slog.Info("Read", "bytes", n, "err", err)
	return n, err
}

func (l *SerialWebSockConn) Write(b []byte) (n int, err error) {
	slog.Info("Writing", "bytes", len(b), "message", b)
	return l.Port.Write(b)
}

func (l *SerialWebSockConn) Close() error {
	return nil // l.Port.Close()
}

func (l *SerialWebSockConn) LocalAddr() net.Addr {
	return nil
}
func (l *SerialWebSockConn) RemoteAddr() net.Addr {
	return nil
}

func (l *SerialWebSockConn) SetDeadline(t time.Time) error {
	return nil
}
func (l *SerialWebSockConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (l *SerialWebSockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func ConnectSerialAuto() (*SerialWebSockConn, error) {
	slog.Info("Enumerating ports")

	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("No serial ports found.")
	}

	for _, port := range ports {
		slog.Info("Trying to open port", "port", port.Name)
		if port.IsUSB && (port.VID == "2ec2" || port.VID == "1532") {
			p, err := serial.Open(port.Name, &serial.Mode{})
			if err != nil {
				return nil, fmt.Errorf("Unable to open port %q", port.Name)
			}
			conn := &SerialWebSockConn{
				Name: port.Name,
				Port: p,
			}
			slog.Info("Port good, continuing")

			return conn, nil
		}
	}

	return nil, fmt.Errorf("No Loupedeck devices found.")
}

func ConnectSerialPath(serialPath string) (*SerialWebSockConn, error) {
	p, err := serial.Open(serialPath, &serial.Mode{})
	if err != nil {
		return nil, fmt.Errorf("Unable to open serial device %q", serialPath)
	}
	conn := &SerialWebSockConn{
		Name: serialPath,
		Port: p,
	}

	return conn, nil
}
