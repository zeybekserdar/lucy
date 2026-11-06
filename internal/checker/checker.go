// Package checker performs a single HTTP check and evaluates its conditions.
package checker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lucy/internal/model"
)

const maxBody = 1 << 20

// ConditionResult is structured so the UI can word it in its own language.
type ConditionResult struct {
	Type     string `json:"type"`
	Op       string `json:"op"`
	Path     string `json:"path,omitempty"`
	OK       bool   `json:"ok"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	// Reason is set when the value could not be compared:
	// no_response, not_json or path_missing.
	Reason string `json:"reason,omitempty"`
}

type Result struct {
	OK          bool              `json:"ok"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Status      int               `json:"status"`
	StatusText  string            `json:"statusText"`
	ResponseMs  int               `json:"responseMs"`
	ContentType string            `json:"contentType"`
	Error       string            `json:"error,omitempty"` // timeout, dns, refused, tls, network
	Conditions  []ConditionResult `json:"conditions"`
	At          time.Time         `json:"at"`
	// CertExpires is zero for plain HTTP and for failed handshakes.
	CertExpires time.Time `json:"-"`
	CertIssuer  string    `json:"-"`
}

// Network reports a failure that a lost internet connection could explain.
func (r Result) Network() bool {
	return r.Error != "" && r.Error != "tls"
}

// Failure is a language neutral cause: net:timeout, http:503 Service
// Unavailable or cond:body. The UI and notifications word it themselves.
func (r Result) Failure() string {
	if r.OK {
		return ""
	}
	if r.Error != "" {
		return "net:" + r.Error
	}
	for _, c := range r.Conditions {
		if !c.OK && c.Type == "status" {
			return "http:" + strings.TrimSpace(strconv.Itoa(r.Status)+" "+r.StatusText)
		}
	}
	for _, c := range r.Conditions {
		if !c.OK {
			return "cond:" + c.Type
		}
	}
	return "cond:unknown"
}

type Checker struct {
	client *http.Client
}

func New() *Checker {
	return NewWithTransport(&http.Transport{DisableKeepAlives: true})
}

func NewWithTransport(transport http.RoundTripper) *Checker {
	return &Checker{client: &http.Client{Transport: transport}}
}

func (c *Checker) Run(ctx context.Context, cfg model.Config) Result {
	start := time.Now()
	result := Result{Method: cfg.Method, URL: cfg.URL, At: start}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	var body io.Reader
	if cfg.Body != "" && (cfg.Method == "POST" || cfg.Method == "PUT") {
		body = strings.NewReader(cfg.Body)
	}
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, body)
	if err != nil {
		result.Error = "network"
		return finish(result, cfg, nil)
	}
	req.Header.Set("User-Agent", "Lucy/0.1")
	for _, h := range cfg.Headers {
		req.Header.Set(h.Name, h.Value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		result.Error = classify(err)
		result.ResponseMs = int(time.Since(start).Milliseconds())
		return finish(result, cfg, nil)
	}
	defer resp.Body.Close()
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		leaf := resp.TLS.PeerCertificates[0]
		result.CertExpires = leaf.NotAfter
		result.CertIssuer = leaf.Issuer.CommonName
		if len(leaf.Issuer.Organization) > 0 {
			result.CertIssuer = leaf.Issuer.Organization[0]
		}
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	result.ResponseMs = int(time.Since(start).Milliseconds())
	if err != nil {
		result.Error = classify(err)
		return finish(result, cfg, nil)
	}
	result.Status = resp.StatusCode
	result.StatusText = http.StatusText(resp.StatusCode)
	result.ContentType = strings.TrimSpace(strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0])
	return finish(result, cfg, data)
}

func finish(r Result, cfg model.Config, body []byte) Result {
	r.Conditions = make([]ConditionResult, 0, len(cfg.Conditions))
	r.OK = r.Error == ""
	for _, cond := range cfg.Conditions {
		outcome := evaluate(cond, r, body)
		r.OK = r.OK && outcome.OK
		r.Conditions = append(r.Conditions, outcome)
	}
	return r
}

func classify(err error) string {
	var dns *net.DNSError
	var cert *tls.CertificateVerificationError
	var unknownAuthority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	var record tls.RecordHeaderError
	switch {
	case errors.Is(err, context.DeadlineExceeded), isTimeout(err):
		return "timeout"
	case errors.As(err, &cert), errors.As(err, &unknownAuthority), errors.As(err, &hostname),
		errors.As(err, &invalid), errors.As(err, &record):
		return "tls"
	case errors.As(err, &dns):
		return "dns"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "refused"
	}
	return "network"
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
