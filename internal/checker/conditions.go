package checker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"lucy/internal/model"
)

func evaluate(cond model.Condition, r Result, body []byte) ConditionResult {
	out := ConditionResult{Type: cond.Type, Op: cond.Op, Path: cond.Path, Expected: cond.Value}
	if r.Error != "" {
		out.Reason = "no_response"
		return out
	}
	switch cond.Type {
	case "status":
		out.Actual = strconv.Itoa(r.Status)
		if cond.Op == "between" {
			low, high, ok := model.ParseStatusRange(cond.Value)
			out.OK = ok && r.Status >= low && r.Status <= high
			out.Expected = fmt.Sprintf("%d–%d", low, high)
		} else {
			want, err := strconv.Atoi(strings.TrimSpace(cond.Value))
			out.OK = err == nil && r.Status == want
		}
	case "response":
		limit, ok := model.ParseMs(cond.Value)
		out.Actual = strconv.Itoa(r.ResponseMs)
		out.Expected = strconv.Itoa(limit)
		out.OK = ok && r.ResponseMs < limit
	case "body":
		found := bytes.Contains(body, []byte(cond.Value))
		out.Actual = strconv.FormatBool(found)
		out.OK = found == (cond.Op == "contains")
	case "json":
		evaluateJSON(&out, cond, r, body)
	}
	return out
}

func evaluateJSON(out *ConditionResult, cond model.Condition, r Result, body []byte) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var doc any
	if err := decoder.Decode(&doc); err != nil {
		out.Reason = "not_json"
		out.Actual = r.ContentType
		return
	}
	value, ok := lookup(doc, cond.Path)
	if !ok {
		out.Reason = "path_missing"
		return
	}
	actual := stringify(value)
	want := strings.Trim(strings.TrimSpace(cond.Value), `"`)
	out.Actual = actual
	out.Expected = want
	if cond.Op == "contains" {
		out.OK = strings.Contains(actual, want)
	} else {
		out.OK = actual == want
	}
}

func stringify(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case nil:
		return "null"
	case map[string]any, []any:
		data, _ := json.Marshal(v)
		return string(data)
	}
	return fmt.Sprint(value)
}

// lookup resolves paths like $, $.a.b and $.items[0].name.
func lookup(doc any, path string) (any, bool) {
	rest := strings.TrimPrefix(path, "$")
	current := doc
	for rest != "" {
		switch rest[0] {
		case '.':
			rest = rest[1:]
			end := strings.IndexAny(rest, ".[")
			if end < 0 {
				end = len(rest)
			}
			object, ok := current.(map[string]any)
			if !ok || end == 0 {
				return nil, false
			}
			current, ok = object[rest[:end]]
			if !ok {
				return nil, false
			}
			rest = rest[end:]
		case '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return nil, false
			}
			index, err := strconv.Atoi(rest[1:end])
			list, ok := current.([]any)
			if err != nil || !ok || index < 0 || index >= len(list) {
				return nil, false
			}
			current = list[index]
			rest = rest[end+1:]
		default:
			return nil, false
		}
	}
	return current, true
}
