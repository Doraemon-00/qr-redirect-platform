package app

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var errQRNotFound = errors.New("qr code not found")

func (s *Server) insertQRCode(ctx context.Context, ownerID, token, targetURL string, expiresAt *time.Time) (qrCode, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO qr_codes (owner_id, token, target_url, normalized_url, expires_at)
		VALUES ($1, $2, $3, $3, $4)
		RETURNING id::text, owner_id::text, token, target_url, normalized_url, expires_at, deleted_at, created_at, updated_at
	`, ownerID, token, targetURL, expiresAt)
	return scanQRCode(row)
}

func (s *Server) getQRCodeForOwner(ctx context.Context, ownerID, token string) (qrCode, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id::text, owner_id::text, token, target_url, normalized_url, expires_at, deleted_at, created_at, updated_at
		FROM qr_codes
		WHERE owner_id = $1 AND token = $2
	`, ownerID, token)
	return scanQRCode(row)
}

func (s *Server) getQRCodeForRedirect(ctx context.Context, token string) (qrCode, error) {
	redirectDBLookupsTotal.Inc()

	row := s.db.QueryRow(ctx, `
		SELECT id::text, owner_id::text, token, target_url, normalized_url, expires_at, deleted_at, created_at, updated_at
		FROM qr_codes
		WHERE token = $1
	`, token)
	return scanQRCode(row)
}

func (s *Server) updateQRCode(ctx context.Context, ownerID, token string, targetURL *string, expiresAt *time.Time) (qrCode, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE qr_codes
		SET
			target_url = COALESCE($3, target_url),
			normalized_url = COALESCE($3, normalized_url),
			expires_at = COALESCE($4, expires_at),
			updated_at = now()
		WHERE owner_id = $1 AND token = $2 AND deleted_at IS NULL
		RETURNING id::text, owner_id::text, token, target_url, normalized_url, expires_at, deleted_at, created_at, updated_at
	`, ownerID, token, targetURL, expiresAt)
	return scanQRCode(row)
}

func (s *Server) softDeleteQRCode(ctx context.Context, ownerID, token string) error {
	result, err := s.db.Exec(ctx, `
		UPDATE qr_codes
		SET deleted_at = COALESCE(deleted_at, now()), updated_at = now()
		WHERE owner_id = $1 AND token = $2
	`, ownerID, token)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errQRNotFound
	}
	return nil
}

type qrRow interface {
	Scan(dest ...any) error
}

func scanQRCode(row qrRow) (qrCode, error) {
	var q qrCode
	err := row.Scan(
		&q.ID,
		&q.OwnerID,
		&q.Token,
		&q.TargetURL,
		&q.NormalizedURL,
		&q.ExpiresAt,
		&q.DeletedAt,
		&q.CreatedAt,
		&q.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return qrCode{}, errQRNotFound
	}
	return q, err
}
