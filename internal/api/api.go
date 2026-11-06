// Package api serves the REST API and the web interface.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"lucy/internal/auth"
	"lucy/internal/checker"
	"lucy/internal/model"
	"lucy/internal/scheduler"
	"lucy/internal/store"
)

type Server struct {
	Store     *store.Store
	Scheduler *scheduler.Scheduler
	Checker   *checker.Checker
	Guard     *checker.Guard
	Web       fs.FS // nil in development, where Vite serves the interface
	Log       *slog.Logger

	limiter *auth.Limiter
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.limiter = auth.NewLimiter()
	mux.HandleFunc("GET /api/auth/status", s.authStatus)
	mux.HandleFunc("POST /api/auth/setup", s.setup)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("POST /api/auth/password", s.authed(s.changePassword))
	mux.HandleFunc("GET /api/monitors", s.authed(s.listMonitors))
	mux.HandleFunc("POST /api/monitors", s.authed(s.createMonitor))
	mux.HandleFunc("POST /api/monitors/test", s.authed(s.testMonitor))
	mux.HandleFunc("GET /api/monitors/{id}", s.authed(s.getMonitor))
	mux.HandleFunc("GET /api/monitors/{id}/history", s.authed(s.history))
	mux.HandleFunc("PUT /api/monitors/{id}", s.authed(s.updateMonitor))
	mux.HandleFunc("DELETE /api/monitors/{id}", s.authed(s.deleteMonitor))
	mux.HandleFunc("POST /api/monitors/{id}/check", s.authed(s.checkMonitor))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
	})
	if s.Web != nil {
		mux.Handle("/", spa(s.Web))
	}
	return requireJSON(mux)
}

// requireJSON makes writes need a CORS preflight, which no other origin
// passes, so a web page cannot drive this API from the owner's browser.
func requireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if mediaType != "application/json" {
				writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "json_required"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// spa serves built assets and falls back to index.html for client routes.
func spa(web fs.FS) http.Handler {
	files := http.FileServerFS(web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if _, err := fs.Stat(web, path); err == nil {
				files.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFileFS(w, r, web, "index.html")
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	s.Log.Error("api", "err", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
}

// readConfig decodes and validates a monitor, answering the request on failure.
func readConfig(w http.ResponseWriter, r *http.Request) (model.Config, bool) {
	var cfg model.Config
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_json"})
		return cfg, false
	}
	cfg.Normalize()
	if errs := cfg.Validate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "invalid", "fields": errs})
		return cfg, false
	}
	return cfg, true
}

func (s *Server) listMonitors(w http.ResponseWriter, r *http.Request) {
	dashboard, err := s.dashboard(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (s *Server) getMonitor(w http.ResponseWriter, r *http.Request) {
	monitor, err := s.Store.GetMonitor(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, monitor.Config)
}

func (s *Server) createMonitor(w http.ResponseWriter, r *http.Request) {
	cfg, ok := readConfig(w, r)
	if !ok {
		return
	}
	cfg, err := s.Store.CreateMonitor(r.Context(), cfg)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.Scheduler.Upsert(cfg)
	writeJSON(w, http.StatusCreated, cfg)
}

func (s *Server) updateMonitor(w http.ResponseWriter, r *http.Request) {
	cfg, ok := readConfig(w, r)
	if !ok {
		return
	}
	cfg.ID = r.PathValue("id")
	if err := s.Store.UpdateMonitor(r.Context(), cfg); err != nil {
		s.fail(w, err)
		return
	}
	s.Scheduler.Upsert(cfg)
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) deleteMonitor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.Store.DeleteMonitor(r.Context(), id); err != nil {
		s.fail(w, err)
		return
	}
	s.Scheduler.Remove(id)
	w.WriteHeader(http.StatusNoContent)
}

// testMonitor runs a draft once without saving anything.
func (s *Server) testMonitor(w http.ResponseWriter, r *http.Request) {
	cfg, ok := readConfig(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.Checker.Run(r.Context(), cfg))
}

func (s *Server) checkMonitor(w http.ResponseWriter, r *http.Request) {
	monitor, err := s.Store.GetMonitor(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	// Detached: a closed browser tab must not leave the check half applied.
	result, err := s.Scheduler.Run(context.WithoutCancel(r.Context()), monitor.Config)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
