// Package yamlio turns monitors into a backup file and plans restoring one.
package yamlio

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"

	"lucy/internal/model"
)

const Version = 1

// ErrVersion means the file comes from a Lucy this one cannot read.
var ErrVersion = errors.New("unsupported backup version")

type Notifications struct {
	Server    string `yaml:"server,omitempty"`
	Topic     string `yaml:"topic,omitempty"`
	PublicURL string `yaml:"publicUrl,omitempty"`
}

// Backup never holds the ntfy token, so the file can be kept anywhere.
type Backup struct {
	Version       int            `yaml:"version"`
	Notifications *Notifications `yaml:"notifications,omitempty"`
	Monitors      []model.Config `yaml:"monitors"`
}

func Encode(b Backup) ([]byte, error) {
	b.Version = Version
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(b); err != nil {
		return nil, err
	}
	return out.Bytes(), encoder.Close()
}

// Decode rejects unknown keys: a misspelt field must not vanish silently.
func Decode(data []byte) (Backup, error) {
	var b Backup
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&b); err != nil {
		return b, err
	}
	if b.Version != Version {
		return b, fmt.Errorf("%w: %d", ErrVersion, b.Version)
	}
	return b, nil
}

// Invalid is an entry that cannot be imported.
type Invalid struct {
	Name   string            `json:"name"`
	Fields map[string]string `json:"fields"`
}

// Plan is what importing a file would do; monitors match by address.
type Plan struct {
	Add       []model.Config
	Update    []model.Config // ID is the monitor to overwrite
	Unchanged []model.Config
	Invalid   []Invalid
}

func NewPlan(existing, incoming []model.Config) Plan {
	byURL := make(map[string]model.Config, len(existing))
	for _, cfg := range existing {
		if _, seen := byURL[cfg.URL]; !seen {
			byURL[cfg.URL] = cfg
		}
	}
	var plan Plan
	seen := map[string]bool{}
	for _, cfg := range incoming {
		cfg.ID = ""
		cfg.Normalize()
		fields := cfg.Validate()
		if seen[cfg.URL] {
			fields["url"] = "duplicate"
		}
		seen[cfg.URL] = true
		if len(fields) > 0 {
			name := cfg.Name
			if name == "" {
				name = cfg.URL
			}
			plan.Invalid = append(plan.Invalid, Invalid{Name: name, Fields: fields})
			continue
		}
		current, found := byURL[cfg.URL]
		if !found {
			plan.Add = append(plan.Add, cfg)
			continue
		}
		cfg.ID = current.ID
		current.Normalize()
		if reflect.DeepEqual(cfg, current) {
			plan.Unchanged = append(plan.Unchanged, cfg)
		} else {
			plan.Update = append(plan.Update, cfg)
		}
	}
	return plan
}
