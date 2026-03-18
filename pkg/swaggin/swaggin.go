// Package swaggin integrates a swag3-generated OpenAPI spec into a Gin application.
//
// Typical usage — Swagger UI (default):
//
//	swaggin.Register(r, swaggin.Options{SpecFile: "openapi.json"})
//	// GET /openapi.json  → raw spec
//	// GET /docs          → Swagger UI
//
// Switch to Redoc:
//
//	swaggin.Register(r, swaggin.Options{
//	    SpecFile: "openapi.json",
//	    Renderer: swaggin.Redoc,
//	})
//	// GET /docs  → Redoc
//
// Serve both at separate paths:
//
//	swaggin.Register(r, swaggin.Options{
//	    SpecFile:  "openapi.json",
//	    UIPath:    "/docs",          // Swagger UI
//	    RedocPath: "/redoc",         // Redoc
//	})
//
// Enable CORS (e.g. for standalone Swagger editors or separate frontends):
//
//	swaggin.Register(r, swaggin.Options{
//	    SpecFile:  "openapi.json",
//	    AllowCORS: true,             // Access-Control-Allow-Origin: *
//	})
//
//	swaggin.Register(r, swaggin.Options{
//	    SpecFile:    "openapi.json",
//	    AllowCORS:   true,
//	    CORSOrigin:  "https://editor.swagger.io", // restrict to a specific origin
//	})
package swaggin

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Renderer selects the UI library rendered at UIPath.
type Renderer string

const (
	// SwaggerUI renders Swagger UI at UIPath (default).
	SwaggerUI Renderer = "swagger-ui"
	// Redoc renders Redoc at UIPath.
	Redoc Renderer = "redoc"
	// Fdoc renders Fdoc (@braydenyang/fdoc) at UIPath.
	Fdoc Renderer = "fdoc"
)

const (
	defaultJSONPath = "/openapi.json"
	defaultUIPath   = "/docs"
	defaultTitle    = "API Documentation"
)

// Options configures the routes registered by Register.
type Options struct {
	// SpecFile is the filesystem path to the OpenAPI JSON spec produced by swag3.
	// The file is read on every request, so hot-reload works during development.
	// One of SpecFile or SpecContent must be provided.
	SpecFile string

	// SpecContent is raw OpenAPI JSON/YAML content served inline.
	// Takes precedence over SpecFile when both are set.
	SpecContent []byte

	// JSONPath is the URL path for the raw spec endpoint.
	// Defaults to "/openapi.json".
	JSONPath string

	// UIPath is the URL path for the primary UI page.
	// Defaults to "/docs". Set to "-" to disable.
	UIPath string

	// Renderer selects the UI library served at UIPath.
	// Defaults to SwaggerUI. Use Redoc to switch to Redoc.
	Renderer Renderer

	// RedocPath registers an additional Redoc route alongside the primary UI.
	// Useful when UIPath already serves Swagger UI and you also want Redoc.
	// Default "" (disabled). Set to "-" to explicitly disable.
	RedocPath string

	// FdocPath registers an additional Fdoc route alongside the primary UI.
	// Default "" (disabled). Set to "-" to explicitly disable.
	FdocPath string

	// Title is the HTML page title. Defaults to "API Documentation".
	Title string

	// AllowCORS adds CORS response headers to every swaggin-registered route
	// (spec endpoint and UI pages).  Enable this when the spec is fetched by a
	// browser running on a different origin — for example, a standalone Swagger
	// editor, a separate frontend app, or Postman web.
	//
	// When true, the following headers are set on every response and an
	// OPTIONS preflight handler is registered for each route:
	//
	//	Access-Control-Allow-Origin:  <CORSOrigin>  (defaults to "*")
	//	Access-Control-Allow-Methods: GET, OPTIONS
	//	Access-Control-Allow-Headers: Origin, Accept, Content-Type, Authorization
	//	Access-Control-Max-Age:       86400
	AllowCORS bool

	// CORSOrigin is the value written into Access-Control-Allow-Origin.
	// Defaults to "*" when AllowCORS is true and CORSOrigin is empty.
	// Set a specific origin (e.g. "https://editor.swagger.io") to restrict access.
	CORSOrigin string
}

const defaultFaviconPath = "/favicon.ico"

func (o *Options) applyDefaults() {
	if o.JSONPath == "" {
		o.JSONPath = defaultJSONPath
	}
	if o.UIPath == "" {
		o.UIPath = defaultUIPath
	}
	if o.Title == "" {
		o.Title = defaultTitle
	}
	if o.Renderer == "" {
		o.Renderer = SwaggerUI
	}

}

// Register attaches the spec and UI routes to r.
//
// Routes registered with default options:
//
//	GET /openapi.json  → raw JSON spec
//	GET /docs          → Swagger UI  (or Redoc if Renderer=Redoc)
//
// When RedocPath is set, an additional Redoc route is also registered.
// When AllowCORS is true, an OPTIONS preflight handler is registered alongside
// every GET route and CORS headers are injected into every response.
func Register(r gin.IRouter, opts Options) {
	opts.applyDefaults()

	registerRoute(r, opts.JSONPath, SpecHandler(opts), opts)

	if opts.UIPath != "-" {
		registerRoute(r, opts.UIPath, UIHandler(opts), opts)
	}

	if opts.RedocPath != "" && opts.RedocPath != "-" {
		registerRoute(r, opts.RedocPath, RedocHandler(opts), opts)
	}

	if opts.FdocPath != "" && opts.FdocPath != "-" {
		registerRoute(r, opts.FdocPath, FdocHandler(opts), opts)
	}

}

