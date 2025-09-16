package openapi

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

// AccessControlConfig defines the configuration for OpenAPI documentation access control
type AccessControlConfig struct {
	// DefaultAccess controls the default access behavior
	DefaultAccess DefaultAccess `json:"defaultAccess"`

	// AllowedCIDRs contains the list of CIDR blocks that are allowed access
	AllowedCIDRs []string `json:"allowedCIDRs"`

	// DeniedCIDRs contains the list of CIDR blocks that are denied access
	DeniedCIDRs []string `json:"deniedCIDRs"`

	// EnableRateLimiter enables rate limiting for documentation access
	EnableRateLimiter bool `json:"enableRateLimiter"`

	// RateLimitRequests is the number of requests allowed per window
	RateLimitRequests int `json:"rateLimitRequests,omitempty"`

	// RateLimitWindow is the time window for rate limiting in seconds
	RateLimitWindow int `json:"rateLimitWindow,omitempty"`

	// EnableAuditLogging enables audit logging for documentation access
	EnableAuditLogging bool `json:"enableAuditLogging"`

	// AuditLogPath is the path to store audit logs
	AuditLogPath string `json:"auditLogPath,omitempty"`
}

// DefaultAccess defines the default access behavior
type DefaultAccess string

const (
	// AccessNonPublic allows access from non-public IPs by default
	AccessNonPublic DefaultAccess = "non-public"

	// AccessDenyAll denies access by default
	AccessDenyAll DefaultAccess = "deny-all"

	// AccessAllowAll allows access by default
	AccessAllowAll DefaultAccess = "allow-all"
)

// AccessController handles access control for OpenAPI documentation
type AccessController struct {
	config       *AccessControlConfig
	allowedNets  []*net.IPNet
	deniedNets   []*net.IPNet
	rateLimiter  *RateLimiter
	auditLogger  *AuditLogger
	mu           sync.RWMutex
}

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	requests map[string]int
	window   int
	mu       sync.Mutex
}

// AuditLogger handles audit logging
type AuditLogger struct {
	enabled bool
	path    string
}

// NewAccessController creates a new access controller
func NewAccessController(config *AccessControlConfig) (*AccessController, error) {
	if config == nil {
		config = &AccessControlConfig{
			DefaultAccess: AccessNonPublic,
		}
	}

	ac := &AccessController{
		config: config,
	}

	// Parse allowed CIDRs
	if err := ac.parseCIDRs(); err != nil {
		return nil, fmt.Errorf("failed to parse CIDRs: %w", err)
	}

	// Initialize rate limiter if enabled
	if config.EnableRateLimiter {
		if config.RateLimitRequests <= 0 {
			config.RateLimitRequests = 100 // default
		}
		if config.RateLimitWindow <= 0 {
			config.RateLimitWindow = 60 // default 1 minute
		}
		ac.rateLimiter = &RateLimiter{
			requests: make(map[string]int),
			window:   config.RateLimitWindow,
		}
	}

	// Initialize audit logger if enabled
	if config.EnableAuditLogging {
		ac.auditLogger = &AuditLogger{
			enabled: true,
			path:    config.AuditLogPath,
		}
	}

	return ac, nil
}

// parseCIDRs parses the CIDR blocks in the configuration
func (ac *AccessController) parseCIDRs() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Parse allowed CIDRs
	for _, cidr := range ac.config.AllowedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid allowed CIDR %s: %w", cidr, err)
		}
		ac.allowedNets = append(ac.allowedNets, ipNet)
	}

	// Parse denied CIDRs
	for _, cidr := range ac.config.DeniedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid denied CIDR %s: %w", cidr, err)
		}
		ac.deniedNets = append(ac.deniedNets, ipNet)
	}

	return nil
}

