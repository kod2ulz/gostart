package api

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"github.com/kod2ulz/gostart/contracts"
)

// DebugTokenMiddleware creates middleware that requires a valid debug token
// The token should be provided via Bearer authentication
func DebugTokenMiddleware(validToken string) MiddlewareFunc {
	return func(ctx contracts.RequestContext) (bool, error) {
		req := ctx.Request()

		// Get Authorization header
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			ErrorResponse(ctx, "UNAUTHORIZED", "Missing Authorization header", http.StatusUnauthorized)
			return false, nil
		}

		// Check Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			ErrorResponse(ctx, "INVALID_TOKEN_FORMAT", "Authorization header must be 'Bearer <token>'", http.StatusUnauthorized)
			return false, nil
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != validToken {
			ErrorResponse(ctx, "INVALID_TOKEN", "Invalid debug token", http.StatusUnauthorized)
			return false, nil
		}

		// Token is valid, continue to next handler
		return true, nil
	}
}

// LocalNetworkOnlyMiddleware creates middleware that restricts access to local network only
// Allows requests from:
// - localhost (127.0.0.1, ::1)
// - local network IPs (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
func LocalNetworkOnlyMiddleware() MiddlewareFunc {
	return func(ctx contracts.RequestContext) (bool, error) {
		req := ctx.Request()

		// Get client IP from RemoteAddr
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			ErrorResponse(ctx, "FORBIDDEN", "Unable to determine client IP", http.StatusForbidden)
			return false, nil
		}

		// Check if the IP is from localhost or local network
		if !isLocalNetwork(host) {
			ErrorResponse(ctx, "FORBIDDEN", "Access allowed from local network only", http.StatusForbidden)
			return false, nil
		}

		// IP is allowed, continue
		return true, nil
	}
}

// isLocalNetwork checks if an IP is from localhost or local network
func isLocalNetwork(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check for localhost
	if ip.IsLoopback() {
		return true
	}

	// Check for private network ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7", // IPv6 unique local addresses
	}

	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// GenerateDebugToken generates a random debug token
func GenerateDebugToken() (string, error) {
	bytes := make([]byte, 32) // 256-bit token
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// MustGenerateDebugToken generates a debug token or panics
func MustGenerateDebugToken() string {
	token, err := GenerateDebugToken()
	if err != nil {
		panic("failed to generate debug token: " + err.Error())
	}
	return token
}
