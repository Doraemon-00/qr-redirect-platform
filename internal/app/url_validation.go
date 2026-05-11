package app

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

const maxURLLength = 2048

var (
	errInvalidURL = errors.New("invalid url")
	errBlockedURL = errors.New("blocked url")
)

func normalizeTargetURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxURLLength {
		return "", errInvalidURL
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errInvalidURL
	}

	if parsed.User != nil {
		return "", errBlockedURL
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", errBlockedURL
	}

	if ip := net.ParseIP(host); ip != nil && isPrivateOrLocalIP(ip) {
		return "", errBlockedURL
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), nil
}

func isPrivateOrLocalIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}

	blockedCIDRs := []string{
		"169.254.169.254/32",
		"0.0.0.0/8",
		"100.64.0.0/10",
		"224.0.0.0/4",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	for _, cidr := range blockedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}

	return false
}
