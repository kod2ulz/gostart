package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kod2ulz/gostart/contracts"
)

// Version represents an API version
type Version struct {
	Major int
	Minor int
	Patch int
}

// String returns the string representation of a version
func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares two versions
// Returns: -1 if v < other, 0 if v == other, 1 if v > other
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// ParseVersion parses a version string (e.g., "v1.2.3", "1.2.3", "v1", "1")
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")

	var v Version
	var err error

	if len(parts) >= 1 {
		v.Major, err = strconv.Atoi(parts[0])
		if err != nil {
			return Version{}, fmt.Errorf("invalid major version: %w", err)
		}
	}

	if len(parts) >= 2 {
		v.Minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return Version{}, fmt.Errorf("invalid minor version: %w", err)
		}
	}

	if len(parts) >= 3 {
		v.Patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return Version{}, fmt.Errorf("invalid patch version: %w", err)
		}
	}

	return v, nil
}

// VersionStrategy defines how version is determined
type VersionStrategy string

const (
	// URLPathStrategy extracts version from URL path (e.g., /v1/users)
	URLPathStrategy VersionStrategy = "url_path"

	// HeaderStrategy extracts version from header (e.g., X-API-Version: v1.0.0)
	HeaderStrategy VersionStrategy = "header"

	// QueryParamStrategy extracts version from query parameter (e.g., ?api_version=v1.0.0)
	QueryParamStrategy VersionStrategy = "query_param"

	// AcceptHeaderStrategy extracts version from Accept header (e.g., Accept: application/vnd.myapi.v1+json)
	AcceptHeaderStrategy VersionStrategy = "accept_header"
)

// VersionConfig configures API versioning
type VersionConfig struct {
	Strategy      VersionStrategy
	HeaderName    string // For HeaderStrategy
	QueryParam    string // For QueryParamStrategy
	DefaultVersion Version
	SupportedVersions []Version
	DeprecatedVersions map[string]Version // version -> replacement version
}

// DefaultVersionConfig returns default versioning configuration
func DefaultVersionConfig() VersionConfig {
	return VersionConfig{
		Strategy:   URLPathStrategy,
		HeaderName: "X-API-Version",
		QueryParam: "api_version",
		DefaultVersion: Version{Major: 1, Minor: 0, Patch: 0},
		SupportedVersions: []Version{
			{Major: 1, Minor: 0, Patch: 0},
		},
		DeprecatedVersions: make(map[string]Version),
	}
}

// VersionExtractor extracts version from request
type VersionExtractor struct {
	config VersionConfig
}

// NewVersionExtractor creates a new version extractor
func NewVersionExtractor(config VersionConfig) *VersionExtractor {
	return &VersionExtractor{config: config}
}

// Extract extracts the API version from the request
func (ve *VersionExtractor) Extract(ctx contracts.RequestContext) (Version, error) {
	switch ve.config.Strategy {
	case URLPathStrategy:
		return ve.extractFromPath(ctx)
	case HeaderStrategy:
		return ve.extractFromHeader(ctx)
	case QueryParamStrategy:
		return ve.extractFromQueryParam(ctx)
	case AcceptHeaderStrategy:
		return ve.extractFromAcceptHeader(ctx)
	default:
		return ve.config.DefaultVersion, nil
	}
}

// extractFromPath extracts version from URL path
func (ve *VersionExtractor) extractFromPath(ctx contracts.RequestContext) (Version, error) {
	path := ctx.Path()

	// Pattern: /v1/resource or /api/v1/resource
	re := regexp.MustCompile(`/v(\d+)(?:\.(\d+))?(?:\.(\d+))?/`)
	matches := re.FindStringSubmatch(path)

	if len(matches) == 0 {
		return ve.config.DefaultVersion, nil
	}

	versionStr := "v" + matches[1]
	if matches[2] != "" {
		versionStr += "." + matches[2]
	}
	if matches[3] != "" {
		versionStr += "." + matches[3]
	}

	return ParseVersion(versionStr)
}

// extractFromHeader extracts version from header
func (ve *VersionExtractor) extractFromHeader(ctx contracts.RequestContext) (Version, error) {
	header := ctx.Header(ve.config.HeaderName)
	if header == "" {
		return ve.config.DefaultVersion, nil
	}

	return ParseVersion(header)
}

// extractFromQueryParam extracts version from query parameter
func (ve *VersionExtractor) extractFromQueryParam(ctx contracts.RequestContext) (Version, error) {
	param := ctx.Query(ve.config.QueryParam).String()
	if param == "" {
		return ve.config.DefaultVersion, nil
	}

	return ParseVersion(param)
}

