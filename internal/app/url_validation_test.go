package app

import (
	"errors"
	"testing"
)

func TestNormalizeTargetURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr error
	}{
		{
			name: "normalizes scheme and host",
			raw:  " HTTPS://Example.COM/path?q=1 ",
			want: "https://example.com/path?q=1",
		},
		{
			name:    "rejects missing scheme",
			raw:     "example.com/path",
			wantErr: errInvalidURL,
		},
		{
			name:    "rejects non http scheme",
			raw:     "ftp://example.com/file",
			wantErr: errInvalidURL,
		},
		{
			name:    "rejects credentials",
			raw:     "https://user:pass@example.com",
			wantErr: errBlockedURL,
		},
		{
			name:    "rejects localhost",
			raw:     "http://localhost:8080",
			wantErr: errBlockedURL,
		},
		{
			name:    "rejects private ipv4",
			raw:     "http://192.168.1.10",
			wantErr: errBlockedURL,
		},
		{
			name:    "rejects link local metadata ip",
			raw:     "http://169.254.169.254/latest/meta-data",
			wantErr: errBlockedURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTargetURL(tt.raw)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
