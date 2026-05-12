package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	scanEventsStream        = "scan_events"
	scanEventsConsumerGroup = "analytics-writers"
)

func (s *Server) RunAnalyticsWorker(ctx context.Context) {
	if !s.cfg.AnalyticsWorkerEnabled {
		s.logger.Info("analytics worker disabled")
		return
	}

	if err := s.ensureAnalyticsConsumerGroup(ctx); err != nil {
		analyticsWorkerFailuresTotal.Inc()
		s.logger.Error("analytics consumer group setup failed", "error", err)
		return
	}

	consumer := "api"
	s.logger.Info("analytics worker started", "stream", scanEventsStream, "group", scanEventsConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("analytics worker stopped")
			return
		default:
		}

		s.refreshAnalyticsStreamMetrics(ctx)

		events, reclaimed, err := s.nextScanEventBatch(ctx, consumer)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			analyticsWorkerFailuresTotal.Inc()
			s.logger.Warn("analytics pending reclaim failed", "error", err)
			sleepOrDone(ctx, time.Second)
			continue
		}
		if len(events) == 0 {
			continue
		}
		if reclaimed {
			analyticsEventsReclaimedTotal.Add(float64(len(events)))
		}

		writeStart := time.Now()
		if err := s.writeScanEvents(ctx, events); err != nil {
			analyticsWorkerFailuresTotal.Inc()
			s.logger.Warn("analytics clickhouse write failed", "events", len(events), "error", err)
			sleepOrDone(ctx, time.Second)
			continue
		}
		analyticsBatchWriteDurationSeconds.Observe(time.Since(writeStart).Seconds())

		ids := make([]string, 0, len(events))
		for _, event := range events {
			ids = append(ids, event.StreamID)
		}
		if err := s.redis.XAck(ctx, scanEventsStream, scanEventsConsumerGroup, ids...).Err(); err != nil {
			analyticsWorkerFailuresTotal.Inc()
			s.logger.Warn("analytics stream ack failed", "events", len(events), "error", err)
			continue
		}
		analyticsEventsWrittenTotal.Add(float64(len(events)))
		analyticsBatchesWrittenTotal.Inc()
		s.refreshAnalyticsStreamMetrics(ctx)
	}
}

func (s *Server) ensureAnalyticsConsumerGroup(ctx context.Context) error {
	err := s.redis.XGroupCreateMkStream(ctx, scanEventsStream, scanEventsConsumerGroup, "0").Err()
	if err != nil && !isRedisBusyGroup(err) {
		return err
	}
	return nil
}

func isRedisBusyGroup(err error) bool {
	var redisErr redis.Error
	return errors.As(err, &redisErr) && strings.HasPrefix(redisErr.Error(), "BUSYGROUP")
}

func (s *Server) nextScanEventBatch(ctx context.Context, consumer string) ([]scanEvent, bool, error) {
	flushBy := time.Now().Add(time.Duration(s.cfg.AnalyticsBlockSeconds) * time.Second)

	events, reclaimed, err := s.reclaimStaleScanEvents(ctx, consumer)
	if err != nil || len(events) >= s.cfg.AnalyticsBatchSize {
		return events, reclaimed, err
	}

	if len(events) == 0 {
		events, err = s.readScanEvents(ctx, consumer, s.cfg.AnalyticsBatchSize, time.Duration(s.cfg.AnalyticsBlockSeconds)*time.Second)
		if err != nil || len(events) == 0 {
			return events, false, err
		}
	}

	for len(events) < s.cfg.AnalyticsBatchSize {
		remaining := time.Until(flushBy)
		if remaining <= 0 {
			break
		}
		block := minDuration(remaining, 50*time.Millisecond)
		more, err := s.readScanEvents(ctx, consumer, s.cfg.AnalyticsBatchSize-len(events), block)
		if err != nil {
			return nil, reclaimed, err
		}
		events = append(events, more...)
	}

	return events, reclaimed, nil
}

func (s *Server) reclaimStaleScanEvents(ctx context.Context, consumer string) ([]scanEvent, bool, error) {
	messages, _, err := s.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   scanEventsStream,
		Group:    scanEventsConsumerGroup,
		Consumer: consumer,
		MinIdle:  time.Duration(s.cfg.AnalyticsReclaimIdleSeconds) * time.Second,
		Start:    "0-0",
		Count:    int64(s.cfg.AnalyticsBatchSize),
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return parseScanMessages(messages), len(messages) > 0, nil
}

