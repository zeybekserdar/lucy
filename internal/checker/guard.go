package checker

import (
	"context"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Guard tells a dead home connection apart from a dead service, so an
// internet outage does not turn into an alarm for every external monitor.
type Guard struct {
	targets []string
	ttl     time.Duration

	mu           sync.Mutex
	probed       time.Time
	online       bool
	offlineSince time.Time
}

func NewGuard(targets []string) *Guard {
	return &Guard{targets: targets, ttl: 15 * time.Second, online: true}
}

// Online probes the targets, reusing a recent answer. Any reachable target
// counts as online.
func (g *Guard) Online(ctx context.Context) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if time.Since(g.probed) < g.ttl {
		return g.online
	}
	g.probed = time.Now()
	g.set(g.probe(ctx))
	return g.online
}

// MarkOnline records proof of connectivity, such as a successful external check.
func (g *Guard) MarkOnline() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.set(true)
}

// State returns the current view without probing.
func (g *Guard) State() (bool, time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.online, g.offlineSince
}

func (g *Guard) set(online bool) {
	if !online && g.online {
		g.offlineSince = time.Now()
	}
	if online {
		g.offlineSince = time.Time{}
	}
	g.online = online
}

func (g *Guard) probe(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	results := make(chan bool, len(g.targets))
	for _, target := range g.targets {
		go func() {
			conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", target)
			if err == nil {
				conn.Close()
			}
			results <- err == nil
		}()
	}
	for range g.targets {
		if <-results {
			return true
		}
	}
	return false
}

// IsLocal reports whether an address stays inside the home network, where an
// internet outage cannot be the cause of a failure.
func IsLocal(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
	}
	if !strings.Contains(host, ".") {
		return true
	}
	for _, suffix := range []string{".home", ".local", ".lan", ".internal", ".localhost", ".home.arpa"} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
