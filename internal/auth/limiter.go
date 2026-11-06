package auth

import (
	"sync"
	"time"
)

const (
	freeAttempts = 5
	lockBase     = time.Minute
	lockMax      = 15 * time.Minute
	forgetAfter  = time.Hour
)

type attempts struct {
	failures int
	lockedTo time.Time
	last     time.Time
}

// Limiter slows password guessing per client address. Lockouts double from one
// minute up to fifteen once the free attempts are used.
type Limiter struct {
	mu      sync.Mutex
	clients map[string]*attempts
	now     func() time.Time
}

func NewLimiter() *Limiter {
	return &Limiter{clients: map[string]*attempts{}, now: time.Now}
}

// Allow reports whether an attempt may run, and if not, how long to wait.
func (l *Limiter) Allow(client string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.clients[client]
	if !ok {
		return true, 0
	}
	if wait := entry.lockedTo.Sub(l.now()); wait > 0 {
		return false, wait
	}
	return true, 0
}

func (l *Limiter) Fail(client string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for key, entry := range l.clients {
		if now.Sub(entry.last) > forgetAfter {
			delete(l.clients, key)
		}
	}
	entry, ok := l.clients[client]
	if !ok {
		entry = &attempts{}
		l.clients[client] = entry
	}
	entry.failures++
	entry.last = now
	if over := entry.failures - freeAttempts; over >= 0 {
		lock := lockBase
		for i := 0; i < over && lock < lockMax; i++ {
			lock *= 2
		}
		entry.lockedTo = now.Add(min(lock, lockMax))
	}
}

func (l *Limiter) Reset(client string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.clients, client)
}