func (s *Server) readScanEvents(ctx context.Context, consumer string, count int, block time.Duration) ([]scanEvent, error) {
	streams, err := s.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    scanEventsConsumerGroup,
		Consumer: consumer,
		Streams:  []string{scanEventsStream, ">"},
		Count:    int64(count),
		Block:    block,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	events := make([]scanEvent, 0, s.cfg.AnalyticsBatchSize)
	for _, stream := range streams {
		events = append(events, parseScanMessages(stream.Messages)...)
	}
	return events, nil
}

func parseScanMessages(messages []redis.XMessage) []scanEvent {
	events := make([]scanEvent, 0, len(messages))
	for _, message := range messages {
		event, ok := parseScanEvent(message)
		if ok {
			events = append(events, event)
		}
	}
	return events
}

func parseScanEvent(message redis.XMessage) (scanEvent, bool) {
	token, ok := streamString(message.Values, "token")
	if !ok {
		return scanEvent{}, false
	}
	scannedAtRaw, ok := streamString(message.Values, "scanned_at")
	if !ok {
		return scanEvent{}, false
	}
	scannedAt, err := time.Parse(time.RFC3339Nano, scannedAtRaw)
	if err != nil {
		return scanEvent{}, false
	}
	userAgentHash, _ := streamString(message.Values, "user_agent_hash")
	ipHash, _ := streamString(message.Values, "ip_hash")

	return scanEvent{
		StreamID:      message.ID,
		Token:         token,
		ScannedAt:     scannedAt.UTC(),
		UserAgentHash: userAgentHash,
		IPHash:        ipHash,
	}, true
}

func streamString(values map[string]any, key string) (string, bool) {
	value, ok := values[key]
	if !ok || value == nil {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		return typed, true
	case []byte:
		return string(typed), true
	default:
		return "", false
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func (s *Server) writeScanEvents(ctx context.Context, events []scanEvent) error {
	batch, err := s.ch.PrepareBatch(ctx, "INSERT INTO scan_events (event_id, token, scanned_at, user_agent_hash, ip_hash)")
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := appendScanEvent(batch, event); err != nil {
			return err
		}
	}
	return batch.Send()
}

func appendScanEvent(batch driver.Batch, event scanEvent) error {
	return batch.Append(
		scanEventUUID(event.StreamID),
		event.Token,
		event.ScannedAt,
		event.UserAgentHash,
		event.IPHash,
	)
}

func scanEventUUID(streamID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(streamID))
}

func (s *Server) getScanAnalytics(ctx context.Context, token string) (scanAnalytics, error) {
	var total uint64
	if err := s.ch.QueryRow(ctx, "SELECT uniqExact(event_id) FROM scan_events WHERE token = ?", token).Scan(&total); err != nil {
		return scanAnalytics{}, err
	}

	rows, err := s.ch.Query(ctx, `
		SELECT toDate(scanned_at) AS day, uniqExact(event_id) AS scans
		FROM scan_events
		WHERE token = ?
		GROUP BY day
		ORDER BY day
	`, token)
	if err != nil {
		return scanAnalytics{}, err
	}
	defer rows.Close()

	byDay := make([]dailyScanCount, 0)
	for rows.Next() {
		var day time.Time
		var count uint64
		if err := rows.Scan(&day, &count); err != nil {
			return scanAnalytics{}, err
		}
		byDay = append(byDay, dailyScanCount{
			Date:  day.Format("2006-01-02"),
			Count: count,
		})
	}
	if err := rows.Err(); err != nil {
		return scanAnalytics{}, err
	}

	return scanAnalytics{
		Token:       token,
		TotalScans:  total,
		ScansByDay:  byDay,
		Consistency: "eventually consistent; redirect events are queued in Redis and batch-written to ClickHouse",
	}, nil
}

func (s *Server) refreshAnalyticsStreamMetrics(ctx context.Context) {
	if length, err := s.redis.XLen(ctx, scanEventsStream).Result(); err == nil {
		analyticsStreamLength.Set(float64(length))
	}
	if pending, err := s.redis.XPending(ctx, scanEventsStream, scanEventsConsumerGroup).Result(); err == nil {
		analyticsEventsPending.Set(float64(pending.Count))
	}
}

func sleepOrDone(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
