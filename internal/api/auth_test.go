package api

import (
	"net/http"
	neturl "net/url"
	"strings"
	"testing"

	"lucy/internal/model"
)

func (r *rig) as(client *http.Client) *rig {
	other := *r
	other.client = client
	return &other
}

func (r *rig) status() string {
	r.t.Helper()
	var body authStatus
	r.call("GET", "/api/auth/status", nil, &body)
	return body.Status
}

func TestFreshServerNeedsSetupOnce(t *testing.T) {
	r := newRig(t)
	stranger := r.as(newClient(t))
	if got := stranger.status(); got != "loggedOut" {
		t.Fatalf("status = %s", got)
	}
	if got := r.status(); got != "loggedIn" {
		t.Fatalf("status = %s", got)
	}
	code := stranger.call("POST", "/api/auth/setup", map[string]string{"password": "başkasının-şifresi"}, nil)
	if code != http.StatusConflict {
		t.Fatalf("second setup: %d", code)
	}
	if code := stranger.call("POST", "/api/auth/login", map[string]string{"password": "başkasının-şifresi"}, nil); code != 401 {
		t.Fatalf("setup overwrote the password: %d", code)
	}
}

func TestEveryProtectedRouteRejectsStrangers(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)
	id := r.add(modelFor(url))
	stranger := r.as(newClient(t))
	routes := [][2]string{
		{"GET", "/api/monitors"},
		{"POST", "/api/monitors"},
		{"POST", "/api/monitors/test"},
		{"GET", "/api/monitors/" + id},
		{"PUT", "/api/monitors/" + id},
		{"DELETE", "/api/monitors/" + id},
		{"POST", "/api/monitors/" + id + "/check"},
		{"POST", "/api/auth/password"},
	}
	for _, route := range routes {
		if code := stranger.call(route[0], route[1], struct{}{}, nil); code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d", route[0], route[1], code)
		}
	}
}

func TestForgedCookieIsRejected(t *testing.T) {
	r := newRig(t)
	req, _ := http.NewRequest("GET", r.api.URL+"/api/monitors", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: strings.Repeat("a", 64)})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("code = %d", resp.StatusCode)
	}
}

func TestSessionCookieFlags(t *testing.T) {
	r := newRig(t)
	req, _ := http.NewRequest("POST", r.api.URL+"/api/auth/login", strings.NewReader(`{"password":"`+testPassword+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %v", cookies)
	}
	c := cookies[0]
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("%+v", c)
	}
}

func TestLogoutEndsTheSession(t *testing.T) {
	r := newRig(t)
	kept := r.client.Jar.Cookies(mustURL(t, r.api.URL))
	if code := r.call("POST", "/api/auth/logout", struct{}{}, nil); code != http.StatusNoContent {
		t.Fatalf("logout: %d", code)
	}
	if got := r.status(); got != "loggedOut" {
		t.Fatalf("status = %s", got)
	}
	// Replaying the old cookie must fail too, not only the browser forgetting it.
	replay := r.as(newClient(t))
	replay.client.Jar.SetCookies(mustURL(t, r.api.URL), kept)
	if got := replay.status(); got != "loggedOut" {
		t.Fatalf("replayed cookie: %s", got)
	}
}

func TestLoginIsThrottled(t *testing.T) {
	r := newRig(t)
	stranger := r.as(newClient(t))
	wrong := map[string]string{"password": "yanlış-şifre-burada"}
	for i := range 5 {
		if code := stranger.call("POST", "/api/auth/login", wrong, nil); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d", i, code)
		}
	}
	var body struct {
		Error         string `json:"error"`
		RetryAfterSec int    `json:"retryAfterSec"`
	}
	code := stranger.call("POST", "/api/auth/login", map[string]string{"password": testPassword}, &body)
	if code != http.StatusTooManyRequests || body.RetryAfterSec < 1 {
		t.Fatalf("%d %+v", code, body)
	}
}

func TestChangePassword(t *testing.T) {
	r := newRig(t)
	other := r.as(newClient(t))
	if code := other.call("POST", "/api/auth/login", map[string]string{"password": testPassword}, nil); code != 204 {
		t.Fatalf("login: %d", code)
	}

	var invalid struct {
		Fields map[string]string `json:"fields"`
	}
	const next = "yepyeni-uzun-şifre"
	code := r.call("POST", "/api/auth/password", map[string]string{"current": "yanlış", "next": next}, &invalid)
	if code != 422 || invalid.Fields["current"] != "wrong_password" {
		t.Fatalf("%d %v", code, invalid.Fields)
	}
	code = r.call("POST", "/api/auth/password", map[string]string{"current": testPassword, "next": "kısa"}, &invalid)
	if code != 422 || invalid.Fields["next"] != "password_short" {
		t.Fatalf("%d %v", code, invalid.Fields)
	}
	if code := r.call("POST", "/api/auth/password", map[string]string{"current": testPassword, "next": next}, nil); code != 204 {
		t.Fatalf("change: %d", code)
	}
	if got := r.status(); got != "loggedIn" {
		t.Fatalf("this session ended: %s", got)
	}
	if got := other.status(); got != "loggedOut" {
		t.Fatalf("other session survived: %s", got)
	}
	if code := other.call("POST", "/api/auth/login", map[string]string{"password": testPassword}, nil); code != 401 {
		t.Fatalf("old password works: %d", code)
	}
	if code := other.call("POST", "/api/auth/login", map[string]string{"password": next}, nil); code != 204 {
		t.Fatalf("new password: %d", code)
	}
}

func modelFor(url string) model.Config { return model.Config{URL: url} }

func mustURL(t *testing.T, raw string) *neturl.URL {
	t.Helper()
	u, err := neturl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
