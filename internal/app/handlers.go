package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"github.com/skip2/go-qrcode"

	"github.com/doraemon-00/qrcode/internal/token"
)

func (s *Server) createQR(w http.ResponseWriter, r *http.Request) {
	var req createQRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	normalized, err := normalizeTargetURL(req.TargetURL)
	if err != nil {
		writeURLError(w, err)
		return
	}

	var q qrCode
	for attempt := 0; attempt < 5; attempt++ {
		generated, err := token.Generate(token.DefaultLength)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}

		q, err = s.insertQRCode(r.Context(), demoOwnerID, generated, normalized, req.ExpiresAt)
		if err == nil {
			writeJSON(w, http.StatusCreated, q.withURLs(s.cfg.PublicBaseURL))
			return
		}
		if !isUniqueViolation(err) {
			s.logger.Error("create qr failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create qr code")
			return
		}
	}

	writeError(w, http.StatusInternalServerError, "failed to allocate unique token")
}

func (s *Server) getQR(w http.ResponseWriter, r *http.Request) {
	q, err := s.getQRCodeForOwner(r.Context(), demoOwnerID, chi.URLParam(r, "token"))
	if errors.Is(err, errQRNotFound) {
		writeError(w, http.StatusNotFound, "qr code not found")
		return
	}
	if err != nil {
		s.logger.Error("get qr failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get qr code")
		return
	}
	writeJSON(w, http.StatusOK, q.withURLs(s.cfg.PublicBaseURL))
}

func (s *Server) updateQR(w http.ResponseWriter, r *http.Request) {
	var req updateQRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	var normalized *string
	if req.TargetURL != nil {
		value, err := normalizeTargetURL(*req.TargetURL)
		if err != nil {
			writeURLError(w, err)
			return
		}
		normalized = &value
	}

	token := chi.URLParam(r, "token")
	q, err := s.updateQRCode(r.Context(), demoOwnerID, token, normalized, req.ExpiresAt)
	if errors.Is(err, errQRNotFound) {
		writeError(w, http.StatusNotFound, "qr code not found")
		return
	}
	if err != nil {
		s.logger.Error("update qr failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update qr code")
		return
	}

	s.invalidateRedirectCache(r.Context(), token)
	writeJSON(w, http.StatusOK, q.withURLs(s.cfg.PublicBaseURL))
}

func (s *Server) deleteQR(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	err := s.softDeleteQRCode(r.Context(), demoOwnerID, token)
	if errors.Is(err, errQRNotFound) {
		writeError(w, http.StatusNotFound, "qr code not found")
		return
	}
	if err != nil {
		s.logger.Error("delete qr failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete qr code")
		return
	}

	s.invalidateRedirectCache(r.Context(), token)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) getAnalytics(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if _, err := s.getQRCodeForOwner(r.Context(), demoOwnerID, token); errors.Is(err, errQRNotFound) {
		writeError(w, http.StatusNotFound, "qr code not found")
		return
	} else if err != nil {
		s.logger.Error("analytics qr lookup failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get analytics")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":       token,
		"totalScans":  0,
		"scansByDay":  []any{},
		"consistency": "analytics pipeline scaffolded; ClickHouse worker pending",
	})
}

func (s *Server) getQRImage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	q, err := s.getQRCodeForRedirect(r.Context(), token)
	if errors.Is(err, errQRNotFound) {
		writeError(w, http.StatusNotFound, "qr code not found")
		return
	}
	if err != nil {
		s.logger.Error("qr image lookup failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get qr code")
		return
	}
	if q.DeletedAt != nil {
		writeError(w, http.StatusGone, "qr code deleted")
		return
	}

	shortURL := s.cfg.PublicBaseURL + "/r/" + token

	png, err := qrcode.Encode(shortURL, qrcode.Medium, 256)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate qr image")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		redirectLatencySeconds.Observe(time.Since(start).Seconds())
	}()

	token := chi.URLParam(r, "token")
	if !isValidRouteToken(token) {
		redirectRequestsTotal.WithLabelValues("not_found").Inc()
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if entry, ok := s.getRedirectCache(r.Context(), token); ok {
		redirectCacheHitsTotal.Inc()
		if isGone(entry.ExpiresAt, entry.DeletedAt) {
			redirectRequestsTotal.WithLabelValues("gone").Inc()
			writeError(w, http.StatusGone, "gone")
			return
		}
		s.enqueueScanEventBestEffort(token, r)
		redirectRequestsTotal.WithLabelValues("redirect").Inc()
		http.Redirect(w, r, entry.TargetURL, http.StatusFound)
		return
	}
	redirectCacheMissesTotal.Inc()

	q, err := s.getQRCodeForRedirect(r.Context(), token)
	if errors.Is(err, errQRNotFound) {
		redirectRequestsTotal.WithLabelValues("not_found").Inc()
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		s.logger.Error("redirect lookup failed", "error", err)
		redirectRequestsTotal.WithLabelValues("error").Inc()
		writeError(w, http.StatusInternalServerError, "redirect lookup failed")
		return
	}

	go s.fillRedirectCacheBestEffort(token, q)

	if isGone(q.ExpiresAt, q.DeletedAt) {
		redirectRequestsTotal.WithLabelValues("gone").Inc()
		writeError(w, http.StatusGone, "gone")
		return
	}

	s.enqueueScanEventBestEffort(token, r)
	redirectRequestsTotal.WithLabelValues("redirect").Inc()
	http.Redirect(w, r, q.TargetURL, http.StatusFound)
}

func writeURLError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBlockedURL):
		writeError(w, http.StatusUnprocessableEntity, "blocked target url")
	default:
		writeError(w, http.StatusBadRequest, "invalid target url")
	}
}

func isGone(expiresAt, deletedAt *time.Time) bool {
	if deletedAt != nil {
		return true
	}
	return expiresAt != nil && time.Now().After(*expiresAt)
}

func isValidRouteToken(value string) bool {
	if len(value) != token.DefaultLength {
		return false
	}
	for _, r := range value {
		if !strings.ContainsRune(token.Alphabet, r) {
			return false
		}
	}
	return true
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Server) enqueueScanEventBestEffort(token string, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	userAgent := r.Header.Get("User-Agent")
	ip := ""
	if r.RemoteAddr != "" {
		ip = r.RemoteAddr
	}

	if err := s.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: "scan_events",
		Approx: true,
		MaxLen: 100000,
		Values: map[string]any{
			"token":           token,
			"scanned_at":      time.Now().UTC().Format(time.RFC3339Nano),
			"user_agent_hash": hashForAnalytics(userAgent),
			"ip_hash":         hashForAnalytics(ip),
		},
	}).Err(); err != nil {
		analyticsEnqueueFailuresTotal.Inc()
		s.logger.Warn("scan event enqueue failed", "token", token, "error", err)
	}
}

func hashForAnalytics(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
