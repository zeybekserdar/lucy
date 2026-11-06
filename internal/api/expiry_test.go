package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"lucy/internal/checker"
	"lucy/internal/model"
)

type fakeDomains struct {
	mu    sync.Mutex
	info  checker.DomainInfo
	err   error
	calls int
}

func (f *fakeDomains) Lookup(context.Context, string) (checker.DomainInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.info, f.err
}

// tlsService serves HTTPS with a certificate ending after the given time, and
// returns a checker that trusts it.
func tlsService(t *testing.T, validFor time.Duration) (string, *checker.Checker) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		Issuer:       pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(validFor),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},

		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	server.StartTLS()
	t.Cleanup(server.Close)

	roots := x509.NewCertPool()
	roots.AddCert(cert)
	return server.URL, checker.NewWithTransport(&http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{RootCAs: roots},
	})
}

func TestExpiringCertificateWarnsOnce(t *testing.T) {
	r := newRig(t)
	url, check := tlsService(t, 10*24*time.Hour+time.Hour)
	r.sched.Checker = check
	id := r.add(model.Config{Name: "api.app", URL: url})
	r.check(id, 3)
	r.worker.Flush(context.Background())

	got := r.view(id)
	if got.Status != "warn" || got.SSL.DaysLeft == nil || *got.SSL.DaysLeft != 10 || got.SSL.Issuer != "test" {
		t.Fatalf("%+v", got)
	}
	if got.ResponseMs == nil {
		t.Fatal("a warning monitor still answers")
	}
	if titles := r.sent.titles(); len(titles) != 1 || titles[0] != "api.app: SSL sertifikası bitiyor" {
		t.Fatalf("%v", titles)
	}
}

func TestHealthyCertificateStaysQuiet(t *testing.T) {
	r := newRig(t)
	url, check := tlsService(t, 90*24*time.Hour)
	r.sched.Checker = check
	id := r.add(model.Config{Name: "api.app", URL: url})
	r.check(id, 2)
	r.worker.Flush(context.Background())
	got := r.view(id)
	if got.Status != "up" || *got.SSL.DaysLeft != 89 && *got.SSL.DaysLeft != 90 {
		t.Fatalf("%+v %v", got.Status, got.SSL)
	}
	if titles := r.sent.titles(); len(titles) != 0 {
		t.Fatalf("%v", titles)
	}
}

func TestCertificateNearThirtyDaysNotifiesWithoutWarning(t *testing.T) {
	r := newRig(t)
	url, check := tlsService(t, 25*24*time.Hour)
	r.sched.Checker = check
	id := r.add(model.Config{Name: "api.app", URL: url})
	r.check(id, 1)
	r.worker.Flush(context.Background())
	if got := r.view(id); got.Status != "up" {
		t.Fatalf("status = %s", got.Status)
	}
	if titles := r.sent.titles(); len(titles) != 1 {
		t.Fatalf("%v", titles)
	}
}

func TestChangingTheAddressForgetsTheCertificate(t *testing.T) {
	r := newRig(t)
	secureURL, check := tlsService(t, 10*24*time.Hour)
	r.sched.Checker = check
	id := r.add(model.Config{Name: "api.app", URL: secureURL})
	r.check(id, 1)

	plainURL, _ := flaky(t)
	var cfg model.Config
	r.call("GET", "/api/monitors/"+id, nil, &cfg)
	cfg.URL = plainURL
	cfg.IntervalSec = 3600
	if code := r.call("PUT", "/api/monitors/"+id, cfg, nil); code != 200 {
		t.Fatalf("update: %d", code)
	}
	r.check(id, 1)
	if got := r.view(id); got.Status != "up" || got.SSL.DaysLeft != nil {
		t.Fatalf("%+v", got.SSL)
	}
}

func TestDomainExpiryWarnsAndSharesOneLookup(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)
	lookup := &fakeDomains{info: checker.DomainInfo{Domain: "example.com", Expires: time.Now().Add(5*24*time.Hour + time.Hour)}}
	r.sched.Domains = lookup
	id := r.add(model.Config{Name: "site", URL: url})
	r.check(id, 1)

	r.sched.RefreshDomains(context.Background())
	r.worker.Flush(context.Background())
	got := r.view(id)
	if got.Status != "warn" || got.Domain.DaysLeft == nil || *got.Domain.DaysLeft != 5 {
		t.Fatalf("%+v %+v", got.Status, got.Domain)
	}
	if titles := r.sent.titles(); len(titles) != 1 || titles[0] != "site: alan adı bitiyor" {
		t.Fatalf("%v", titles)
	}

	// A fresh answer is not looked up again.
	calls := lookup.calls
	r.sched.RefreshDomains(context.Background())
	if lookup.calls != calls {
		t.Fatal("looked up a fresh domain")
	}
}

func TestDomainLookupFailureKeepsNothingBroken(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)
	r.sched.Domains = &fakeDomains{err: errors.New("registry down")}
	id := r.add(model.Config{Name: "site", URL: url})
	r.check(id, 1)
	r.sched.RefreshDomains(context.Background())
	if got := r.view(id); got.Status != "up" || got.Domain.DaysLeft != nil {
		t.Fatalf("%+v", got)
	}
}

func TestUnknownDomainNeverAlarms(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)
	r.sched.Domains = &fakeDomains{info: checker.DomainInfo{Reason: checker.ReasonUnsupported}}
	id := r.add(model.Config{Name: "site", URL: url})
	r.check(id, 1)
	r.sched.RefreshDomains(context.Background())
	r.worker.Flush(context.Background())
	got := r.view(id)
	if got.Status != "up" || got.Domain.Reason != "unsupported" || len(r.sent.titles()) != 0 {
		t.Fatalf("%+v %v", got, r.sent.titles())
	}
}
