package model

import "testing"

func valid() Config {
	cfg := Config{URL: "https://foto.home/health"}
	cfg.Normalize()
	return cfg
}

func TestNormalizeFillsDefaults(t *testing.T) {
	cfg := valid()
	if cfg.Name != "foto.home" || cfg.Method != "GET" || cfg.TimeoutSec != 10 ||
		cfg.IntervalSec != 60 || cfg.AlertThreshold != 3 || len(cfg.Conditions) != 1 {
		t.Fatalf("%+v", cfg)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Fatal(errs)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*Config)
		field string
		code  string
	}{
		{"scheme", func(c *Config) { c.URL = "nas.home:5001" }, "url", "url_scheme"},
		{"timeout", func(c *Config) { c.TimeoutSec = 60 }, "timeoutSec", "timeout_range"},
		{"interval", func(c *Config) { c.IntervalSec = 10; c.TimeoutSec = 5 }, "intervalSec", "interval_min"},
		{"threshold", func(c *Config) { c.AlertThreshold = 11 }, "alertThreshold", "threshold_range"},
		{"json path", func(c *Config) {
			c.Conditions = []Condition{{Type: "json", Op: "equals", Path: "status", Value: "ok"}}
		}, "conditions.0", "json_path"},
		{"status", func(c *Config) {
			c.Conditions = []Condition{{Type: "status", Op: "between", Value: "299-200"}}
		}, "conditions.0", "status_value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := valid()
			c.edit(&cfg)
			if got := cfg.Validate()[c.field]; got != c.code {
				t.Fatalf("%s = %q, want %q", c.field, got, c.code)
			}
		})
	}
}
