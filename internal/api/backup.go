package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"lucy/internal/model"
	"lucy/internal/store"
	"lucy/internal/yamlio"
)

func (s *Server) exportBackup(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.Store.ListMonitors(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	settings, err := s.Store.Ntfy(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	backup := yamlio.Backup{Monitors: make([]model.Config, 0, len(monitors))}
	for _, m := range monitors {
		backup.Monitors = append(backup.Monitors, m.Config)
	}
	if settings.Topic != "" {
		backup.Notifications = &yamlio.Notifications{Server: settings.Server, Topic: settings.Topic}
	}
	data, err := yamlio.Encode(backup)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="lucy-%s.yaml"`, time.Now().Format("2006-01-02")))
	w.Write(data)
}

type importView struct {
	Add           []string         `json:"add"`
	Update        []string         `json:"update"`
	Unchanged     int              `json:"unchanged"`
	Invalid       []yamlio.Invalid `json:"invalid"`
	Notifications bool             `json:"notifications"` // the file would change where messages go
}

func names(configs []model.Config) []string {
	list := make([]string, 0, len(configs))
	for _, cfg := range configs {
		list = append(list, cfg.Name)
	}
	return list
}

// importing is a parsed upload and what it would change.
type importing struct {
	plan     yamlio.Plan
	settings *store.NtfySettings // nil leaves them alone
}

func (i importing) view() importView {
	invalid := i.plan.Invalid
	if invalid == nil {
		invalid = []yamlio.Invalid{}
	}
	return importView{
		Add: names(i.plan.Add), Update: names(i.plan.Update), Unchanged: len(i.plan.Unchanged),
		Invalid: invalid, Notifications: i.settings != nil,
	}
}

// readImport answers the request itself when the file cannot be used.
func (s *Server) readImport(w http.ResponseWriter, r *http.Request) (importing, bool) {
	var request struct {
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_json"})
		return importing{}, false
	}
	backup, err := yamlio.Decode([]byte(request.YAML))
	if err != nil {
		code := "bad_yaml"
		if errors.Is(err, yamlio.ErrVersion) {
			code = "bad_version"
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": code, "detail": err.Error()})
		return importing{}, false
	}
	result, err := s.planImport(r.Context(), backup)
	if err != nil {
		s.fail(w, err)
		return importing{}, false
	}
	return result, true
}

func (s *Server) planImport(ctx context.Context, backup yamlio.Backup) (importing, error) {
	monitors, err := s.Store.ListMonitors(ctx)
	if err != nil {
		return importing{}, err
	}
	existing := make([]model.Config, 0, len(monitors))
	for _, m := range monitors {
		existing = append(existing, m.Config)
	}
	result := importing{plan: yamlio.NewPlan(existing, backup.Monitors)}
	if incoming := backup.Notifications; incoming != nil && incoming.Topic != "" {
		current, err := s.Store.Ntfy(ctx)
		if err != nil {
			return importing{}, err
		}
		next := current // the token is never in the file, so the current one stays
		next.Topic = incoming.Topic
		if incoming.Server != "" {
			next.Server = incoming.Server
		}
		if next != current {
			result.settings = &next
		}
	}
	return result, nil
}

func (s *Server) previewBackup(w http.ResponseWriter, r *http.Request) {
	if result, ok := s.readImport(w, r); ok {
		writeJSON(w, http.StatusOK, result.view())
	}
}

// importBackup applies a file only as a whole: one bad entry stops all of it.
func (s *Server) importBackup(w http.ResponseWriter, r *http.Request) {
	result, ok := s.readImport(w, r)
	if !ok {
		return
	}
	if len(result.plan.Invalid) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "invalid", "invalid": result.plan.Invalid})
		return
	}
	// Detached: a closed tab must not stop the import halfway.
	ctx := context.WithoutCancel(r.Context())
	for _, cfg := range result.plan.Add {
		created, err := s.Store.CreateMonitor(ctx, cfg)
		if err != nil {
			s.fail(w, err)
			return
		}
		s.Scheduler.Upsert(created)
	}
	for _, cfg := range result.plan.Update {
		if err := s.Store.UpdateMonitor(ctx, cfg); err != nil {
			s.fail(w, err)
			return
		}
		s.Scheduler.Upsert(cfg)
	}
	if result.settings != nil {
		if err := s.Store.SaveNtfy(ctx, *result.settings); err != nil {
			s.fail(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, result.view())
}
