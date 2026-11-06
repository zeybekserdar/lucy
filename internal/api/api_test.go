package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"lucy/internal/checker"
	"lucy/internal/model"
	"lucy/internal/notify"
	"lucy/internal/scheduler"
	"lucy/internal/store"
)

const testPassword = "doğru-at-pili-zımba"

type rig struct {
	t      *testing.T
	api    *httptest.Server
	client *http.Client
	sched  *scheduler.Scheduler
	worker notify.Worker
	guard  *checker.Guard
	sent   *inbox
}

type inbox struct {
	mu       sync.Mutex
	messages []map[string]any
	down     atomic.Bool
}

func (b *inbox) titles() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	titles := []string{}
	for _, m := range b.messages {
		titles = append(titles, m["title"].(string))
	}
	return titles
}

func newRig(t *testing.T, probeTargets ...string) *rig {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	db, err := store.Open(filepath.Join(t.TempDir(), "lucy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	sent := &inbox{}
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sent.down.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		var message map[string]any
		json.NewDecoder(r.Body).Decode(&message)
		sent.mu.Lock()
		sent.messages = append(sent.messages, message)
		sent.mu.Unlock()
	}))
	t.Cleanup(ntfy.Close)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	check := checker.New()
	guard := checker.NewGuard(probeTargets)
	sched := &scheduler.Scheduler{
		Store: db, Checker: check, Guard: guard, Log: log,
		Composer: &notify.Composer{Language: "tr", PublicURL: "http://lucy.home"},
	}
	if err := sched.Start(ctx); err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db, Scheduler: sched, Checker: check, Guard: guard, Log: log}
	api := httptest.NewServer(server.Handler())
	t.Cleanup(api.Close)
	r := &rig{
		t: t, api: api, guard: guard, sent: sent, client: newClient(t), sched: sched,
		worker: notify.Worker{Store: db, Notifier: notify.Ntfy{Server: ntfy.URL, Topic: "lucy"}, Log: log},
	}
	if code := r.call("POST", "/api/auth/setup", map[string]string{"password": testPassword}, nil); code != http.StatusNoContent {
		t.Fatalf("setup: %d", code)
	}
	return r
}

func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func (r *rig) call(method, path string, body any, out any) int {
	r.t.Helper()
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, r.api.URL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		r.t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			r.t.Fatal(err)
		}
	}
	return resp.StatusCode
}

func (r *rig) add(cfg model.Config) string {
	r.t.Helper()
	cfg.IntervalSec = 3600 // the test drives checks itself
	var created model.Config
	if code := r.call("POST", "/api/monitors", cfg, &created); code != http.StatusCreated {
		r.t.Fatalf("create: %d", code)
	}
	return created.ID
}

func (r *rig) check(id string, times int) {
	r.t.Helper()
	for range times {
		if code := r.call("POST", "/api/monitors/"+id+"/check", struct{}{}, nil); code != http.StatusOK {
			r.t.Fatalf("check: %d", code)
		}
	}
}

func (r *rig) view(id string) monitorView {
	r.t.Helper()
	var dashboard dashboardView
	r.call("GET", "/api/monitors", nil, &dashboard)
	for _, m := range dashboard.Monitors {
		if m.ID == id {
			return m
		}
	}
	r.t.Fatalf("monitor %s missing", id)
	return monitorView{}
}

// flaky is a service whose health the test controls.
func flaky(t *testing.T) (string, *atomic.Bool) {
	t.Helper()
	healthy := &atomic.Bool{}
	healthy.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !healthy.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL, healthy
}

func TestOutageLifecycleNotifiesOnceEachWay(t *testing.T) {
	r := newRig(t)
	url, healthy := flaky(t)
	id := r.add(model.Config{Name: "api.app", URL: url, AlertThreshold: 3})
	r.check(id, 1)
	if got := r.view(id); got.Status != "up" || got.ResponseMs == nil {
		t.Fatalf("%+v", got)
	}

	healthy.Store(false)
	r.check(id, 2)
	r.worker.Flush(context.Background())
	if got := r.view(id); got.Status != "up" || len(r.sent.titles()) != 0 {
		t.Fatalf("alarmed below the threshold: %+v %v", got, r.sent.titles())
	}

	r.check(id, 2)
	r.worker.Flush(context.Background())
	down := r.view(id)
	if down.Status != "down" || down.Failure != "http:503 Service Unavailable" || down.ResponseMs != nil {
		t.Fatalf("%+v", down)
	}
	if down.NotifiedAt == nil {
		t.Fatal("notifiedAt is missing")
	}
	if titles := r.sent.titles(); len(titles) != 1 || titles[0] != "api.app kesintide" {
		t.Fatalf("%v", titles)
	}
	if message := r.sent.messages[0]; message["click"] != "http://lucy.home/m/"+id ||
		!strings.Contains(message["message"].(string), "HTTP 503") {
		t.Fatalf("%v", message)
	}

	healthy.Store(true)
	r.check(id, 1)
	if got := r.view(id); got.Status != "down" {
		t.Fatalf("recovered too early: %s", got.Status)
	}
	r.check(id, 1)
	r.worker.Flush(context.Background())
	up := r.view(id)
	if up.Status != "up" || up.Failure != "" || up.LastIncident == nil {
		t.Fatalf("%+v", up)
	}
	if titles := r.sent.titles(); len(titles) != 2 || titles[1] != "api.app geri geldi" {
		t.Fatalf("%v", titles)
	}
	if up.Days[29] != "partial" && up.Days[29] != "down" {
		t.Fatalf("today = %s", up.Days[29])
	}
	if *up.Uptime.D1 >= 100 {
		t.Fatalf("uptime = %v", *up.Uptime.D1)
	}
}

