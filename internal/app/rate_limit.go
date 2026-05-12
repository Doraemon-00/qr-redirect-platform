package app

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const ownerRateLimitWindow = time.Minute

type rateLimitDecision struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
	ResetAt    time.Time
}

func (s *Server) rateLimitOwnerAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ownerID := ownerIDFromContext(r.Context())
		if ownerID == "" {
			writeError(w, http.StatusUnauthorized, "missing owner context")
			return
		}

		decision, err := s.checkOwnerRateLimit(r.Context(), ownerID, time.Now())
		if err != nil {
			ownerRateLimitFailuresTotal.Inc()
			s.logger.Warn("owner rate limit unavailable", "owner_id", ownerID, "error", err)
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("RateLimit-Limit", strconv.Itoa(s.cfg.OwnerRateLimit))
		w.Header().Set("RateLimit-Remaining", strconv.Itoa(decision.Remaining))
		w.Header().Set("RateLimit-Reset", strconv.FormatInt(decision.ResetAt.Unix(), 10))

		if !decision.Allowed {
			ownerRateLimitedTotal.Inc()
			w.Header().Set("Retry-After", strconv.Itoa(int(decision.RetryAfter.Seconds())))
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) checkOwnerRateLimit(ctx context.Context, ownerID string, now time.Time) (rateLimitDecision, error) {
	key, resetAt := ownerRateLimitKey(ownerID, now)

	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return rateLimitDecision{}, err
	}
	if count == 1 {
		if err := s.redis.Expire(ctx, key, ownerRateLimitWindow+time.Second).Err(); err != nil {
			return rateLimitDecision{}, err
		}
	}

	remaining := s.cfg.OwnerRateLimit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	retryAfter := time.Until(resetAt)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return rateLimitDecision{
		Allowed:    count <= int64(s.cfg.OwnerRateLimit),
		Remaining:  remaining,
		RetryAfter: retryAfter,
		ResetAt:    resetAt,
	}, nil
}

func ownerRateLimitKey(ownerID string, now time.Time) (string, time.Time) {
	windowStart := now.Truncate(ownerRateLimitWindow)
	resetAt := windowStart.Add(ownerRateLimitWindow)
	return fmt.Sprintf("rate:owner:%s:%d", ownerID, windowStart.Unix()), resetAt
}
