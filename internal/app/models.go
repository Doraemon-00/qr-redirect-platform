package app

import "time"

const demoOwnerID = "00000000-0000-0000-0000-000000000001"

type qrCode struct {
	ID            string     `json:"id"`
	OwnerID       string     `json:"-"`
	Token         string     `json:"token"`
	TargetURL     string     `json:"targetUrl"`
	NormalizedURL string     `json:"normalizedUrl"`
	ShortURL      string     `json:"shortUrl"`
	QRImageURL    string     `json:"qrImageUrl"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type createQRRequest struct {
	TargetURL string     `json:"targetUrl"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type updateQRRequest struct {
	TargetURL *string    `json:"targetUrl"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type redirectCacheEntry struct {
	TargetURL string     `json:"targetUrl"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

func (q qrCode) withURLs(baseURL string) qrCode {
	q.ShortURL = baseURL + "/r/" + q.Token
	q.QRImageURL = baseURL + "/api/qr/" + q.Token + "/image"
	return q
}