func TestUnreachableNtfyKeepsTheMessage(t *testing.T) {
	r := newRig(t)
	url, healthy := flaky(t)
	id := r.add(model.Config{Name: "api.app", URL: url, AlertThreshold: 1})
	r.check(id, 1)
	healthy.Store(false)
	r.sent.down.Store(true)
	r.check(id, 1)
	r.worker.Flush(context.Background())
	if len(r.sent.titles()) != 0 {
		t.Fatal("unexpected delivery")
	}
	if got := r.view(id); got.NotifiedAt != nil {
		t.Fatal("marked as notified")
	}
	r.sent.down.Store(false)

	// Before the backoff elapses nothing is retried.
	r.worker.Flush(context.Background())
	if len(r.sent.titles()) != 0 {
		t.Fatal("retried before the backoff elapsed")
	}

	later := r.worker
	later.Now = func() time.Time { return time.Now().Add(notify.Backoff(0) + time.Second) }
	later.Flush(context.Background())
	if titles := r.sent.titles(); len(titles) != 1 || titles[0] != "api.app kesintide" {
		t.Fatalf("%v", titles)
	}
	if got := r.view(id); got.NotifiedAt == nil {
		t.Fatal("notifiedAt is missing after delivery")
	}
	later.Flush(context.Background())
	if len(r.sent.titles()) != 1 {
		t.Fatal("delivered twice")
	}
}

func closedPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

// An external address that fails at the network level while no probe target
// answers must not count as an outage.
func TestInternetOutageIsNotAnIncident(t *testing.T) {
	r := newRig(t, closedPort(t))
	external := "http://lucy-test.invalid/"
	id := r.add(model.Config{Name: "dış", URL: external, AlertThreshold: 1, TimeoutSec: 2})
	r.check(id, 3)
	r.worker.Flush(context.Background())

	got := r.view(id)
	if got.Status != "unknown" || len(r.sent.titles()) != 0 {
		t.Fatalf("%+v %v", got, r.sent.titles())
	}
	if got.Uptime.D1 != nil {
		t.Fatalf("unknown counted in uptime: %v", *got.Uptime.D1)
	}
	if got.Days[29] != "unknown" {
		t.Fatalf("today = %s", got.Days[29])
	}
	var dashboard dashboardView
	r.call("GET", "/api/monitors", nil, &dashboard)
	if dashboard.Network.Online || dashboard.Network.OfflineSince == nil {
		t.Fatalf("%+v", dashboard.Network)
	}
}

func TestLocalMonitorStillAlarmsDuringInternetOutage(t *testing.T) {
	r := newRig(t, closedPort(t))
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	id := r.add(model.Config{Name: "nas.home", URL: url, AlertThreshold: 1})
	r.check(id, 1)
	if got := r.view(id); got.Status != "down" || got.Failure != "net:refused" {
		t.Fatalf("%+v", got)
	}
}

func TestHalfHoursAndDaysWithoutDataAreNoData(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)
	id := r.add(model.Config{URL: url})
	r.check(id, 1)
	got := r.view(id)
	if len(got.Days) != 30 || got.Days[0] != "no_data" || got.Days[29] != "ok" {
		t.Fatalf("%v", got.Days)
	}
	if len(got.HalfHours) != 48 || got.HalfHours[0] != "no_data" || got.HalfHours[47] != "ok" {
		t.Fatalf("%v", got.HalfHours)
	}
}

