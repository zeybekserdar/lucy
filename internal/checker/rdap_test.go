package checker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func rdapServer(t *testing.T, hits *[]string) *Domains {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/dns.json", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"services": [][][]string{
			{{"com", "uk"}, {server.URL + "/rdap/"}},
		}})
	})
	mux.HandleFunc("/rdap/domain/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		*hits = append(*hits, name)
		switch name {
		case "missing.com":
			w.WriteHeader(http.StatusNotFound)
		case "broken.com":
			w.WriteHeader(http.StatusBadGateway)
		case "undated.com":
			w.Write([]byte(`{"events":[{"eventAction":"registration","eventDate":"2020-01-01T00:00:00Z"}]}`))
		default:
			w.Write([]byte(`{"events":[{"eventAction":"registration","eventDate":"2020-01-01T00:00:00Z"},
				{"eventAction":"expiration","eventDate":"2030-05-06T07:08:09Z"}]}`))
		}
	})
	return &Domains{Bootstrap: server.URL + "/dns.json", Client: server.Client()}
}

func TestLookup(t *testing.T) {
	var hits []string
	domains := rdapServer(t, &hits)
	ctx := context.Background()

	info, err := domains.Lookup(ctx, "https://app.sub.example.co.uk/health")
	if err != nil || info.Domain != "example.co.uk" || info.Reason != "" {
		t.Fatalf("%+v %v", info, err)
	}
	if want := time.Date(2030, 5, 6, 7, 8, 9, 0, time.UTC); !info.Expires.Equal(want) {
		t.Fatalf("expires = %v", info.Expires)
	}

	cases := map[string]string{
		"https://missing.com":     ReasonNotFound,
		"https://undated.com":     ReasonNoExpiry,
		"https://nobody.dev":      ReasonUnsupported,
		"https://nas.home:5001":   ReasonLocal,
		"http://192.168.1.10":     ReasonLocal,
		"https://93.184.216.34/x": ReasonLocal,
	}
	for url, want := range cases {
		info, err := domains.Lookup(ctx, url)
		if err != nil || info.Reason != want {
			t.Errorf("%s: %+v %v", url, info, err)
		}
	}
}

func TestLookupTemporaryFailureIsAnError(t *testing.T) {
	var hits []string
	domains := rdapServer(t, &hits)
	if _, err := domains.Lookup(context.Background(), "https://broken.com"); err == nil {
		t.Fatal("a 502 must not look like a settled answer")
	}
}

func TestRegistrable(t *testing.T) {
	cases := map[string]string{
		"https://example.com":         "example.com",
		"https://a.b.example.com/x":   "example.com",
		"https://www.bbc.co.uk":       "bbc.co.uk",
		"https://www.trendyol.com.tr": "trendyol.com.tr",
		"https://github.io":           "github.io",
		"https://me.github.io":        "github.io",
		"https://uygulama.vercel.app": "vercel.app",
		"https://x.y.pages.dev":       "pages.dev",
		"https://api.app":             "api.app",
		"https://sunucu.unknowntld":   "sunucu.unknowntld",
	}
	for url, want := range cases {
		if got, ok := Registrable(url); !ok || got != want {
			t.Errorf("%s = %q %v, want %q", url, got, ok, want)
		}
	}
	for _, url := range []string{"https://com", "http://8.8.8.8", "https://nas.home", "http://localhost:3000"} {
		if got, ok := Registrable(url); ok {
			t.Errorf("%s = %q", url, got)
		}
	}
}
