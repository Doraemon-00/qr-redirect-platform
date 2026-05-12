package app

import (
	"context"
	"encoding/json"
	"time"
)

const (
	activeRedirectCacheTTL    = 10 * time.Minute
	tombstoneRedirectCacheTTL = 5 * time.Minute
)

func redirectCacheKey(token string) string {
	return "redirect:" + token
}

func (s *Server) getRedirectCache(ctx context.Context, token string) (redirectCacheEntry, bool) {
	if !s.cfg.RedirectCacheEnabled {
		return redirectCacheEntry{}, false
	}

	raw, err := s.redis.Get(ctx, redirectCacheKey(token)).Bytes()
	if err != nil {
		return redirectCacheEntry{}, false
	}

	var entry redirectCacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return redirectCacheEntry{}, false
	}
	return entry, true
}

func (s *Server) fillRedirectCacheBestEffort(token string, q qrCode) {
	if !s.cfg.RedirectCacheEnabled {
		return
	}

	entry := redirectCacheEntry{
		TargetURL: q.TargetURL,
		ExpiresAt: q.ExpiresAt,
		DeletedAt: q.DeletedAt,
	}

	ttl := redirectCacheTTL(q, time.Now())
	if ttl <= 0 {
		return
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := s.redis.Set(ctx, redirectCacheKey(token), payload, ttl).Err(); err != nil {
		s.logger.Warn("redirect cache fill failed", "token", token, "error", err)
	}
}

func (s *Server) invalidateRedirectCache(ctx context.Context, token string) {
	if !s.cfg.RedirectCacheEnabled {
		return
	}

	if err := s.redis.Del(ctx, redirectCacheKey(token)).Err(); err != nil {
		s.logger.Warn("redirect cache invalidation failed", "token", token, "error", err)
	}
}

func redirectCacheTTL(q qrCode, now time.Time) time.Duration {
	if q.DeletedAt != nil {
		return tombstoneRedirectCacheTTL
	}
	if q.ExpiresAt == nil {
		return activeRedirectCacheTTL
	}

	untilExpiry := q.ExpiresAt.Sub(now)
	if untilExpiry <= 0 {
		return tombstoneRedirectCacheTTL
	}
	if untilExpiry < activeRedirectCacheTTL {
		return untilExpiry
	}
	return activeRedirectCacheTTL
}
