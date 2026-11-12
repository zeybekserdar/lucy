package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lucy/internal/store"
)

func TestDownPriorityAndActions(t *testing.T) {
	c := Composer{Language: "tr", PublicURL: "https://lucy.home/"}
	n := c.Down("m1", "api.app", "https://api.app/health", "http:503 Service Unavailable", false)
	if n.Priority != 4 || len(n.Tags) != 0 {
		t.Fatalf("normal down: %+v", n)
	}
	want := []store.Action{
		{Label: "Lucy'de aç", URL: "https://lucy.home/m/m1"},
		{Label: "Siteyi aç", URL: "https://api.app/health"},
	}
	if len(n.Actions) != 2 || n.Actions[0] != want[0] || n.Actions[1] != want[1] {
		t.Fatalf("actions: %+v", n.Actions)
	}
	if urgent := c.Down("m1", "api.app", "https://api.app", "net:timeout", true); urgent.Priority != 5 {
		t.Fatalf("urgent priority: %d", urgent.Priority)
	}
}

func TestActionsWithoutLucyAddress(t *testing.T) {
	n := Composer{Language: "en"}.Recovered("m1", "api.app", "https://api.app", 0)
	if len(n.Actions) != 1 || n.Actions[0].Label != "Open site" {
		t.Fatalf("actions: %+v", n.Actions)
	}
}

func TestPublishSendsIconAndActions(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
	}))
	defer server.Close()
	msg := Composer{Language: "tr", PublicURL: "https://lucy.home"}.Down("m1", "a", "https://a.app", "net:dns", false)
	target := Ntfy{Server: server.URL, Topic: "t", Icon: IconURL("https://lucy.home/")}
	if _, err := target.Publish(context.Background(), *msg); err != nil {
		t.Fatal(err)
	}
	if got["icon"] != "https://lucy.home/icon.png" {
		t.Fatalf("icon: %v", got["icon"])
	}
	if _, ok := got["tags"]; ok {
		t.Fatalf("tags sent: %v", got["tags"])
	}
	actions, _ := got["actions"].([]any)
	first, _ := actions[0].(map[string]any)
	if len(actions) != 2 || first["action"] != "view" || first["url"] != "https://lucy.home/m/m1" {
		t.Fatalf("actions: %v", got["actions"])
	}
}
