//go:build windows

package core

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

const (
	PIPE_ACCESS_DUPLEX       = 0x00000003
	PIPE_TYPE_BYTE           = 0x00000000
	PIPE_READMODE_BYTE       = 0x00000000
	PIPE_WAIT                = 0x00000000
	PIPE_UNLIMITED_INSTANCES = 255
)

type winPipeConn struct {
	handle   windows.Handle
	isServer bool
}

func (c *winPipeConn) Read(b []byte) (int, error) {
	var done uint32
	err := windows.ReadFile(c.handle, b, &done, nil)
	if err != nil {
		if err == windows.ERROR_BROKEN_PIPE || err == windows.ERROR_PIPE_NOT_CONNECTED {
			return int(done), fmt.Errorf("pipe closed: %w", err)
		}
		return int(done), err
	}
	return int(done), nil
}

func (c *winPipeConn) Write(b []byte) (int, error) {
	var done uint32
	err := windows.WriteFile(c.handle, b, &done, nil)
	return int(done), err
}

func (c *winPipeConn) Close() error {
	if c.isServer {
		_ = windows.FlushFileBuffers(c.handle)
		_ = windows.DisconnectNamedPipe(c.handle)
	}
	return windows.CloseHandle(c.handle)
}

func (c *winPipeConn) LocalAddr() net.Addr                { return pipeAddr(UI_PIPE_NAME) }
func (c *winPipeConn) RemoteAddr() net.Addr               { return pipeAddr(UI_PIPE_NAME) }
func (c *winPipeConn) SetDeadline(t time.Time) error      { return nil }
func (c *winPipeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *winPipeConn) SetWriteDeadline(t time.Time) error { return nil }

type pipeAddr string

func (a pipeAddr) Network() string { return "pipe" }
func (a pipeAddr) String() string  { return string(a) }

type winPipeListener struct {
	pipeName string
	closed   bool
	mu       sync.Mutex
}

func OpenPipeListener(pipeName string) (net.Listener, error) {
	return &winPipeListener{
		pipeName: pipeName,
	}, nil
}

// DialPipe connects to a Windows Named Pipe via Win32 CreateFile.
func DialPipe(pipeName string, timeout time.Duration) (net.Conn, error) {
	namePtr, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	for {
		h, err := windows.CreateFile(
			namePtr,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			0,
			0,
		)
		if err == nil {
			return &winPipeConn{handle: h, isServer: false}, nil
		}
		if time.Since(start) > timeout {
			return nil, fmt.Errorf("timeout connecting to pipe %s: %w", pipeName, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (l *winPipeListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, net.ErrClosed
	}
	l.mu.Unlock()

	namePtr, err := windows.UTF16PtrFromString(l.pipeName)
	if err != nil {
		return nil, err
	}

	h, err := windows.CreateNamedPipe(
		namePtr,
		PIPE_ACCESS_DUPLEX,
		PIPE_TYPE_BYTE|PIPE_READMODE_BYTE|PIPE_WAIT,
		PIPE_UNLIMITED_INSTANCES,
		4096,
		4096,
		0,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateNamedPipe failed: %w", err)
	}

	err = windows.ConnectNamedPipe(h, nil)
	if err != nil && err != windows.ERROR_PIPE_CONNECTED {
		_ = windows.CloseHandle(h)
		return nil, err
	}

	return &winPipeConn{handle: h, isServer: true}, nil
}

func (l *winPipeListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return nil
}

func (l *winPipeListener) Addr() net.Addr {
	return pipeAddr(l.pipeName)
}