func TestCrud(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)

	var invalid struct {
		Fields map[string]string `json:"fields"`
	}
	if code := r.call("POST", "/api/monitors", model.Config{URL: "nas.home:5001"}, &invalid); code != 422 {
		t.Fatalf("code = %d", code)
	}
	if invalid.Fields["url"] != "url_scheme" {
		t.Fatalf("%v", invalid.Fields)
	}

	id := r.add(model.Config{Name: "a", URL: url})
	var cfg model.Config
	r.call("GET", "/api/monitors/"+id, nil, &cfg)
	cfg.Name = "b"
	if code := r.call("PUT", "/api/monitors/"+id, cfg, nil); code != 200 {
		t.Fatalf("update: %d", code)
	}
	if got := r.view(id); got.Name != "b" {
		t.Fatalf("%+v", got)
	}
	if code := r.call("DELETE", "/api/monitors/"+id, nil, nil); code != 204 {
		t.Fatalf("delete: %d", code)
	}
	if code := r.call("GET", "/api/monitors/"+id, nil, nil); code != 404 {
		t.Fatalf("get: %d", code)
	}
}

func TestTestEndpointSavesNothing(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)
	var result checker.Result
	r.call("POST", "/api/monitors/test", model.Config{URL: url}, &result)
	if !result.OK || result.Status != 200 {
		t.Fatalf("%+v", result)
	}
	var dashboard dashboardView
	r.call("GET", "/api/monitors", nil, &dashboard)
	if len(dashboard.Monitors) != 0 {
		t.Fatal("test created a monitor")
	}
}

func TestWritesRequireJSON(t *testing.T) {
	r := newRig(t)
	resp, err := http.Post(r.api.URL+"/api/monitors/test", "text/plain", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("code = %d", resp.StatusCode)
	}
}

type historyResult struct {
	Monitor      monitorView `json:"monitor"`
	Points       []*int      `json:"points"`
	AvgMs        *int        `json:"avgMs"`
	Events       []event     `json:"events"`
	FailedChecks int         `json:"failedChecks"`
	LastResult   *checker.Result
}

func TestHistoryFollowsAnOutage(t *testing.T) {
	r := newRig(t)
	url, healthy := flaky(t)
	id := r.add(model.Config{Name: "a", URL: url, AlertThreshold: 2})
	r.check(id, 1)

	var history historyResult
	if code := r.call("GET", "/api/monitors/"+id+"/history?range=24h", nil, &history); code != http.StatusOK {
		t.Fatalf("history: %d", code)
	}
	if len(history.Points) != 96 || history.AvgMs == nil {
		t.Fatalf("series: %d points, average %v", len(history.Points), history.AvgMs)
	}
	if history.LastResult == nil || !history.LastResult.OK || len(history.LastResult.Conditions) == 0 {
		t.Fatalf("last result: %+v", history.LastResult)
	}
	if len(history.Events) != 0 {
		t.Fatalf("events: %+v", history.Events)
	}

	healthy.Store(false)
	r.check(id, 3)
	history = historyResult{}
	r.call("GET", "/api/monitors/"+id+"/history", nil, &history)
	if len(history.Points) != 168 {
		t.Fatalf("default range: %d points", len(history.Points))
	}
	if history.FailedChecks != 3 {
		t.Fatalf("failed checks: %d", history.FailedChecks)
	}
	if history.LastResult == nil || history.LastResult.OK || history.LastResult.Status != 503 {
		t.Fatalf("last result: %+v", history.LastResult)
	}
	if len(history.Events) != 1 || history.Events[0].Kind != "outage" || history.Events[0].End != nil ||
		history.Events[0].Cause != "http:503 Service Unavailable" {
		t.Fatalf("ongoing outage: %+v", history.Events)
	}

	healthy.Store(true)
	r.check(id, 2)
	history = historyResult{}
	r.call("GET", "/api/monitors/"+id+"/history", nil, &history)
	if len(history.Events) != 1 || history.Events[0].End == nil {
		t.Fatalf("finished outage: %+v", history.Events)
	}
	if history.FailedChecks != 0 {
		t.Fatalf("failed checks once up: %d", history.FailedChecks)
	}
}

func TestHistoryRejectsWhatItCannotServe(t *testing.T) {
	r := newRig(t)
	url, _ := flaky(t)
	id := r.add(model.Config{Name: "a", URL: url})
	if code := r.call("GET", "/api/monitors/"+id+"/history?range=1y", nil, nil); code != http.StatusBadRequest {
		t.Fatalf("bad range: %d", code)
	}
	if code := r.call("GET", "/api/monitors/nope/history", nil, nil); code != http.StatusNotFound {
		t.Fatalf("missing monitor: %d", code)
	}
	r.client = newClient(t)
	if code := r.call("GET", "/api/monitors/"+id+"/history", nil, nil); code != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", code)
	}
}

func TestShortSilenceIsNotAnEvent(t *testing.T) {
	if got := gapLimit(30); got != minGap {
		t.Fatalf("short interval: %v", got)
	}
	if got := gapLimit(3600); got != 3*time.Hour {
		t.Fatalf("long interval: %v", got)
	}
}
