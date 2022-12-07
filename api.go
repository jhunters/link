package link

import (
	"io"
	"net"
	"strings"
	"time"
)

type Protocol[S, R any] interface {
	NewCodec(rw io.ReadWriter) (Codec[S, R], error)
}

type ProtocolFunc[S, R any] func(rw io.ReadWriter) (Codec[S, R], error)

func (pf ProtocolFunc[S, R]) NewCodec(rw io.ReadWriter) (Codec[S, R], error) {
	return pf(rw)
}

type Codec[S, R any] interface {
	Receive() (R, error)
	Send(S) error
	Close() error
}

type ClearSendChan[E any] interface {
	ClearSendChan(<-chan E)
}

func Listen[S, R any](network, address string, protocol Protocol[S, R], sendChanSize int, handler HandlerFunc[S, R]) (*Server[S, R], error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return NewServer(listener, protocol, sendChanSize, handler), nil
}

func Dial[S, R any](network, address string, protocol Protocol[S, R], sendChanSize int) (*Session[S, R], error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	codec, err := protocol.NewCodec(conn)
	if err != nil {
		return nil, err
	}
	return NewSession(codec, sendChanSize), nil
}

func DialTimeout[S, R any](network, address string, timeout time.Duration, protocol Protocol[S, R], sendChanSize int) (*Session[S, R], error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	codec, err := protocol.NewCodec(conn)
	if err != nil {
		return nil, err
	}
	return NewSession(codec, sendChanSize), nil
}

func Accept(listener net.Listener) (net.Conn, error) {
	var tempDelay time.Duration
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil, io.EOF
			}
			return nil, err
		}
		return conn, nil
	}
}
