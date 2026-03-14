package swaggin_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shuttlefy/go-openapi3-swag/pkg/swaggin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRouter() *gin.Engine {
	return gin.New()
}

func do(r http.Handler, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// ────────────────────────────────────────────────
// SpecHandler / inline content
// ────────────────────────────────────────────────

func TestSpecHandler_InlineContent(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.3","info":{"title":"Test","version":"1.0"}}`)

	r := newRouter()
	swaggin.Register(r, swaggin.Options{SpecContent: spec})

	w := do(r, "GET", "/openapi.json")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if body := w.Body.String(); !strings.Contains(body, `"openapi"`) {
		t.Errorf("body missing openapi field: %s", body)
	}
}

func TestSpecHandler_FileContent(t *testing.T) {
	specData := []byte(`{"openapi":"3.0.3","info":{"title":"FileTest","version":"0.1"}}`)

	dir := t.TempDir()
	path := filepath.Join(dir, "openapi.json")
	if err := os.WriteFile(path, specData, 0644); err != nil {
		t.Fatal(err)
	}

	r := newRouter()
	swaggin.Register(r, swaggin.Options{SpecFile: path})

	w := do(r, "GET", "/openapi.json")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "FileTest") {
		t.Errorf("body does not contain spec content: %s", w.Body.String())
	}
}

func TestSpecHandler_MissingSpec_Returns500(t *testing.T) {
	r := newRouter()
	// Neither SpecFile nor SpecContent → should return 500.
	swaggin.Register(r, swaggin.Options{})

	w := do(r, "GET", "/openapi.json")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestSpecHandler_NonExistentFile_Returns500(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{SpecFile: "/no/such/file.json"})

	w := do(r, "GET", "/openapi.json")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ────────────────────────────────────────────────
// Swagger UI
// ────────────────────────────────────────────────

func TestUIHandler_DefaultPath(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{SpecContent: []byte(`{}`)})

	w := do(r, "GET", "/docs")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := w.Body.String()
	for _, want := range []string{"swagger-ui", "/openapi.json", "API Documentation"} {
		if !strings.Contains(body, want) {
			t.Errorf("UI body missing %q", want)
		}
	}
}

func TestUIHandler_CustomTitle(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		Title:       "My Custom API",
	})

	w := do(r, "GET", "/docs")
	if !strings.Contains(w.Body.String(), "My Custom API") {
		t.Errorf("UI title not found in: %s", w.Body.String())
	}
}

func TestUIHandler_Disabled(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		UIPath:      "-",
	})

	w := do(r, "GET", "/docs")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (UI disabled)", w.Code)
	}
}

// ────────────────────────────────────────────────
// Custom paths
// ────────────────────────────────────────────────

func TestCustomPaths(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.3"}`)

	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: spec,
		JSONPath:    "/api/spec.json",
		UIPath:      "/api/ui",
	})

	if w := do(r, "GET", "/api/spec.json"); w.Code != http.StatusOK {
		t.Errorf("custom JSON path: status = %d", w.Code)
	}
	if w := do(r, "GET", "/api/ui"); w.Code != http.StatusOK {
		t.Errorf("custom UI path: status = %d", w.Code)
	}
	// Default paths should NOT be registered.
	if w := do(r, "GET", "/openapi.json"); w.Code != http.StatusNotFound {
		t.Errorf("default JSON path should not exist: status = %d", w.Code)
	}
}

// ────────────────────────────────────────────────
// RouterGroup support
// ────────────────────────────────────────────────

func TestRegister_RouterGroup(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.3"}`)

	r := newRouter()
	v1 := r.Group("/v1")
	swaggin.Register(v1, swaggin.Options{
		SpecContent: spec,
		JSONPath:    "/openapi.json",
		UIPath:      "/docs",
	})

	if w := do(r, "GET", "/v1/openapi.json"); w.Code != http.StatusOK {
		t.Errorf("group spec: status = %d", w.Code)
	}
	if w := do(r, "GET", "/v1/docs"); w.Code != http.StatusOK {
		t.Errorf("group UI: status = %d", w.Code)
	}
}

// ────────────────────────────────────────────────
// Standalone handlers
// ────────────────────────────────────────────────

func TestSpecHandler_Standalone(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.3"}`)
	opts := swaggin.Options{SpecContent: spec}

	r := newRouter()
	r.GET("/my-spec", swaggin.SpecHandler(opts))

	w := do(r, "GET", "/my-spec")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestUIHandler_Standalone(t *testing.T) {
	opts := swaggin.Options{JSONPath: "/spec.json", Title: "Standalone UI"}

	r := newRouter()
	r.GET("/my-ui", swaggin.UIHandler(opts))

	w := do(r, "GET", "/my-ui")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Standalone UI") {
		t.Error("title not in response")
	}
	if !strings.Contains(w.Body.String(), "/spec.json") {
		t.Error("spec URL not in response")
	}
}

// ────────────────────────────────────────────────
// Redoc renderer
// ────────────────────────────────────────────────

