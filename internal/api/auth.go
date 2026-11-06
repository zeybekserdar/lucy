package api

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"lucy/internal/auth"
)

const (
	cookieName = "lucy_session"
	sessionTTL = 30 * 24 * time.Hour
)

type authStatus struct {
	Status            string     `json:"status"` // needsSetup, loggedOut or loggedIn
	PasswordChangedAt *time.Time `json:"passwordChangedAt,omitempty"`
}

func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func secure(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (s *Server) signedIn(r *http.Request) (bool, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return false, nil
	}
	return s.Store.SessionValid(r.Context(), auth.HashToken(cookie.Value), time.Now())
}

// authed rejects a request without a live session.
func (s *Server) authed(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, err := s.signedIn(r)
		if err != nil {
			s.fail(w, err)
			return
		}
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) error {
	token, hash, err := auth.NewToken()
	if err != nil {
		return err
	}
	now := time.Now()
	if err := s.Store.CreateSession(r.Context(), hash, now, now.Add(sessionTTL)); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure(r),
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	hash, err := s.Store.PasswordHash(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	if hash == "" {
		writeJSON(w, http.StatusOK, authStatus{Status: "needsSetup"})
		return
	}
	ok, err := s.signedIn(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusOK, authStatus{Status: "loggedOut"})
		return
	}
	changed, err := s.Store.PasswordChangedAt(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authStatus{Status: "loggedIn", PasswordChangedAt: optional(changed)})
}

func readJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(value); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_json"})
		return false
	}
	return true
}

func invalidField(w http.ResponseWriter, field, code string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "invalid", "fields": map[string]string{field: code}})
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if !auth.ValidPassword(body.Password) {
		invalidField(w, "password", "password_short")
		return
	}
	hash, err := auth.Hash(body.Password)
	if err != nil {
		s.fail(w, err)
		return
	}
	claimed, err := s.Store.ClaimPassword(r.Context(), hash, time.Now())
	if err != nil {
		s.fail(w, err)
		return
	}
	if !claimed {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "already_set"})
		return
	}
	if err := s.startSession(w, r); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// throttled answers the request when this client must wait.
func (s *Server) throttled(w http.ResponseWriter, r *http.Request) bool {
	ok, wait := s.limiter.Allow(clientKey(r))
	if ok {
		return false
	}
	seconds := int(math.Ceil(wait.Seconds()))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "rate_limited", "retryAfterSec": seconds})
	return true
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.throttled(w, r) {
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	hash, err := s.Store.PasswordHash(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	if hash == "" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "needs_setup"})
		return
	}
	if !auth.Verify(body.Password, hash) {
		s.limiter.Fail(clientKey(r))
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "wrong_password"})
		return
	}
	s.limiter.Reset(clientKey(r))
	if err := s.startSession(w, r); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		if err := s.Store.DeleteSession(r.Context(), auth.HashToken(cookie.Value)); err != nil {
			s.fail(w, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Path: "/", MaxAge: -1, HttpOnly: true,
		Secure: secure(r), SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// changePassword ends every other session; a stolen one dies with the old password.
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	if s.throttled(w, r) {
		return
	}
	var body struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	stored, err := s.Store.PasswordHash(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	if !auth.Verify(body.Current, stored) {
		s.limiter.Fail(clientKey(r))
		invalidField(w, "current", "wrong_password")
		return
	}
	if !auth.ValidPassword(body.Next) {
		invalidField(w, "next", "password_short")
		return
	}
	hash, err := auth.Hash(body.Next)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Store.ReplacePassword(r.Context(), hash, time.Now()); err != nil {
		s.fail(w, err)
		return
	}
	s.limiter.Reset(clientKey(r))
	if err := s.startSession(w, r); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
