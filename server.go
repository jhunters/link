package link

import "net"

type Server[S, R any] struct {
	manager      *Manager[S, R]
	listener     net.Listener
	protocol     Protocol[S, R]
	handler      HandlerFunc[S, R]
	sendChanSize int
}

type Handler[S, R any] interface {
	HandleSession(*Session[S, R])
}

type HandlerFunc[S, R any] func(*Session[S, R])

func (f HandlerFunc[S, R]) HandleSession(session *Session[S, R]) {
	f(session)
}

func NewServer[S, R any](listener net.Listener, protocol Protocol[S, R], sendChanSize int, handler HandlerFunc[S, R]) *Server[S, R] {
	return &Server[S, R]{
		manager:      NewManager[S, R](),
		listener:     listener,
		protocol:     protocol,
		handler:      handler,
		sendChanSize: sendChanSize,
	}
}

func (server *Server[S, R]) Listener() net.Listener {
	return server.listener
}

func (server *Server[S, R]) Serve() error {
	for {
		conn, err := Accept(server.listener)
		if err != nil {
			return err
		}

		go func() {
			codec, err := server.protocol.NewCodec(conn)
			if err != nil {
				conn.Close()
				return
			}
			session := server.manager.NewSession(codec, server.sendChanSize)
			server.handler.HandleSession(session)
		}()
	}
}

func (server *Server[S, R]) GetSession(sessionID uint64) *Session[S, R] {
	return server.manager.GetSession(sessionID)
}

func (server *Server[S, R]) Stop() {
	server.listener.Close()
	server.manager.Dispose()
}
