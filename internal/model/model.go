// Package model holds the monitor configuration shared by every layer.
package model

import (
	"net/url"
	"strconv"
	"strings"
)

type Header struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

// Condition is one rule a response must satisfy.
//
//	status:   op between (value "200-299") or equals
//	response: op below (value in ms)
//	body:     op contains or not_contains
//	json:     op equals or contains, with path like $.a.b[0]
type Condition struct {
	Type  string `json:"type" yaml:"type"`
	Op    string `json:"op" yaml:"op"`
	Path  string `json:"path,omitempty" yaml:"path,omitempty"`
	Value string `json:"value" yaml:"value"`
}

type Config struct {
	ID             string      `json:"id" yaml:"-"`
	Name           string      `json:"name" yaml:"name"`
	URL            string      `json:"url" yaml:"url"`
	Method         string      `json:"method" yaml:"method,omitempty"`
	Headers        []Header    `json:"headers" yaml:"headers,omitempty"`
	Body           string      `json:"body" yaml:"body,omitempty"`
	TimeoutSec     int         `json:"timeoutSec" yaml:"timeoutSec,omitempty"`
	IntervalSec    int         `json:"intervalSec" yaml:"intervalSec,omitempty"`
	AlertThreshold int         `json:"alertThreshold" yaml:"alertThreshold,omitempty"`
	Conditions     []Condition `json:"conditions" yaml:"conditions,omitempty"`
}

const MinIntervalSec = 30

// DefaultConditions apply when a monitor defines none.
func DefaultConditions() []Condition {
	return []Condition{{Type: "status", Op: "between", Value: "200-299"}}
}

// Normalize trims input and fills in defaults.
func (c *Config) Normalize() {
	c.URL = strings.TrimSpace(c.URL)
	c.Name = strings.TrimSpace(c.Name)
	c.Method = strings.ToUpper(strings.TrimSpace(c.Method))
	if c.Method == "" {
		c.Method = "GET"
	}
	if c.TimeoutSec == 0 {
		c.TimeoutSec = 10
	}
	if c.IntervalSec == 0 {
		c.IntervalSec = 60
	}
	if c.AlertThreshold == 0 {
		c.AlertThreshold = 3
	}
	headers := make([]Header, 0, len(c.Headers))
	for _, h := range c.Headers {
		h.Name = strings.TrimSpace(h.Name)
		if h.Name != "" {
			headers = append(headers, h)
		}
	}
	c.Headers = headers
	if len(c.Conditions) == 0 {
		c.Conditions = DefaultConditions()
	}
	if c.Name == "" {
		if u, err := url.Parse(c.URL); err == nil {
			c.Name = u.Host
		}
	}
}

// Validate returns a field to error code map; empty means valid.
func (c *Config) Validate() map[string]string {
	errs := map[string]string{}
	u, err := url.Parse(c.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs["url"] = "url_scheme"
	}
	switch c.Method {
	case "GET", "POST", "PUT", "HEAD":
	default:
		errs["method"] = "method"
	}
	if c.IntervalSec < MinIntervalSec {
		errs["intervalSec"] = "interval_min"
	}
	if c.TimeoutSec < 1 || c.TimeoutSec >= c.IntervalSec {
		errs["timeoutSec"] = "timeout_range"
	}
	if c.AlertThreshold < 1 || c.AlertThreshold > 10 {
		errs["alertThreshold"] = "threshold_range"
	}
	for i, cond := range c.Conditions {
		if code := cond.validate(); code != "" {
			errs["conditions."+strconv.Itoa(i)] = code
		}
	}
	return errs
}

func (c Condition) validate() string {
	switch c.Type {
	case "status":
		if c.Op == "between" {
			if _, _, ok := ParseStatusRange(c.Value); !ok {
				return "status_value"
			}
			return ""
		}
		if c.Op == "equals" {
			if _, err := strconv.Atoi(strings.TrimSpace(c.Value)); err != nil {
				return "status_value"
			}
			return ""
		}
	case "response":
		if c.Op == "below" {
			if _, ok := ParseMs(c.Value); !ok {
				return "response_value"
			}
			return ""
		}
	case "body":
		if c.Op == "contains" || c.Op == "not_contains" {
			if c.Value == "" {
				return "value_required"
			}
			return ""
		}
	case "json":
		if c.Op == "equals" || c.Op == "contains" {
			if !strings.HasPrefix(c.Path, "$") {
				return "json_path"
			}
			return ""
		}
	}
	return "condition"
}

// ParseStatusRange reads "200-299" or "200–299".
func ParseStatusRange(value string) (low, high int, ok bool) {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '-' || r == '–' })
	if len(parts) != 2 {
		return 0, 0, false
	}
	low, errLow := strconv.Atoi(strings.TrimSpace(parts[0]))
	high, errHigh := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errLow != nil || errHigh != nil || low > high {
		return 0, 0, false
	}
	return low, high, true
}

// ParseMs reads "2000" or "2000 ms".
func ParseMs(value string) (int, bool) {
	text := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "ms"))
	ms, err := strconv.Atoi(text)
	return ms, err == nil && ms > 0
}
