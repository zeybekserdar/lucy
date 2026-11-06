package yamlio

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"lucy/internal/model"
)

func config(name, url string) model.Config {
	cfg := model.Config{Name: name, URL: url}
	cfg.Normalize()
	return cfg
}

func TestRoundTripKeepsEveryField(t *testing.T) {
	want := model.Config{
		Name: "api", URL: "https://a.test/health", Method: "POST", Body: `{"a":1}`,
		Headers:    []model.Header{{Name: "X-Key", Value: "değer"}},
		TimeoutSec: 5, IntervalSec: 120, AlertThreshold: 4,
		Conditions: []model.Condition{
			{Type: "status", Op: "between", Value: "200-299"},
			{Type: "json", Op: "equals", Path: "$.status", Value: "ok"},
		},
	}
	data, err := Encode(Backup{Monitors: []model.Config{want}, Notifications: &Notifications{Server: "https://n.test", Topic: "t"}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Monitors[0], want) {
		t.Fatalf("got %+v\nwant %+v", got.Monitors[0], want)
	}
	if got.Notifications.Topic != "t" {
		t.Fatalf("notifications: %+v", got.Notifications)
	}
}

func TestEncodeLeavesOutTheID(t *testing.T) {
	cfg := config("a", "https://a.test")
	cfg.ID = "secret-id"
	data, err := Encode(Backup{Monitors: []model.Config{cfg}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-id") {
		t.Fatalf("id exported:\n%s", data)
	}
}

func TestDecodeAcceptsAHandWrittenFile(t *testing.T) {
	b, err := Decode([]byte("version: 1\nmonitors:\n  - url: https://a.test\n    conditions:\n      - {type: status, op: equals, value: 200}\n"))
	if err != nil {
		t.Fatal(err)
	}
	plan := NewPlan(nil, b.Monitors)
	if len(plan.Invalid) != 0 || len(plan.Add) != 1 {
		t.Fatalf("plan: %+v", plan)
	}
	added := plan.Add[0]
	if added.Name != "a.test" || added.IntervalSec != 60 || added.Conditions[0].Value != "200" {
		t.Fatalf("defaults: %+v", added)
	}
}

func TestDecodeRejects(t *testing.T) {
	cases := map[string]string{
		"not yaml":     "version: [",
		"typo":         "version: 1\nmonitors:\n  - url: https://a.test\n    intervalSecs: 30\n",
		"not a backup": "hello: world\n",
	}
	for name, input := range cases {
		if _, err := Decode([]byte(input)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := Decode([]byte("version: 2\nmonitors: []\n")); !errors.Is(err, ErrVersion) {
		t.Errorf("version: %v", err)
	}
}

func TestPlanMatchesByAddress(t *testing.T) {
	same := config("same", "https://same.test")
	same.ID = "1"
	changed := config("old name", "https://changed.test")
	changed.ID = "2"

	plan := NewPlan([]model.Config{same, changed}, []model.Config{
		config("same", "https://same.test"),
		config("new name", "https://changed.test"),
		config("fresh", "https://fresh.test"),
	})
	if len(plan.Unchanged) != 1 || len(plan.Update) != 1 || len(plan.Add) != 1 || len(plan.Invalid) != 0 {
		t.Fatalf("plan: %+v", plan)
	}
	if plan.Update[0].ID != "2" || plan.Update[0].Name != "new name" {
		t.Fatalf("update: %+v", plan.Update[0])
	}
	if plan.Add[0].ID != "" {
		t.Fatalf("add keeps no id: %+v", plan.Add[0])
	}
}

func TestPlanReportsBadEntries(t *testing.T) {
	plan := NewPlan(nil, []model.Config{
		{Name: "bad", URL: "ftp://a.test"},
		{URL: "https://twice.test"},
		{URL: "https://twice.test"},
	})
	if len(plan.Invalid) != 2 || plan.Invalid[0].Fields["url"] != "url_scheme" || plan.Invalid[1].Fields["url"] != "duplicate" {
		t.Fatalf("invalid: %+v", plan.Invalid)
	}
}
