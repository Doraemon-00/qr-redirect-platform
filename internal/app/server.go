package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg    Config
	logger *slog.Logger
	db     *pgxpool.Pool
	redis  *redis.Client
}

func NewServer(ctx context.Context, cfg Config, logger *slog.Logger) (*Server, func(), error) {
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:        cfg.RedisAddr,
		DialTimeout: 500 * time.Millisecond,
		ReadTimeout: 500 * time.Millisecond,
	})

	s := &Server{
		cfg:    cfg,
		logger: logger,
		db:     db,
		redis:  rdb,
	}

	cleanup := func() {
		_ = rdb.Close()
		db.Close()
	}

	return s, cleanup, nil
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.healthz)
	r.Get("/readyz", s.readyz)
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/qr", func(r chi.Router) {
		r.Use(s.requireAPIKey)
		r.Post("/create", s.createQR)
		r.Get("/{token}", s.getQR)
		r.Patch("/{token}", s.updateQR)
		r.Delete("/{token}", s.deleteQR)
		r.Get("/{token}/analytics", s.getAnalytics)
	})

	r.Get("/api/qr/{token}/image", s.getQRImage)
	r.Get("/r/{token}", s.redirect)

	return r
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Millisecond)
	defer cancel()

	if err := s.redis.Ping(ctx).Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "redis": "unavailable"})
		return
	}
	if err := s.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "postgres": "unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
