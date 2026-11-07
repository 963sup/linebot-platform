// Package miniapp serves the provider UI and its authenticated session API.
package miniapp

import (
	"embed"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//go:embed assets/*
var assets embed.FS

type Config struct{ LIFFID, AllowedOrigin string }
type Handler struct {
	liffID         string
	sessionHandler http.Handler
}

func New(config Config, verifier SessionVerifier) (*Handler, error) {
	if verifier == nil || strings.TrimSpace(config.LIFFID) == "" || config.LIFFID != strings.TrimSpace(config.LIFFID) {
		return nil, errors.New("miniapp requires a LIFF ID and session verifier")
	}
	origin, err := url.Parse(config.AllowedOrigin)
	localHTTP := origin != nil && origin.Scheme == "http" && (origin.Hostname() == "localhost" || origin.Hostname() == "127.0.0.1")
	if err != nil || origin == nil || (origin.Scheme != "https" && !localHTTP) || origin.Host == "" || origin.User != nil || config.AllowedOrigin != origin.Scheme+"://"+origin.Host {
		return nil, errors.New("miniapp allowed origin must be exact HTTPS or loopback HTTP")
	}
	handler := &Handler{liffID: config.LIFFID}
	handler.sessionHandler = authenticator{verifier: verifier, allowedOrigin: origin.String()}.wrap(http.HandlerFunc(handler.session))
	return handler, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	secureHeaders(w)
	switch r.URL.Path {
	case "/miniapp":
		if r.Method != http.MethodGet {
			writeError(w, 405, "method_not_allowed", "method not allowed")
			return
		}
		http.Redirect(w, r, "/miniapp/", http.StatusPermanentRedirect)
	case "/miniapp/":
		h.asset(w, r, "assets/index.html", "text/html; charset=utf-8")
	case "/miniapp/config.json":
		if r.Method != http.MethodGet {
			writeError(w, 405, "method_not_allowed", "method not allowed")
			return
		}
		writeJSON(w, 200, struct {
			LIFFID string `json:"liffId"`
		}{h.liffID})
	case "/miniapp/assets/app.css":
		h.asset(w, r, "assets/app.css", "text/css; charset=utf-8")
	case "/miniapp/assets/app.js":
		h.asset(w, r, "assets/app.js", "text/javascript; charset=utf-8")
	case "/miniapp/api/v1/session":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		h.sessionHandler.ServeHTTP(w, r)
	default:
		writeError(w, 404, "not_found", "resource not found")
	}
}

func secureHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' https://static.line-scdn.net; style-src 'self'; connect-src 'self' https://api.line.me; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.Header().Set("Permissions-Policy", "geolocation=(self)")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
}
func (h *Handler) asset(w http.ResponseWriter, r *http.Request, name, contentType string) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	data, err := assets.ReadFile(name)
	if err != nil {
		writeError(w, 500, "internal_error", "request could not be completed")
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}
func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	identity, ok := IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "request could not be completed")
		return
	}
	writeJSON(w, 200, struct {
		Authenticated bool   `json:"authenticated"`
		ValidatedAt   string `json:"validatedAt"`
	}{true, identity.ValidatedAt.UTC().Format(time.RFC3339Nano)})
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var value errorEnvelope
	value.Error.Code, value.Error.Message = code, message
	writeJSON(w, status, value)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
