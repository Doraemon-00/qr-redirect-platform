package app

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
)

type contextKey string

const ownerIDContextKey contextKey = "owner_id"

func (s *Server) requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(authHeader, "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.DemoAPIKey)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		ctx := context.WithValue(r.Context(), ownerIDContextKey, demoOwnerID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ownerIDFromContext(ctx context.Context) string {
	ownerID, _ := ctx.Value(ownerIDContextKey).(string)
	return ownerID
}
