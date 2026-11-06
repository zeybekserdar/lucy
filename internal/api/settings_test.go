package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"lucy/internal/model"
	"lucy/internal/notify"
)

func fakeNtfy(t *testing.T, status int) (*httptest.Server, *atomic.Pointer[map[string]any], *atomic.Pointer[string]) {
	t.Helper()
	body := &atomic.Pointer[map[string]any]{}
	auth := &atomic.Pointer[string]{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message map[string]any
		json.NewDecoder(r.Body).Decode(&message)
		body.Store(&message)
		header := r.Header.Get("Authorization")
		auth.Store(&header)
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server, body, auth
}

func TestNtfySettingsPersist(t *testing.T) {
	r := newRig(t)
	var settings ntfyView
	r.call("GET", "/api/settings/ntfy", nil, &settings)
	if settings.Server != "https://ntfy.sh" || settings.Topic != "" {
		t.Fatalf("defaults: %+v", settings)
	}
	saved := ntfyView{Server: " https://n.test/ ", Topic: "lucy-1", Token: "tk"}
	if code := r.call("PUT", "/api/settings/ntfy", saved, nil); code != http.StatusNoContent {
		t.Fatalf("save: %d", code)
	}
	r.call("GET", "/api/settings/ntfy", nil, &settings)
	if settings != (ntfyView{Server: "https://n.test", Topic: "lucy-1", Token: "tk"}) {
		t.Fatalf("saved: %+v", settings)
	}
}

func TestNtfySettingsAreValidated(t *testing.T) {
	r := newRig(t)
	var failure struct {
		Fields map[string]string `json:"fields"`
	}
	code := r.call("PUT", "/api/settings/ntfy", ntfyView{Server: "ftp://x", Topic: "bad topic"}, &failure)
	if code != http.StatusUnprocessableEntity || failure.Fields["server"] == "" || failure.Fields["topic"] == "" {
		t.Fatalf("%d %+v", code, failure)
	}
}

func TestTestNotificationUsesTheValuesOnScreen(t *testing.T) {
	r := newRig(t)
	server, body, auth := fakeNtfy(t, http.StatusOK)
	var result struct {
		OK     bool `json:"ok"`
		Status int  `json:"status"`
	}
	r.call("POST", "/api/notify/test", ntfyView{Server: server.URL, Topic: "draft", Token: "tk"}, &result)
	if !result.OK || result.Status != 200 {
		t.Fatalf("result: %+v", result)
	}
	if (*body.Load())["topic"] != "draft" || *auth.Load() != "Bearer tk" {
		t.Fatalf("sent: %+v %q", *body.Load(), *auth.Load())
	}
	var settings ntfyView
	r.call("GET", "/api/settings/ntfy", nil, &settings)
	if settings.Topic != "" {
		t.Fatalf("a test must not save: %+v", settings)
	}
}

func TestTestNotificationReportsFailure(t *testing.T) {
	r := newRig(t)
	server, _, _ := fakeNtfy(t, http.StatusForbidden)
	var result struct {
		OK     bool `json:"ok"`
		Status int  `json:"status"`
	}
	r.call("POST", "/api/notify/test", ntfyView{Server: server.URL, Topic: "t"}, &result)
	if result.OK || result.Status != 403 {
		t.Fatalf("rejected: %+v", result)
	}
	server.Close()
	r.call("POST", "/api/notify/test", ntfyView{Server: server.URL, Topic: "t"}, &result)
	if result.OK || result.Status != 0 {
		t.Fatalf("unreachable: %+v", result)
	}
	if code := r.call("POST", "/api/notify/test", ntfyView{Server: server.URL}, nil); code != http.StatusUnprocessableEntity {
		t.Fatalf("no topic: %d", code)
	}
}

func (r *rig) export() (string, http.Header) {
	r.t.Helper()
	resp, err := r.client.Get(r.api.URL + "/api/backup.yaml")
	if err != nil {
		r.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		r.t.Fatalf("export: %d", resp.StatusCode)
	}
	return string(data), resp.Header
}

func TestBackupRestoresOnAnEmptyLucy(t *testing.T) {
	source := newRig(t)
	url, _ := flaky(t)
	source.add(model.Config{Name: "a", URL: url, Headers: []model.Header{{Name: "X-Key", Value: "v"}}})
	source.call("PUT", "/api/settings/ntfy", ntfyView{Server: "https://n.test", Topic: "t1", Token: "gizli-token"}, nil)
	data, headers := source.export()
	if strings.Contains(data, "gizli-token") {
		t.Fatalf("token exported:\n%s", data)
	}
	if !strings.Contains(headers.Get("Content-Disposition"), "attachment") {
		t.Fatalf("headers: %v", headers)
	}

	target := newRig(t)
	var preview importView
	if code := target.call("POST", "/api/backup/preview", map[string]string{"yaml": data}, &preview); code != http.StatusOK {
		t.Fatalf("preview: %d", code)
	}
	if len(preview.Add) != 1 || !preview.Notifications {
		t.Fatalf("preview: %+v", preview)
	}
	var dashboard dashboardView
	target.call("GET", "/api/monitors", nil, &dashboard)
	if len(dashboard.Monitors) != 0 {
		t.Fatal("a preview must change nothing")
	}

	if code := target.call("POST", "/api/backup/import", map[string]string{"yaml": data}, nil); code != http.StatusOK {
		t.Fatalf("import: %d", code)
	}
	target.call("GET", "/api/monitors", nil, &dashboard)
	if len(dashboard.Monitors) != 1 || dashboard.Monitors[0].URL != url || len(dashboard.Monitors[0].Headers) != 1 {
		t.Fatalf("restored: %+v", dashboard.Monitors)
	}
	var settings ntfyView
	target.call("GET", "/api/settings/ntfy", nil, &settings)
	if settings.Topic != "t1" || settings.Token != "" {
		t.Fatalf("settings: %+v", settings)
	}

	target.call("POST", "/api/backup/preview", map[string]string{"yaml": data}, &preview)
	if len(preview.Add)+len(preview.Update) != 0 || preview.Unchanged != 1 || preview.Notifications {
		t.Fatalf("importing twice must be a no-op: %+v", preview)
	}
}

func TestImportUpdatesInPlaceAndKeepsTheToken(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)
	id := r.add(model.Config{Name: "old", URL: url})
	r.check(id, 1)
	r.call("PUT", "/api/settings/ntfy", ntfyView{Server: "https://n.test", Topic: "t1", Token: "tk"}, nil)

	file := "version: 1\nnotifications: {server: https://n.test, topic: t2}\nmonitors:\n  - {name: new, url: " + url + "}\n"
	if code := r.call("POST", "/api/backup/import", map[string]string{"yaml": file}, nil); code != http.StatusOK {
		t.Fatalf("import: %d", code)
	}
	if view := r.view(id); view.Name != "new" || view.LastCheck == nil {
		t.Fatalf("history must stay with the monitor: %+v", view)
	}
	var settings ntfyView
	r.call("GET", "/api/settings/ntfy", nil, &settings)
	if settings.Topic != "t2" || settings.Token != "tk" {
		t.Fatalf("settings: %+v", settings)
	}
}

