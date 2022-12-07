package link

import (
	"sync"
)

type KEY interface{}

type Channel[S, R any] struct {
	mutex    sync.RWMutex
	sessions map[KEY]*Session[S, R]

	// channel state
	State interface{}
}

func NewChannel[S, R any]() *Channel[S, R] {
	return &Channel[S, R]{
		sessions: make(map[KEY]*Session[S, R]),
	}
}

func (channel *Channel[S, R]) Len() int {
	channel.mutex.RLock()
	defer channel.mutex.RUnlock()
	return len(channel.sessions)
}

func (channel *Channel[S, R]) Fetch(callback func(*Session[S, R])) {
	channel.mutex.RLock()
	defer channel.mutex.RUnlock()
	for _, session := range channel.sessions {
		callback(session)
	}
}

func (channel *Channel[S, R]) Get(key KEY) *Session[S, R] {
	channel.mutex.RLock()
	defer channel.mutex.RUnlock()
	session, _ := channel.sessions[key]
	return session
}

func (channel *Channel[S, R]) Put(key KEY, session *Session[S, R]) {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	if session, exists := channel.sessions[key]; exists {
		channel.remove(key, session)
	}
	session.AddCloseCallback(channel, key, func() {
		channel.Remove(key)
	})
	channel.sessions[key] = session
}

func (channel *Channel[S, R]) remove(key KEY, session *Session[S, R]) {
	session.RemoveCloseCallback(channel, key)
	delete(channel.sessions, key)
}

func (channel *Channel[S, R]) Remove(key KEY) bool {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	session, exists := channel.sessions[key]
	if exists {
		channel.remove(key, session)
	}
	return exists
}

func (channel *Channel[S, R]) FetchAndRemove(callback func(*Session[S, R])) {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	for key, session := range channel.sessions {
		session.RemoveCloseCallback(channel, key)
		delete(channel.sessions, key)
		callback(session)
	}
}

func (channel *Channel[S, R]) Close() {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	for key, session := range channel.sessions {
		channel.remove(key, session)
	}
}
