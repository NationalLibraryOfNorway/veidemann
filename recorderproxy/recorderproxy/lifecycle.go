package recorderproxy

import (
	"context"
	"sync"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
)

// lifecycleTracker owns work belonging to one RecorderProxy. A condition-like
// change channel is used instead of a WaitGroup because an accepted handler may
// create a RecordContext after shutdown has started waiting.
type lifecycleTracker struct {
	mu      sync.Mutex
	closed  bool
	changed chan struct{}
	conns   map[*wrappedConnection]struct{}
	records map[*rpcontext.RecordContext]struct{}
}

func newLifecycleTracker() *lifecycleTracker {
	return &lifecycleTracker{
		changed: make(chan struct{}),
		conns:   make(map[*wrappedConnection]struct{}),
		records: make(map[*rpcontext.RecordContext]struct{}),
	}
}

func (l *lifecycleTracker) notifyLocked() {
	close(l.changed)
	l.changed = make(chan struct{})
}

func (l *lifecycleTracker) addConnection(conn *wrappedConnection) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return false
	}
	l.conns[conn] = struct{}{}
	l.notifyLocked()
	return true
}

func (l *lifecycleTracker) removeConnection(conn *wrappedConnection) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.conns[conn]; ok {
		delete(l.conns, conn)
		l.notifyLocked()
	}
}

func (l *lifecycleTracker) addRecord(rc *rpcontext.RecordContext) {
	l.mu.Lock()
	l.records[rc] = struct{}{}
	l.notifyLocked()
	l.mu.Unlock()

	previousClose := rc.CloseFunc
	var once sync.Once
	rc.CloseFunc = func() {
		once.Do(func() {
			if previousClose != nil {
				previousClose()
			}
			l.mu.Lock()
			delete(l.records, rc)
			l.notifyLocked()
			l.mu.Unlock()
		})
	}
}

func (l *lifecycleTracker) closeAndSnapshotConnections() []*wrappedConnection {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	connections := make([]*wrappedConnection, 0, len(l.conns))
	for conn := range l.conns {
		connections = append(connections, conn)
	}
	l.notifyLocked()
	return connections
}

// drained requires closed admission as well as an empty tracker.
func (l *lifecycleTracker) drained() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closed && len(l.conns) == 0 && len(l.records) == 0
}

func (l *lifecycleTracker) wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		if len(l.conns) == 0 && len(l.records) == 0 {
			l.mu.Unlock()
			return nil
		}
		changed := l.changed
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}