// extractFromAcceptHeader extracts version from Accept header
func (ve *VersionExtractor) extractFromAcceptHeader(ctx contracts.RequestContext) (Version, error) {
	accept := ctx.Header("Accept")
	if accept == "" {
		return ve.config.DefaultVersion, nil
	}

	// Pattern: application/vnd.myapi.v1+json
	re := regexp.MustCompile(`\.v(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:\+|$)`)
	matches := re.FindStringSubmatch(accept)

	if len(matches) == 0 {
		return ve.config.DefaultVersion, nil
	}

	versionStr := "v" + matches[1]
	if matches[2] != "" {
		versionStr += "." + matches[2]
	}
	if matches[3] != "" {
		versionStr += "." + matches[3]
	}

	return ParseVersion(versionStr)
}

// IsSupported checks if a version is supported
func (ve *VersionExtractor) IsSupported(version Version) bool {
	for _, supported := range ve.config.SupportedVersions {
		if version.Compare(supported) == 0 {
			return true
		}
	}
	return false
}

// IsDeprecated checks if a version is deprecated
func (ve *VersionExtractor) IsDeprecated(version Version) (bool, Version) {
	replacement, deprecated := ve.config.DeprecatedVersions[version.String()]
	return deprecated, replacement
}

// VersionMiddleware adds version validation middleware
func VersionMiddleware(config VersionConfig) func(contracts.RequestContext) {
	extractor := NewVersionExtractor(config)

	return func(ctx contracts.RequestContext) {
		version, err := extractor.Extract(ctx)
		if err != nil {
			ctx.JSON(400, map[string]interface{}{
				"error": "Invalid API version",
			})
			ctx.Abort()
			return
		}

		// Check if version is supported
		if !extractor.IsSupported(version) {
			ctx.JSON(404, map[string]interface{}{
				"error": fmt.Sprintf("API version %s is not supported", version.String()),
				"supported_versions": getSupportedVersionStrings(config.SupportedVersions),
			})
			ctx.Abort()
			return
		}

		// Check if version is deprecated
		if deprecated, replacement := extractor.IsDeprecated(version); deprecated {
			ctx.Header("X-API-Deprecated", "true")
			ctx.Header("X-API-Replacement-Version", replacement.String())
			ctx.Header("Warning", fmt.Sprintf("299 - \"API version %s is deprecated. Please use version %s\"", version.String(), replacement.String()))
		}

		// Store version in context
		ctx.Set("api_version", version)

		ctx.Next()
	}
}

// GetVersion retrieves the API version from context
func GetVersion(ctx contracts.RequestContext) (Version, bool) {
	v, exists := ctx.Get("api_version")
	if !exists {
		return Version{}, false
	}

	version, ok := v.(Version)
	return version, ok
}

// VersionedRouter wraps a router with version-specific routes
type VersionedRouter struct {
	versions map[string]contracts.Router
	default Version
}

// NewVersionedRouter creates a new versioned router
func NewVersionedRouter(defaultVersion Version) *VersionedRouter {
	return &VersionedRouter{
		versions: make(map[string]contracts.Router),
		defaultVersion: defaultVersion,
	}
}

// AddVersion adds a version-specific router
func (vr *VersionedRouter) AddVersion(version Version, router contracts.Router) {
	vr.versions[version.String()] = router
}

// GetRouter retrieves the router for a specific version
func (vr *VersionedRouter) GetRouter(version Version) (contracts.Router, bool) {
	router, exists := vr.versions[version.String()]
	if !exists {
		router, exists = vr.versions[vr.defaultVersion.String()]
	}
	return router, exists
}

// Helper functions

func getSupportedVersionStrings(versions []Version) []string {
	strs := make([]string, len(versions))
	for i, v := range versions {
		strs[i] = v.String()
	}
	return strs
}

// Example usage

// ExampleVersioning demonstrates how to use API versioning
func ExampleVersioning() {
	// Configure versioning
	config := VersionConfig{
		Strategy:   URLPathStrategy,
		HeaderName: "X-API-Version",
		QueryParam: "api_version",
		DefaultVersion: Version{Major: 1, Minor: 0, Patch: 0},
		SupportedVersions: []Version{
			{Major: 1, Minor: 0, Patch: 0},
			{Major: 1, Minor: 1, Patch: 0},
			{Major: 2, Minor: 0, Patch: 0},
		},
		DeprecatedVersions: map[string]Version{
			"v1.0.0": {Major: 1, Minor: 1, Patch: 0},
		},
	}

	// Example routes:
	// GET /v1/users - Uses version 1.0.0
	// GET /v2/users - Uses version 2.0.0
	// GET /users with X-API-Version: v1.1.0 header - Uses version 1.1.0
}