// Middleware returns an HTTP middleware that enforces access control
func (ac *AccessController) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			clientIP := ac.getClientIP(r)

			// Check access
			if !ac.isAllowed(clientIP) {
				ac.logAccess(r, clientIP, false, "access denied")
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			// Check rate limit
			if ac.rateLimiter != nil {
				if !ac.rateLimiter.isAllowed(clientIP) {
					ac.logAccess(r, clientIP, false, "rate limit exceeded")
					http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			}

			// Log successful access
			ac.logAccess(r, clientIP, true, "access granted")

			// Serve the request
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request
func (ac *AccessController) getClientIP(r *http.Request) string {
	// Check for forwarded IP first
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for real IP
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Use remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isAllowed checks if an IP is allowed access
func (ac *AccessController) isAllowed(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// Check denied CIDRs first (deny takes precedence)
	for _, deniedNet := range ac.deniedNets {
		if deniedNet.Contains(ip) {
			return false
		}
	}

	// Check allowed CIDRs
	if len(ac.allowedNets) > 0 {
		for _, allowedNet := range ac.allowedNets {
			if allowedNet.Contains(ip) {
				return true
			}
		}
		return false
	}

	// Default access behavior
	switch ac.config.DefaultAccess {
	case AccessNonPublic:
		return !ac.isPublicIP(ip)
	case AccessAllowAll:
		return true
	case AccessDenyAll:
		return false
	default:
		return false
	}
}

// isPublicIP checks if an IP is a public IP address
func (ac *AccessController) isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return false
	}

	if ip4 := ip.To4(); ip4 != nil {
		// Private IPv4 ranges
		switch {
		case ip4[0] == 10:
			return false // 10.0.0.0/8
		case ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127:
			return false // 100.64.0.0/10 (Tailscale and CGNAT)
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false // 172.16.0.0/12
		case ip4[0] == 192 && ip4[1] == 168:
			return false // 192.168.0.0/16
		case ip4[0] == 169 && ip4[1] == 254:
			return false // 169.254.0.0/16 (link-local)
		default:
			return true
		}
	}

	// For IPv6, check if it's not private
	return !ip.IsPrivate()
}

// isAllowed checks if the rate limit is not exceeded for an IP
func (rl *RateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	count := rl.requests[ip]
	if count >= rl.window {
		return false
	}

	rl.requests[ip] = count + 1
	return true
}

// reset resets the rate limiter (should be called periodically)
func (rl *RateLimiter) reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requests = make(map[string]int)
}

// logAccess logs access attempts
func (ac *AccessController) logAccess(r *http.Request, ip string, allowed bool, reason string) {
	if ac.auditLogger == nil || !ac.auditLogger.enabled {
		return
	}

	// In a real implementation, this would write to a log file or database
	// For now, we'll just format the log entry
	logEntry := fmt.Sprintf("Access Log - IP: %s, Method: %s, Path: %s, Allowed: %v, Reason: %s, Time: %s",
		ip, r.Method, r.URL.Path, allowed, reason, r.Header.Get("X-Request-Id"))

	// Here you would write to the actual log file
	_ = logEntry // placeholder for actual logging
}

// DefaultAccessControlConfig returns a default access control configuration
func DefaultAccessControlConfig() *AccessControlConfig {
	return &AccessControlConfig{
		DefaultAccess:      AccessNonPublic,
		EnableRateLimiter: false,
		EnableAuditLogging: false,
	}
}

// DevelopmentAccessControlConfig returns a development-friendly configuration
func DevelopmentAccessControlConfig() *AccessControlConfig {
	return &AccessControlConfig{
		DefaultAccess:      AccessAllowAll,
		EnableRateLimiter:  false,
		EnableAuditLogging: false,
	}
}

// ProductionAccessControlConfig returns a production-ready configuration
func ProductionAccessControlConfig() *AccessControlConfig {
	return &AccessControlConfig{
		DefaultAccess:      AccessNonPublic,
		AllowedCIDRs:       []string{"10.0.0.0/8", "100.64.0.0/10", "172.16.0.0/12", "192.168.0.0/16"},
		EnableRateLimiter:  true,
		RateLimitRequests:  60,
		RateLimitWindow:    60,
		EnableAuditLogging: true,
	}
}

// StrictAccessControlConfig returns a strict security configuration
func StrictAccessControlConfig() *AccessControlConfig {
	return &AccessControlConfig{
		DefaultAccess:      AccessDenyAll,
		AllowedCIDRs:       []string{"192.168.1.0/24"}, // Only specific subnet
		EnableRateLimiter:  true,
		RateLimitRequests:  30,
		RateLimitWindow:    60,
		EnableAuditLogging: true,
	}
}