package notify

import (
	"fmt"
	"strings"
	"time"

	"lucy/internal/store"
)

type phrases struct {
	down, recovered, lasted         string
	hour, minute, second            string
	timeout, dns, refused, tls, net string
	body, response, json            string
	sslSoon, sslDone                string
	domainSoon, domainDone          string
	expiresIn, expired              string
	testTitle, testBody             string
}

var languages = map[string]phrases{
	"tr": {
		down: "%s kesintide", recovered: "%s geri geldi", lasted: "Kesinti %s sürdü",
		hour: "sa", minute: "dk", second: "sn",
		timeout: "Zaman aşımı", dns: "Alan adı çözülemedi", refused: "Bağlantı reddedildi",
		tls: "TLS sertifikası doğrulanamadı", net: "Bağlantı hatası",
		body: "Gövde koşulu sağlanmadı", response: "Yanıt süresi sınırı aşıldı", json: "JSON koşulu sağlanmadı",
		sslSoon: "%s: SSL sertifikası bitiyor", sslDone: "%s: SSL sertifikasının süresi doldu",
		domainSoon: "%s: alan adı bitiyor", domainDone: "%s: alan adının süresi doldu",
		expiresIn: "%d gün kaldı · %s", expired: "Bitiş tarihi %s",
		testTitle: "Lucy test bildirimi", testBody: "Bildirimler çalışıyor.",
	},
	"en": {
		down: "%s is down", recovered: "%s is back", lasted: "The outage lasted %s",
		hour: "h", minute: "min", second: "s",
		timeout: "Timed out", dns: "Could not resolve the domain", refused: "Connection refused",
		tls: "TLS certificate could not be verified", net: "Connection error",
		body: "Body condition not met", response: "Response time limit exceeded", json: "JSON condition not met",
		sslSoon: "%s: SSL certificate is expiring", sslDone: "%s: SSL certificate has expired",
		domainSoon: "%s: domain is expiring", domainDone: "%s: domain has expired",
		expiresIn: "%d days left · %s", expired: "Expiry date %s",
		testTitle: "Lucy test notification", testBody: "Notifications are working.",
	},
}

// Composer words notifications in the configured language.
type Composer struct {
	Language  string
	PublicURL string
}

func (c Composer) phrases() phrases {
	if p, ok := languages[c.Language]; ok {
		return p
	}
	return languages["tr"]
}

func (c Composer) click(id string) string {
	if c.PublicURL == "" {
		return ""
	}
	return strings.TrimRight(c.PublicURL, "/") + "/m/" + id
}

func (c Composer) Test() store.Notification {
	p := c.phrases()
	return store.Notification{
		Title: p.testTitle, Message: p.testBody, Priority: 3, Tags: []string{"white_check_mark"},
		Click: c.PublicURL, // the test also proves the address opens on the phone
	}
}

func (c Composer) Down(id, name, url, failure string) *store.Notification {
	p := c.phrases()
	return &store.Notification{
		Title:    fmt.Sprintf(p.down, name),
		Message:  describe(p, failure) + " · " + url,
		Priority: 4,
		Tags:     []string{"rotating_light"},
		Click:    c.click(id),
	}
}

func (c Composer) Recovered(id, name string, lasted time.Duration) *store.Notification {
	p := c.phrases()
	return &store.Notification{
		Title:    fmt.Sprintf(p.recovered, name),
		Message:  fmt.Sprintf(p.lasted, duration(p, lasted)),
		Priority: 3,
		Tags:     []string{"white_check_mark"},
		Click:    c.click(id),
	}
}

// Expiry words a certificate ("ssl") or domain ("domain") warning.
func (c Composer) Expiry(id, name, kind string, daysLeft int, expires time.Time) *store.Notification {
	p := c.phrases()
	soon, done := p.sslSoon, p.sslDone
	if kind == "domain" {
		soon, done = p.domainSoon, p.domainDone
	}
	date := expires.Format("2006-01-02")
	n := &store.Notification{Priority: 3, Tags: []string{"warning"}, Click: c.click(id)}
	if daysLeft <= 0 {
		n.Title = fmt.Sprintf(done, name)
		n.Message = fmt.Sprintf(p.expired, date)
		n.Priority = 4
		return n
	}
	n.Title = fmt.Sprintf(soon, name)
	n.Message = fmt.Sprintf(p.expiresIn, daysLeft, date)
	return n
}

func describe(p phrases, failure string) string {
	kind, detail, _ := strings.Cut(failure, ":")
	texts := map[string]string{
		"net:timeout":   p.timeout,
		"net:dns":       p.dns,
		"net:refused":   p.refused,
		"net:tls":       p.tls,
		"cond:body":     p.body,
		"cond:response": p.response,
		"cond:json":     p.json,
	}
	if kind == "http" {
		return "HTTP " + detail
	}
	if text, ok := texts[failure]; ok {
		return text
	}
	return p.net
}

func duration(p phrases, d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%d %s", seconds, p.second)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%d %s", minutes, p.minute)
	}
	if rest := minutes % 60; rest > 0 {
		return fmt.Sprintf("%d %s %d %s", minutes/60, p.hour, rest, p.minute)
	}
	return fmt.Sprintf("%d %s", minutes/60, p.hour)
}