func TestImportIsAllOrNothing(t *testing.T) {
	r := newRig(t)
	file := "version: 1\nmonitors:\n  - {url: 'https://good.test'}\n  - {name: bad, url: 'ftp://bad.test'}\n"
	var failure struct {
		Error string `json:"error"`
	}
	if code := r.call("POST", "/api/backup/import", map[string]string{"yaml": file}, &failure); code != http.StatusUnprocessableEntity {
		t.Fatalf("import: %d", code)
	}
	var dashboard dashboardView
	r.call("GET", "/api/monitors", nil, &dashboard)
	if len(dashboard.Monitors) != 0 {
		t.Fatalf("nothing must be imported: %+v", dashboard.Monitors)
	}
	for yaml, want := range map[string]string{"nope: [": "bad_yaml", "version: 9\nmonitors: []\n": "bad_version"} {
		r.call("POST", "/api/backup/preview", map[string]string{"yaml": yaml}, &failure)
		if failure.Error != want {
			t.Errorf("%q: %q", yaml, failure.Error)
		}
	}
}

func TestSettingsNeedASession(t *testing.T) {
	r := newRig(t)
	r.client = newClient(t)
	for _, route := range [][2]string{
		{"GET", "/api/settings/ntfy"}, {"PUT", "/api/settings/ntfy"}, {"POST", "/api/notify/test"},
		{"GET", "/api/backup.yaml"}, {"POST", "/api/backup/preview"}, {"POST", "/api/backup/import"},
	} {
		if code := r.call(route[0], route[1], struct{}{}, nil); code != http.StatusUnauthorized {
			t.Errorf("%s: %d", route[1], code)
		}
	}
}

func TestPublicAddressReachesNotificationsWithoutARestart(t *testing.T) {
	r := newRig(t)
	r.sched.Link = notify.Live{Store: r.sched.Store}.PublicURL
	url, healthy := flaky(t)
	id := r.add(model.Config{Name: "a", URL: url, AlertThreshold: 1})

	saved := ntfyView{Server: "https://n.test", PublicURL: " http://192.168.1.9:8080/ "}
	if code := r.call("PUT", "/api/settings/ntfy", saved, nil); code != http.StatusNoContent {
		t.Fatalf("save: %d", code)
	}
	healthy.Store(false)
	r.check(id, 1)
	r.worker.Flush(context.Background())

	r.sent.mu.Lock()
	defer r.sent.mu.Unlock()
	if len(r.sent.messages) != 1 || r.sent.messages[0]["click"] != "http://192.168.1.9:8080/m/"+id {
		t.Fatalf("messages: %+v", r.sent.messages)
	}
}

func TestPublicAddressIsOptionalButMustBeAnAddress(t *testing.T) {
	r := newRig(t)
	if code := r.call("PUT", "/api/settings/ntfy", ntfyView{Server: "https://n.test"}, nil); code != http.StatusNoContent {
		t.Fatalf("empty: %d", code)
	}
	var failure struct {
		Fields map[string]string `json:"fields"`
	}
	code := r.call("PUT", "/api/settings/ntfy", ntfyView{Server: "https://n.test", PublicURL: "192.168.1.9"}, &failure)
	if code != http.StatusUnprocessableEntity || failure.Fields["publicUrl"] != "url_scheme" {
		t.Fatalf("%d %+v", code, failure)
	}
}

func TestTestNotificationOpensThePublicAddress(t *testing.T) {
	r := newRig(t)
	server, body, _ := fakeNtfy(t, http.StatusOK)
	r.call("POST", "/api/notify/test", ntfyView{Server: server.URL, Topic: "t", PublicURL: "http://192.168.1.9:8080/"}, nil)
	if (*body.Load())["click"] != "http://192.168.1.9:8080" {
		t.Fatalf("sent: %+v", *body.Load())
	}
}
