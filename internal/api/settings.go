package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"lucy/internal/notify"
	"lucy/internal/store"
)

var ntfyTopic = regexp.MustCompile(`^[-_A-Za-z0-9]{1,64}$`)

type ntfyView struct {
	Server    string `json:"server"`
	Topic     string `json:"topic"`
	Token     string `json:"token"`
	PublicURL string `json:"publicUrl"`
}

func webAddress(value string) bool {
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// readNtfy decodes and validates settings, answering the request on failure.
func readNtfy(w http.ResponseWriter, r *http.Request) (store.NtfySettings, bool) {
	var view ntfyView
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&view); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_json"})
		return store.NtfySettings{}, false
	}
	settings := store.NtfySettings{
		Server: strings.TrimRight(strings.TrimSpace(view.Server), "/"),
		Topic:  strings.TrimSpace(view.Topic),
		Token:  strings.TrimSpace(view.Token),

		PublicURL: strings.TrimRight(strings.TrimSpace(view.PublicURL), "/"),
	}
	fields := map[string]string{}
	if !webAddress(settings.Server) {
		fields["server"] = "url_scheme"
	}
	if settings.PublicURL != "" && !webAddress(settings.PublicURL) {
		fields["publicUrl"] = "url_scheme"
	}
	if settings.Topic != "" && !ntfyTopic.MatchString(settings.Topic) {
		fields["topic"] = "topic"
	}
	if len(fields) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "invalid", "fields": fields})
		return settings, false
	}
	return settings, true
}

func (s *Server) getNtfy(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.Ntfy(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ntfyView{
		Server: settings.Server, Topic: settings.Topic, Token: settings.Token, PublicURL: settings.PublicURL,
	})
}

func (s *Server) putNtfy(w http.ResponseWriter, r *http.Request) {
	settings, ok := readNtfy(w, r)
	if !ok {
		return
	}
	if err := s.Store.SaveNtfy(r.Context(), settings); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testNtfy publishes to the values on screen, saved or not, so a mistake shows
// up before it is kept.
func (s *Server) testNtfy(w http.ResponseWriter, r *http.Request) {
	settings, ok := readNtfy(w, r)
	if !ok {
		return
	}
	if settings.Topic == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "invalid", "fields": map[string]string{"topic": "required"}})
		return
	}
	composer := notify.Composer{}
	if s.Scheduler.Composer != nil {
		composer = *s.Scheduler.Composer
	}
	composer.PublicURL = settings.PublicURL
	target := notify.Ntfy{Server: settings.Server, Topic: settings.Topic, Token: settings.Token}
	// Detached: leaving the page must not turn a sent message into a failure.
	status, err := target.Publish(context.WithoutCancel(r.Context()), composer.Test())
	writeJSON(w, http.StatusOK, map[string]any{"ok": err == nil, "status": status})
}
