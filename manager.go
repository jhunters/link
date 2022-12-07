package link

import "sync"

const sessionMapNum = 32

type Manager[S, R any] struct {
	sessionMaps [sessionMapNum]sessionMap[S, R]
	disposeOnce sync.Once
	disposeWait sync.WaitGroup
}

type sessionMap[S, R any] struct {
	sync.RWMutex
	sessions map[uint64]*Session[S, R]
	disposed bool
}

func NewManager[S, R any]() *Manager[S, R] {
	manager := &Manager[S, R]{}
	for i := 0; i < len(manager.sessionMaps); i++ {
		manager.sessionMaps[i].sessions = make(map[uint64]*Session[S, R])
	}
	return manager
}

func (manager *Manager[S, R]) Dispose() {
	manager.disposeOnce.Do(func() {
		for i := 0; i < sessionMapNum; i++ {
			smap := &manager.sessionMaps[i]
			smap.Lock()
			smap.disposed = true
			for _, session := range smap.sessions {
				session.Close()
			}
			smap.Unlock()
		}
		manager.disposeWait.Wait()
	})
}

func (manager *Manager[S, R]) NewSession(codec Codec[S, R], sendChanSize int) *Session[S, R] {
	session := newSession(manager, codec, sendChanSize)
	manager.putSession(session)
	return session
}

func (manager *Manager[S, R]) GetSession(sessionID uint64) *Session[S, R] {
	smap := &manager.sessionMaps[sessionID%sessionMapNum]
	smap.RLock()
	defer smap.RUnlock()

	session, _ := smap.sessions[sessionID]
	return session
}

func (manager *Manager[S, R]) putSession(session *Session[S, R]) {
	smap := &manager.sessionMaps[session.id%sessionMapNum]

	smap.Lock()
	defer smap.Unlock()

	if smap.disposed {
		session.Close()
		return
	}

	smap.sessions[session.id] = session
	manager.disposeWait.Add(1)
}

func (manager *Manager[S, R]) delSession(session *Session[S, R]) {
	smap := &manager.sessionMaps[session.id%sessionMapNum]

	smap.Lock()
	defer smap.Unlock()

	if _, ok := smap.sessions[session.id]; ok {
		delete(smap.sessions, session.id)
		manager.disposeWait.Done()
	}
}