func TestRedoc_RendererOption(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		Renderer:    swaggin.Redoc,
		Title:       "Redoc Title",
	})

	w := do(r, "GET", "/docs")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	// Redoc-specific markers
	for _, want := range []string{"redoc", "redoc.standalone.js", "Redoc Title", "/openapi.json"} {
		if !strings.Contains(body, want) {
			t.Errorf("Redoc body missing %q", body)
		}
	}
	// Must NOT contain Swagger UI markers
	if strings.Contains(body, "swagger-ui-bundle.js") {
		t.Error("Redoc page should not include swagger-ui-bundle.js")
	}
}

func TestRedoc_ExtraRoute(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		UIPath:      "/docs",  // Swagger UI
		RedocPath:   "/redoc", // Redoc at separate path
	})

	// /docs → Swagger UI
	swaggerBody := do(r, "GET", "/docs").Body.String()
	if !strings.Contains(swaggerBody, "swagger-ui-bundle.js") {
		t.Error("/docs should render Swagger UI")
	}

	// /redoc → Redoc
	redocBody := do(r, "GET", "/redoc").Body.String()
	if !strings.Contains(redocBody, "redoc.standalone.js") {
		t.Error("/redoc should render Redoc")
	}
	if strings.Contains(redocBody, "swagger-ui-bundle.js") {
		t.Error("/redoc must not include Swagger UI")
	}
}

func TestRedoc_BothDisabled(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		UIPath:      "-", // Swagger UI disabled
		RedocPath:   "-", // Redoc disabled
	})

	if w := do(r, "GET", "/docs"); w.Code != http.StatusNotFound {
		t.Errorf("/docs: status = %d, want 404", w.Code)
	}
	if w := do(r, "GET", "/redoc"); w.Code != http.StatusNotFound {
		t.Errorf("/redoc: status = %d, want 404", w.Code)
	}
	// Spec itself must still work.
	if w := do(r, "GET", "/openapi.json"); w.Code != http.StatusOK {
		t.Errorf("/openapi.json: status = %d, want 200", w.Code)
	}
}

func TestRedocHandler_Standalone(t *testing.T) {
	opts := swaggin.Options{JSONPath: "/my-spec.json", Title: "Standalone Redoc"}

	r := newRouter()
	r.GET("/my-redoc", swaggin.RedocHandler(opts))

	w := do(r, "GET", "/my-redoc")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Standalone Redoc") {
		t.Error("title not in response")
	}
	if !strings.Contains(body, "/my-spec.json") {
		t.Error("spec URL not in response")
	}
	if !strings.Contains(body, "redoc.standalone.js") {
		t.Error("redoc script not in response")
	}
}

func TestRedoc_CustomPath(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		Renderer:    swaggin.Redoc,
		UIPath:      "/api-docs",
	})

	if w := do(r, "GET", "/api-docs"); w.Code != http.StatusOK {
		t.Errorf("custom redoc path: status = %d", w.Code)
	}
	if w := do(r, "GET", "/docs"); w.Code != http.StatusNotFound {
		t.Errorf("default /docs should not exist when UIPath is custom")
	}
}

// ────────────────────────────────────────────────
// CORS
// ────────────────────────────────────────────────

func TestCORS_Disabled_NoHeaders(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		AllowCORS:   false,
	})

	w := do(r, "GET", "/openapi.json")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS header when AllowCORS=false, got %q", got)
	}
}

func TestCORS_Enabled_WildcardOrigin(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		AllowCORS:   true,
	})

	w := do(r, "GET", "/openapi.json")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Access-Control-Allow-Methods header to be set")
	}
}

func TestCORS_Enabled_CustomOrigin(t *testing.T) {
	const origin = "https://editor.swagger.io"
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		AllowCORS:   true,
		CORSOrigin:  origin,
	})

	w := do(r, "GET", "/openapi.json")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Errorf("expected %q, got %q", origin, got)
	}
}

func TestCORS_Preflight_OPTIONS(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		AllowCORS:   true,
	})

	w := do(r, "OPTIONS", "/openapi.json")
	if w.Code != http.StatusNoContent {
		t.Errorf("preflight expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("preflight expected CORS header, got %q", got)
	}
}

func TestCORS_Preflight_NotRegistered_WhenDisabled(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		AllowCORS:   false,
	})

	w := do(r, "OPTIONS", "/openapi.json")
	// Gin returns 404 or 405 for unregistered methods; either is acceptable.
	if w.Code == http.StatusNoContent {
		t.Error("OPTIONS should not be handled when AllowCORS=false")
	}
}

func TestCORS_AppliedToUIPath(t *testing.T) {
	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecContent: []byte(`{}`),
		AllowCORS:   true,
	})

	w := do(r, "GET", "/docs")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS header missing on UI path, got %q", got)
	}
}

// ────────────────────────────────────────────────
// SpecContent takes precedence over SpecFile
// ────────────────────────────────────────────────

func TestSpecContent_TakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openapi.json")
	_ = os.WriteFile(path, []byte(`{"from":"file"}`), 0644)

	r := newRouter()
	swaggin.Register(r, swaggin.Options{
		SpecFile:    path,
		SpecContent: []byte(`{"from":"inline"}`),
	})

	w := do(r, "GET", "/openapi.json")
	if !strings.Contains(w.Body.String(), "inline") {
		t.Errorf("SpecContent should take precedence, got: %s", w.Body.String())
	}
}