// registerRoute registers a GET handler and, when CORS is enabled, also
// registers an OPTIONS preflight handler for the same path.
func registerRoute(r gin.IRouter, path string, h gin.HandlerFunc, opts Options) {
	r.GET(path, withCORS(h, opts))
	if opts.AllowCORS {
		r.OPTIONS(path, preflightHandler(opts))
	}
}

// SpecHandler returns a HandlerFunc that serves the raw OpenAPI JSON spec.
//
//	r.GET("/api-docs/openapi.json", swaggin.SpecHandler(opts))
func SpecHandler(opts Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := resolveSpec(opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("failed to load OpenAPI spec: %s", err),
			})
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	}
}

// UIHandler returns a HandlerFunc that serves the UI selected by opts.Renderer
// (SwaggerUI by default, Redoc when opts.Renderer == Redoc).
func UIHandler(opts Options) gin.HandlerFunc {
	opts.applyDefaults()
	return func(c *gin.Context) {
		var html string
		switch opts.Renderer {
		case Redoc:
			html = redocHTML(opts)
		case Fdoc:
			html = fdocHTML(opts)
		default:
			html = swaggerUIHTML(opts)
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// RedocHandler returns a HandlerFunc that always serves Redoc, regardless of
// the Renderer field. Use this to register Redoc at a custom path.
//
//	r.GET("/redoc", swaggin.RedocHandler(opts))
func RedocHandler(opts Options) gin.HandlerFunc {
	opts.applyDefaults()
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(redocHTML(opts)))
	}
}

// FdocHandler returns a HandlerFunc that always serves Fdoc, regardless of
// the Renderer field. Use this to register Fdoc at a custom path.
//
//	r.GET("/fdoc", swaggin.FdocHandler(opts))
func FdocHandler(opts Options) gin.HandlerFunc {
	opts.applyDefaults()
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fdocHTML(opts)))
	}
}

// ── internal helpers ──────────────────────────────────────────────────────────

// withCORS wraps h to inject CORS headers when opts.AllowCORS is true.
// When AllowCORS is false the original handler is returned unchanged.
func withCORS(h gin.HandlerFunc, opts Options) gin.HandlerFunc {
	if !opts.AllowCORS {
		return h
	}
	return func(c *gin.Context) {
		setCORSHeaders(c, opts)
		h(c)
	}
}

// preflightHandler responds to OPTIONS requests with the appropriate CORS
// headers and a 204 No Content status (no body).
func preflightHandler(opts Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		setCORSHeaders(c, opts)
		c.AbortWithStatus(http.StatusNoContent)
	}
}

// setCORSHeaders writes Access-Control-* headers onto the response.
func setCORSHeaders(c *gin.Context, opts Options) {
	origin := opts.CORSOrigin
	if origin == "" {
		origin = "*"
	}
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Origin, Accept, Content-Type, Authorization")
	c.Header("Access-Control-Max-Age", "86400")
}

func resolveSpec(opts Options) ([]byte, error) {
	if len(opts.SpecContent) > 0 {
		return opts.SpecContent, nil
	}
	if opts.SpecFile != "" {
		data, err := os.ReadFile(opts.SpecFile)
		if err != nil {
			return nil, fmt.Errorf("read spec file %q: %w", opts.SpecFile, err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("neither SpecFile nor SpecContent is configured")
}

// swaggerUIHTML generates the Swagger UI page.
func swaggerUIHTML(opts Options) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
  SwaggerUIBundle({
    url: %q,
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIBundle.SwaggerUIStandalonePreset
    ],
    layout: 'BaseLayout'
  })
</script>
</body>
</html>`, opts.Title, opts.JSONPath)
}

// fdocHTML generates the Fdoc page (@braydenyang/fdoc).
func fdocHTML(opts Options) string {
	return fmt.Sprintf(`<!doctype html>
<html>
<head>
  <title>%s</title>
  <link rel="icon" type="image/svg+xml" href="https://unpkg.com/@braydenyang/fdoc@latest/dist/lib/favicon.svg">
  <link rel="stylesheet" href="https://unpkg.com/@braydenyang/fdoc@latest/dist/lib/style.css">
  <style>html, body { margin: 0; height: 100%% }</style>
</head>
<body>
  <div id="fdoc" style="height: 100vh"></div>
  <script src="https://unpkg.com/@braydenyang/fdoc@latest/dist/lib/fdoc.iife.js"></script>
  <script>
    Fdoc.mount('#fdoc', { url: window.location.origin + %q })
  </script>
</body>
</html>`, opts.Title, opts.JSONPath)
}

// redocHTML generates the Redoc page.
// Redoc is a clean, three-panel OpenAPI documentation renderer by Redocly.
func redocHTML(opts Options) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
  <redoc spec-url=%q></redoc>
  <script src="https://cdn.jsdelivr.net/npm/redoc@latest/bundles/redoc.standalone.js"></script>
</body>
</html>`, opts.Title, opts.JSONPath)
}
