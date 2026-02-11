package api

import (
	"net/http"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/contracts"
)

// ConfigDebugHandler returns a handler that dumps the current application configuration
// This should ONLY be used in development/staging, never in production
func ConfigDebugHandler() HandlerFunc {
	return func(ctx contracts.RequestContext) {
		// Get the actual HTTP request
		req := ctx.Request()

		// Check for format query parameter (json, yaml, env)
		format := req.URL.Query().Get("format")
		if format == "" {
			format = "yaml" // Default to YAML for browser viewing
		}

		// Check for mode query parameter (cached, static, effective)
		mode := req.URL.Query().Get("mode")
		if mode == "" {
			mode = "effective" // Default to showing everything
		}

		// Get the appropriate dump
		var err error
		switch mode {
		case "cached":
			// Show only cached values (Vault, DB)
			err = config.DumpCachedConfig(config.ConfigFormat(format))
		case "static":
			// Show only static values (YAML, ENV)
			err = config.DumpConfig(config.ConfigFormat(format))
		case "effective":
			// Show everything
			dump, dumpErr := config.CollectEffectiveConfig()
			if dumpErr != nil {
				ErrorResponse(ctx, "COLLECTION_FAILED", dumpErr.Error(), http.StatusInternalServerError)
				return
			}

			switch config.ConfigFormat(format) {
			case config.FormatJSON:
				err = dump.PrintJSON()
			case config.FormatYAML:
				err = dump.PrintYAML()
			case config.FormatENV:
				err = dump.PrintENV()
			default:
				ErrorResponse(ctx, "INVALID_FORMAT", "Format must be 'json', 'yaml', or 'env'", http.StatusBadRequest)
				return
			}
		default:
			ErrorResponse(ctx, "INVALID_MODE", "Mode must be 'cached', 'static', or 'effective'", http.StatusBadRequest)
			return
		}

		if err != nil {
			ErrorResponse(ctx, "DUMP_FAILED", err.Error(), http.StatusInternalServerError)
		}
	}
}

// ConfigCacheStatsHandler returns cache statistics
func ConfigCacheStatsHandler() HandlerFunc {
	return func(ctx contracts.RequestContext) {
		type CacheStats struct {
			Source string `json:"source"`
			Size   int    `json:"size"`
		}

		stats := []CacheStats{
			{Source: "vault", Size: config.Vault.CacheSize()},
			{Source: "database", Size: config.DB.CacheSize()},
		}

		SuccessResponse(ctx, stats)
	}
}

// RegisterConfigDebugEndpoints registers configuration debug endpoints on a router
// WARNING: Only use this in development/staging, never in production!
//
// Usage:
//
//	router.GET("/debug/config", api.ConfigDebugHandler())
//	router.GET("/debug/config/stats", api.ConfigCacheStatsHandler())
func RegisterConfigDebugEndpoints(router Router) {
	// Main config dump endpoint
	// GET /debug/config?mode=effective&format=yaml
	// modes: cached, static, effective
	// formats: json, yaml, env
	router.GET("/debug/config", ConfigDebugHandler())

	// Cache statistics endpoint
	// GET /debug/config/stats
	router.GET("/debug/config/stats", ConfigCacheStatsHandler())
}

// RegisterSecureConfigDebugEndpoints registers configuration debug endpoints with security
// Requires both local network access AND a valid bearer token
//
// Usage in main():
//
//	debugToken := api.MustGenerateDebugToken()
//	log.Printf("Debug token: %s", debugToken)
//	log.Printf("Use: curl -H 'Authorization: Bearer %s' http://localhost:8080/debug/config", debugToken)
//
//	api.RegisterSecureConfigDebugEndpoints(router, debugToken)
func RegisterSecureConfigDebugEndpoints(router Router, debugToken string) {
	// Apply security middleware
	options := []RouteOption{
		RouteMiddleware{Middleware: LocalNetworkOnlyMiddleware()},
		RouteMiddleware{Middleware: DebugTokenMiddleware(debugToken)},
	}

	// Register endpoints with middleware
	router.GET("/debug/config", ConfigDebugHandler(), options...)
	router.GET("/debug/config/stats", ConfigCacheStatsHandler(), options...)
}
