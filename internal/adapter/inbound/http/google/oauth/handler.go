// Package oauth contains an opt-in OAuth HTTP handler. It is NOT mounted in
// the public LINE webhook until a verified LINE identity resolver is supplied.
package oauth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
)

const oauthCSRFCookie = "__Host-google_oauth_state"

// TrustedLINEIdentity MUST authenticate the request with LINE Login or
// an equivalent server-side verified LINE session. Never read a claimed
// user ID from request URL, form, query or an unsigned cookie.
type TrustedLINEIdentity func(*http.Request) (string, error)

type LinkActions interface {
	Begin(context.Context, string, []string) (string, error)
	Complete(context.Context, string, string) (workspace.LinkedAccount, error)
}

type Handler struct {
	links     LinkActions
	principal TrustedLINEIdentity
	scopes    []string
}

func NewHandler(links LinkActions, principal TrustedLINEIdentity, scopes []string) (*Handler, error) {
	if links == nil || principal == nil || len(scopes) == 0 {
		return nil, errors.New("Google OAuth handler requires a trusted LINE identity provider and scopes")
	}
	return &Handler{links: links, principal: principal, scopes: append([]string(nil), scopes...)}, nil
}

func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	owner, err := h.principal(r)
	if err != nil || owner == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	redirect, err := h.links.Begin(r.Context(), owner, h.scopes)
	if err != nil {
		http.Error(w, "unable to start Google authorization", http.StatusServiceUnavailable)
		return
	}
	u, err := url.Parse(redirect)
	if err != nil || u.Scheme != "https" || u.Hostname() != "accounts.google.com" {
		http.Error(w, "invalid Google authorization endpoint", http.StatusServiceUnavailable)
		return
	}
	state := u.Query().Get("state")
	if len(state) != 43 {
		http.Error(w, "invalid Google authorization state", http.StatusServiceUnavailable)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: oauthCSRFCookie, Value: state, Path: "/", MaxAge: 600,
		Expires: time.Now().Add(10 * time.Minute), Secure: true, HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, redirect, http.StatusFound)
}

func clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: oauthCSRFCookie, Value: "", Path: "/", MaxAge: -1,
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clearStateCookie(w)
	q := r.URL.Query()
	if len(q["state"]) != 1 || len(q["code"]) != 1 || q.Get("error") != "" {
		http.Error(w, "invalid Google callback", http.StatusBadRequest)
		return
	}
	state, code := q.Get("state"), q.Get("code")
	if len(state) != 43 || code == "" || len(code) > 2048 {
		http.Error(w, "invalid Google callback", http.StatusBadRequest)
		return
	}
	cookie, err := r.Cookie(oauthCSRFCookie)
	if err != nil || len(cookie.Value) != 43 ||
		subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(state)) != 1 {
		http.Error(w, "Google authorization state mismatch", http.StatusBadRequest)
		return
	}
	_, err = h.links.Complete(r.Context(), state, code)
	if err != nil {
		http.Error(w, "Google account connection could not be completed", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Google account linked."))
}
