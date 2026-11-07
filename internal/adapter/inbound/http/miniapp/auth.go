package miniapp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	linesession "github.com/963sup/linebot-platform/internal/application/usecases/line-session"
)

type SessionVerifier interface {
	Verify(context.Context, string) (linesession.Identity, error)
}

type identityContextKey struct{}

type authenticator struct {
	verifier      SessionVerifier
	allowedOrigin string
}

func (a authenticator) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !originAllowed(r, a.allowedOrigin) {
			writeError(w, http.StatusForbidden, "forbidden", "request origin is not allowed")
			return
		}
		token, ok := bearerToken(r.Header.Values("Authorization"))
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return
		}
		identity, err := a.verifier.Verify(r.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, linesession.ErrUnauthenticated):
				writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
			case errors.Is(err, context.Canceled):
				writeError(w, http.StatusRequestTimeout, "request_cancelled", "request was cancelled")
			case errors.Is(err, context.DeadlineExceeded):
				writeError(w, http.StatusGatewayTimeout, "deadline_exceeded", "request deadline exceeded")
			default:
				writeError(w, http.StatusServiceUnavailable, "identity_unavailable", "identity verification is unavailable")
			}
			return
		}
		ctx := context.WithValue(r.Context(), identityContextKey{}, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IdentityFromContext returns the server-verified LINE identity attached by
// the MINI App authentication boundary.
func IdentityFromContext(ctx context.Context) (linesession.Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(linesession.Identity)
	return identity, ok
}

func originAllowed(r *http.Request, allowedOrigin string) bool {
	origins := r.Header.Values("Origin")
	allowed := len(origins) == 1 && origins[0] == allowedOrigin
	if len(origins) == 0 {
		allowed = len(r.Header.Values("Sec-Fetch-Site")) == 1 && r.Header.Get("Sec-Fetch-Site") == "same-origin"
	}
	return allowed && (len(r.Header.Values("Sec-Fetch-Site")) == 0 || r.Header.Get("Sec-Fetch-Site") == "same-origin")
}

func bearerToken(values []string) (string, bool) {
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return "", false
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	return token, token != "" && token == strings.TrimSpace(token) && !strings.ContainsAny(token, "\t ,") && len(token) <= 4096
}
