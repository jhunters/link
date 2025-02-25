package link

import (
	"errors"
	"sync"
	"sync/atomic"
)

var SessionClosedError = errors.New("Session Closed")
var SessionBlockedError = errors.New("Session Blocked")

var globalSessionId uint64

type Session[S, R any] struct {
	id        uint64
	codec     Codec[S, R]
	manager   *Manager[S, R]
	sendChan  chan S
	recvMutex sync.Mutex
	sendMutex sync.RWMutex

	closeFlag          int32
	closeChan          chan int
	closeMutex         sync.Mutex
	firstCloseCallback *closeCallback
	lastCloseCallback  *closeCallback

	State interface{}
}

func NewSession[S, R any](codec Codec[S, R], sendChanSize int) *Session[S, R] {
	return newSession(nil, codec, sendChanSize)
}

func newSession[S, R any](manager *Manager[S, R], codec Codec[S, R], sendChanSize int) *Session[S, R] {
	session := &Session[S, R]{
		codec:     codec,
		manager:   manager,
		closeChan: make(chan int),
		id:        atomic.AddUint64(&globalSessionId, 1),
	}
	if sendChanSize > 0 {
		session.sendChan = make(chan S, sendChanSize)
		go session.sendLoop()
	}
	return session
}

func (session *Session[S, R]) ID() uint64 {
	return session.id
}

func (session *Session[S, R]) IsClosed() bool {
	return atomic.LoadInt32(&session.closeFlag) == 1
}

func (session *Session[S, R]) Close() error {
	if atomic.CompareAndSwapInt32(&session.closeFlag, 0, 1) {
		close(session.closeChan)

		if session.sendChan != nil {
			session.sendMutex.Lock()
			close(session.sendChan)
			if clear, ok := session.codec.(ClearSendChan[S]); ok {
				clear.ClearSendChan(session.sendChan)
			}
			session.sendMutex.Unlock()
		}

		err := session.codec.Close()

		go func() {
			session.invokeCloseCallbacks()

			if session.manager != nil {
				session.manager.delSession(session)
			}
		}()
		return err
	}
	return SessionClosedError
}

func (session *Session[S, R]) Codec() Codec[S, R] {
	return session.codec
}

func (session *Session[S, R]) Receive() (R, error) {
	session.recvMutex.Lock()
	defer session.recvMutex.Unlock()

	msg, err := session.codec.Receive()
	if err != nil {
		session.Close()
	}
	return msg, err
}

func (session *Session[S, R]) sendLoop() {
	defer session.Close()
	for {
		select {
		case msg, ok := <-session.sendChan:
			if !ok || session.codec.Send(msg) != nil {
				return
			}
		case <-session.closeChan:
			return
		}
	}
}

func (session *Session[S, R]) Send(msg S) error {
	if session.sendChan == nil {
		if session.IsClosed() {
			return SessionClosedError
		}

		session.sendMutex.Lock()
		defer session.sendMutex.Unlock()

		err := session.codec.Send(msg)
		if err != nil {
			session.Close()
		}
		return err
	}

	session.sendMutex.RLock()
	if session.IsClosed() {
		session.sendMutex.RUnlock()
		return SessionClosedError
	}

	select {
	case session.sendChan <- msg:
		session.sendMutex.RUnlock()
		return nil
	default:
		session.sendMutex.RUnlock()
		session.Close()
		return SessionBlockedError
	}
}

type closeCallback struct {
	Handler interface{}
	Key     interface{}
	Func    func()
	Next    *closeCallback
}

func (session *Session[S, R]) AddCloseCallback(handler, key interface{}, callback func()) {
	if session.IsClosed() {
		return
	}

	session.closeMutex.Lock()
	defer session.closeMutex.Unlock()

	newItem := &closeCallback{handler, key, callback, nil}

	if session.firstCloseCallback == nil {
		session.firstCloseCallback = newItem
	} else {
		session.lastCloseCallback.Next = newItem
	}
	session.lastCloseCallback = newItem
}

func (session *Session[S, R]) RemoveCloseCallback(handler, key interface{}) {
	if session.IsClosed() {
		return
	}

	session.closeMutex.Lock()
	defer session.closeMutex.Unlock()

	var prev *closeCallback
	for callback := session.firstCloseCallback; callback != nil; prev, callback = callback, callback.Next {
		if callback.Handler == handler && callback.Key == key {
			if session.firstCloseCallback == callback {
				session.firstCloseCallback = callback.Next
			} else {
				prev.Next = callback.Next
			}
			if session.lastCloseCallback == callback {
				session.lastCloseCallback = prev
			}
			return
		}
	}
}

func (session *Session[S, R]) invokeCloseCallbacks() {
	session.closeMutex.Lock()
	defer session.closeMutex.Unlock()

	for callback := session.firstCloseCallback; callback != nil; callback = callback.Next {
		callback.Func()
	}
}
