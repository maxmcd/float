package main

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/gliderlabs/ssh"
)

type sshSessionTTYWrapper struct {
	session        ssh.Session
	width          int
	height         int
	started        bool
	ctx            context.Context
	cancel         func()
	resizeCallback func()
	lock           sync.Mutex

	reader *io.PipeReader
}

func newSSHSessionTTYWrapper(s ssh.Session) *sshSessionTTYWrapper {
	return &sshSessionTTYWrapper{session: s}
}

var _ tcell.Tty = new(sshSessionTTYWrapper)

// Start is used to activate the Tty for use.  Upon return the terminal should be
// in raw mode, non-blocking, etc.  The implementation should take care of saving
// any state that is required so that it may be restored when Stop is called.
func (s *sshSessionTTYWrapper) Start() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.started {
		return errors.New("session has already been started")
	}

	s.started = true
	s.ctx, s.cancel = context.WithCancel(context.Background())

	pty, sizeChanges, accepted := s.session.Pty()
	if !accepted {
		return errors.New("ssh session is not a pty")
	}
	s.width = pty.Window.Width
	s.height = pty.Window.Height

	reader, writer := io.Pipe()
	s.reader = reader
	go func() { _, _ = io.Copy(writer, s.session) }()

	// Start listener for size changes
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case window, ok := <-sizeChanges:
				if !ok {
					return
				}
				s.lock.Lock()
				s.width = window.Width
				s.height = window.Height
				s.lock.Unlock()
				if s.resizeCallback != nil {
					s.resizeCallback()
				}
			}
		}
	}()
	return nil
}

// Stop is used to stop using this Tty instance.  This may be a suspend, so that other
// terminal based applications can run in the foreground.  Implementations should
// restore any state collected at Start(), and return to ordinary blocking mode, etc.
// Drain is called first to drain the input.  Once this is called, no more Read
// or Write calls will be made until Start is called again.
func (s *sshSessionTTYWrapper) Stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.started {
		return errors.New("session hasn't been started")
	}
	s.started = false
	s.cancel()

	// Don't call s.session.Close() in case the parent would like to use the connection further.
	return nil
}

// Drain is called before Stop, and ensures that the reader will wake up appropriately
// if it was blocked.  This workaround is required for /dev/tty on certain UNIX systems
// to ensure that Read() does not block forever.  This typically arranges for the tty driver
// to send data immediately (e.g. VMIN and VTIME both set zero) and sets a deadline on input.
// Implementations may reasonably make this a no-op.  There will still be control sequences
// emitted between the time this is called, and when Stop is called.
func (s *sshSessionTTYWrapper) Drain() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.reader.Close()
}

// NotifyResize is used register a callback when the tty thinks the dimensions have
// changed.  The standard UNIX implementation links this to a handler for SIGWINCH.
// If the supplied callback is nil, then any handler should be unregistered.
func (s *sshSessionTTYWrapper) NotifyResize(cb func()) { s.resizeCallback = cb }

// WindowSize is called to determine the terminal dimensions.  This might be determined
// by an ioctl or other means.
func (s *sshSessionTTYWrapper) WindowSize() (width int, height int, err error) {
	return s.width, s.height, nil
}

func (s *sshSessionTTYWrapper) Write(p []byte) (n int, err error) { return s.session.Write(p) }
func (s *sshSessionTTYWrapper) Read(p []byte) (n int, err error)  { return s.reader.Read(p) }
func (s *sshSessionTTYWrapper) Close() (err error)                { return nil }
