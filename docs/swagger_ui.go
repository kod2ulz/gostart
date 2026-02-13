package docs

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/kod2ulz/gostart/contracts"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>{{.Title}} - Swagger UI</title>
	<link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css" />
	<style>
		html {
			box-sizing: border-box;
			overflow: -moz-scrollbars-vertical;
			overflow-y: scroll;
		}
		*, *:before, *:after {
			box-sizing: inherit;
		}
		body {
			margin:0;
			padding:0;
		}
	</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
	<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
	<script>
		window.onload = function() {
			window.ui = SwaggerUIBundle({
				url: "{{.SpecURL}}",
				dom_id: '#swagger-ui',
				deepLinking: true,
				presets: [
					SwaggerUIBundle.presets.apis,
					SwaggerUIStandalonePreset
				],
				plugins: [
					SwaggerUIBundle.plugins.DownloadUrl
				],
				layout: "StandaloneLayout"
			});
		};
	</script>
</body>
</html>`

// SwaggerUIConfig holds configuration for Swagger UI
type SwaggerUIConfig struct {
	// Title of the documentation page
	Title string

	// SpecURL is the URL to the OpenAPI specification (JSON or YAML)
	SpecURL string

	// BasePath is the base path for the Swagger UI
	// Default: /docs
	BasePath string
}

// DefaultSwaggerUIConfig returns default Swagger UI configuration
func DefaultSwaggerUIConfig() SwaggerUIConfig {
	return SwaggerUIConfig{
		Title:    "API Documentation",
		SpecURL:  "/swagger.json",
		BasePath: "/docs",
	}
}

// SwaggerUIHandler returns an HTTP handler that serves Swagger UI
func SwaggerUIHandler(config SwaggerUIConfig) http.HandlerFunc {
	if config.Title == "" {
		config.Title = "API Documentation"
	}
	if config.SpecURL == "" {
		config.SpecURL = "/swagger.json"
	}

	tmpl, err := template.New("swagger").Parse(swaggerUIHTML)
	if err != nil {
		panic(fmt.Errorf("failed to parse swagger UI template: %w", err))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		data := map[string]string{
			"Title":   config.Title,
			"SpecURL": config.SpecURL,
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render Swagger UI", http.StatusInternalServerError)
		}
	}
}

// SwaggerUIMiddleware wraps SwaggerUIHandler for use with framework RequestContext
func SwaggerUIMiddleware(config SwaggerUIConfig) func(contracts.RequestContext) {
	handler := SwaggerUIHandler(config)

	return func(ctx contracts.RequestContext) {
		handler(ctx.Writer(), ctx.Request())
	}
}

// SwaggerSpecHandler returns a handler that serves the OpenAPI specification
func SwaggerSpecHandler(generator *Generator, format OutputFormat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spec, err := generator.Export(format)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate specification: %v", err), http.StatusInternalServerError)
			return
		}

		// Set appropriate content type
		switch format {
		case FormatYAML:
			w.Header().Set("Content-Type", "application/x-yaml")
		case FormatJSON:
			w.Header().Set("Content-Type", "application/json")
		default:
			w.Header().Set("Content-Type", "application/json")
		}

		w.WriteHeader(http.StatusOK)
		w.Write(spec)
	}
}

// SwaggerSpecMiddleware wraps SwaggerSpecHandler for use with framework RequestContext
func SwaggerSpecMiddleware(generator *Generator, format OutputFormat) func(contracts.RequestContext) {
	handler := SwaggerSpecHandler(generator, format)

	return func(ctx contracts.RequestContext) {
		handler(ctx.Writer(), ctx.Request())
	}
}

// SetupSwaggerUI sets up complete Swagger UI with OpenAPI spec endpoints
// This is a convenience function that registers all necessary routes
//
// Example usage:
//
//	generator := docs.NewGenerator("http://localhost:8080", "My API", "1.0.0")
//	// Register your routes with the generator...
//
//	// Setup Swagger UI
//	docs.SetupSwaggerUI(router, generator, docs.SwaggerUIConfig{
//		Title:    "My API Documentation",
//		SpecURL:  "/api-spec.json",
//		BasePath: "/docs",
//	})
func SetupSwaggerUI(router contracts.Router, generator *Generator, config SwaggerUIConfig) {
	if config.SpecURL == "" {
		config.SpecURL = "/swagger.json"
	}
	if config.BasePath == "" {
		config.BasePath = "/docs"
	}

	// Register Swagger UI
	router.GET(config.BasePath, SwaggerUIMiddleware(config))

	// Register JSON spec
	router.GET(config.SpecURL, SwaggerSpecMiddleware(generator, FormatJSON))

	// Also register YAML spec if different URL is used
	yamlURL := config.SpecURL
	if config.SpecURL == "/swagger.json" {
		yamlURL = "/swagger.yaml"
	}
	router.GET(yamlURL, SwaggerSpecMiddleware(generator, FormatYAML))
}

// Example usage showing how to integrate Swagger UI with your application
func ExampleSwaggerUI() {
	// This is示例 pseudo-code showing the usage pattern
	/*
		// Create generator
		generator := docs.NewGenerator("http://localhost:8080", "My API", "1.0.0")

		// Register your routes and handlers
		// generator.RegisterRoute("GET", "/users", getUsersHandler, ...)

		// Option 1: Use the convenience function
		docs.SetupSwaggerUI(app.Router(), generator, docs.DefaultSwaggerUIConfig())

		// Option 2: Manual setup
		config := docs.SwaggerUIConfig{
			Title:    "My Application API",
			SpecURL:  "/api-docs.json",
			BasePath: "/api/docs",
		}

		app.Router().GET("/api/docs", docs.SwaggerUIMiddleware(config))
		app.Router().GET("/api-docs.json", docs.SwaggerSpecMiddleware(generator, docs.FormatJSON))
		app.Router().GET("/api-docs.yaml", docs.SwaggerSpecMiddleware(generator, docs.FormatYAML))

		// Now users can visit http://localhost:8080/api/docs to see Swagger UI
	*/
}
