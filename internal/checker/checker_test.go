package checker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lucy/internal/model"
)

func config(url string, conditions ...model.Condition) model.Config {
	cfg := model.Config{URL: url, Conditions: conditions}
	cfg.Normalize()
	return cfg
}

func serve(t *testing.T, status int, contentType, body string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestDefaultConditionAccepts2xx(t *testing.T) {
	result := New().Run(context.Background(), config(serve(t, 204, "text/plain", "")))
	if !result.OK || result.Status != 204 {
		t.Fatalf("%+v", result)
	}
}

func TestFailureNamesTheHTTPStatus(t *testing.T) {
	result := New().Run(context.Background(), config(serve(t, 503, "text/html", "")))
	if result.OK || result.Failure() != "http:503 Service Unavailable" || result.Network() {
		t.Fatalf("%+v failure=%q", result, result.Failure())
	}
}

func TestConditions(t *testing.T) {
	const body = `{"status":"ok","count":3,"items":[{"name":"a"}],"up":true}`
	cases := []struct {
		name string
		cond model.Condition
		ok   bool
	}{
		{"status equals", model.Condition{Type: "status", Op: "equals", Value: "200"}, true},
		{"status range en dash", model.Condition{Type: "status", Op: "between", Value: "200–299"}, true},
		{"status outside", model.Condition{Type: "status", Op: "between", Value: "300-399"}, false},
		{"response below", model.Condition{Type: "response", Op: "below", Value: "5000 ms"}, true},
		{"body contains", model.Condition{Type: "body", Op: "contains", Value: `"count":3`}, true},
		{"body lacks", model.Condition{Type: "body", Op: "contains", Value: "pong"}, false},
		{"body not contains", model.Condition{Type: "body", Op: "not_contains", Value: "pong"}, true},
		{"json quoted string", model.Condition{Type: "json", Op: "equals", Path: "$.status", Value: `"ok"`}, true},
		{"json number", model.Condition{Type: "json", Op: "equals", Path: "$.count", Value: "3"}, true},
		{"json bool", model.Condition{Type: "json", Op: "equals", Path: "$.up", Value: "true"}, true},
		{"json index", model.Condition{Type: "json", Op: "equals", Path: "$.items[0].name", Value: "a"}, true},
		{"json contains", model.Condition{Type: "json", Op: "contains", Path: "$.status", Value: "o"}, true},
		{"json mismatch", model.Condition{Type: "json", Op: "equals", Path: "$.status", Value: "bad"}, false},
		{"json missing", model.Condition{Type: "json", Op: "equals", Path: "$.nope", Value: "x"}, false},
		{"json index out of range", model.Condition{Type: "json", Op: "equals", Path: "$.items[5]", Value: "x"}, false},
	}
	url := serve(t, 200, "application/json", body)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := New().Run(context.Background(), config(url, c.cond))
			if result.OK != c.ok {
				t.Fatalf("OK = %v: %+v", result.OK, result.Conditions)
			}
		})
	}
}

func TestNonJSONBodyIsReported(t *testing.T) {
	cond := model.Condition{Type: "json", Op: "equals", Path: "$.a", Value: "1"}
	result := New().Run(context.Background(), config(serve(t, 200, "text/html", "<html>"), cond))
	if got := result.Conditions[0]; got.Reason != "not_json" || got.Actual != "text/html" {
		t.Fatalf("%+v", got)
	}
	if result.Failure() != "cond:json" {
		t.Fatalf("failure = %q", result.Failure())
	}
}

func TestTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })
	cfg := config(server.URL)
	cfg.TimeoutSec = 1
	result := New().Run(context.Background(), cfg)
	if result.Error != "timeout" || !result.Network() {
		t.Fatalf("%+v", result)
	}
}

func TestRefusedConnection(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	result := New().Run(context.Background(), config(url))
	if result.Error != "refused" || result.Failure() != "net:refused" {
		t.Fatalf("%+v", result)
	}
	if got := result.Conditions[0]; got.Reason != "no_response" {
		t.Fatalf("%+v", got)
	}
}

func TestBadCertificateIsNotAConnectivityProblem(t *testing.T) {
	server := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(server.Close)
	result := New().Run(context.Background(), config(server.URL))
	if result.Error != "tls" || result.Network() {
		t.Fatalf("%+v", result)
	}
}

func TestRequestCarriesMethodHeadersAndBody(t *testing.T) {
	var method, header, body string
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		method, header, body = r.Method, r.Header.Get("X-Key"), string(data)
	}))
	t.Cleanup(server.Close)
	cfg := config(server.URL)
	cfg.Method, cfg.Body = "POST", "ping"
	cfg.Headers = []model.Header{{Name: "X-Key", Value: "v"}}
	New().Run(context.Background(), cfg)
	if method != "POST" || header != "v" || body != "ping" {
		t.Fatalf("%q %q %q", method, header, body)
	}
}

func TestIsLocal(t *testing.T) {
	cases := map[string]bool{
		"https://nas.home:5001":    true,
		"http://192.168.1.10":      true,
		"http://127.0.0.1:9":       true,
		"http://printer":           true,
		"https://x.lan/health":     true,
		"https://api.app/health":   false,
		"https://8.8.8.8":          false,
		"https://home.example.com": false,
	}
	for url, want := range cases {
		if got := IsLocal(url); got != want {
			t.Errorf("IsLocal(%q) = %v", url, got)
		}
	}
}

func TestGuardTracksOutages(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	target := server.Listener.Addr().String()
	guard := NewGuard([]string{target})
	guard.ttl = 0
	if !guard.Online(context.Background()) {
		t.Fatal("expected online")
	}
	server.Close()
	before := time.Now()
	if guard.Online(context.Background()) {
		t.Fatal("expected offline")
	}
	online, since := guard.State()
	if online || since.Before(before) {
		t.Fatalf("state = %v %v", online, since)
	}
	guard.MarkOnline()
	if online, since := guard.State(); !online || !since.IsZero() {
		t.Fatalf("state = %v %v", online, since)
	}
}
