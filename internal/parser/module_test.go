package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── ParseGoMod ────────────────────────────────────────────────────────────────

func TestParseGoMod_Basic(t *testing.T) {
	content := `module github.com/example/myapp

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.8.4 // indirect
)

require github.com/single/line v0.1.0
`
	f := writeTempFile(t, content)
	info, err := ParseGoMod(f)
	if err != nil {
		t.Fatalf("ParseGoMod: %v", err)
	}

	if info.Module != "github.com/example/myapp" {
		t.Errorf("Module = %q, want github.com/example/myapp", info.Module)
	}

	cases := map[string]string{
		"github.com/gin-gonic/gin":   "v1.9.1",
		"github.com/stretchr/testify": "v1.8.4",
		"github.com/single/line":     "v0.1.0",
	}
	for pkg, want := range cases {
		if got := info.Require[pkg]; got != want {
			t.Errorf("Require[%q] = %q, want %q", pkg, got, want)
		}
	}
}

func TestParseGoMod_RealGoMod(t *testing.T) {
	// 解析项目根目录的 go.mod（从 internal/parser 向上两层）
	path := filepath.Join("..", "..", "go.mod")
	if _, err := os.Stat(path); err != nil {
		t.Skip("go.mod not found, skipping")
	}
	info, err := ParseGoMod(path)
	if err != nil {
		t.Fatalf("ParseGoMod real: %v", err)
	}
	if info.Module == "" {
		t.Error("Module should not be empty")
	}
	if len(info.Require) == 0 {
		t.Error("Require should not be empty")
	}
	t.Logf("module=%s, %d dependencies", info.Module, len(info.Require))
}

func TestParseGoMod_NotFound(t *testing.T) {
	_, err := ParseGoMod("/nonexistent/path/go.mod")
	if err == nil {
		t.Error("expected error for non-existent go.mod")
	}
}

// ── escapeModulePath ──────────────────────────────────────────────────────────

func TestEscapeModulePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"github.com/foo/bar", "github.com/foo/bar"},
		{"github.com/BurntSushi/toml", "github.com/!burnt!sushi/toml"},
		{"github.com/Azure/go-autorest", "github.com/!azure/go-autorest"},
		{"all-lowercase", "all-lowercase"},
	}
	for _, tc := range cases {
		if got := escapeModulePath(tc.in); got != tc.want {
			t.Errorf("escapeModulePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── ResolvePackageDir ─────────────────────────────────────────────────────────

func TestResolvePackageDir(t *testing.T) {
	info := &ModuleInfo{
		Module: "github.com/example/myapp",
		Require: map[string]string{
			"github.com/gin-gonic/gin":           "v1.9.1",
			"github.com/gin-contrib/cors":        "v1.5.0",
			"github.com/stretchr/testify":        "v1.8.4",
		},
	}
	cacheDir := "/go/pkg/mod"

	cases := []struct {
		importPath string
		wantSuffix string
		wantFound  bool
	}{
		{
			"github.com/gin-gonic/gin",
			"github.com/gin-gonic/gin@v1.9.1",
			true,
		},
		{
			"github.com/gin-gonic/gin/render",
			"github.com/gin-gonic/gin@v1.9.1/render",
			true,
		},
		{
			"github.com/gin-contrib/cors",
			"github.com/gin-contrib/cors@v1.5.0",
			true,
		},
		{
			"github.com/unknown/pkg",
			"",
			false,
		},
	}

	for _, tc := range cases {
		dir, ok := ResolvePackageDir(tc.importPath, info, cacheDir)
		if ok != tc.wantFound {
			t.Errorf("ResolvePackageDir(%q) found=%v, want %v", tc.importPath, ok, tc.wantFound)
			continue
		}
		if tc.wantFound && !strings.HasSuffix(filepath.ToSlash(dir), tc.wantSuffix) {
			t.Errorf("ResolvePackageDir(%q) = %q, want suffix %q", tc.importPath, dir, tc.wantSuffix)
		}
	}
}

// ── ModuleCacheDir ────────────────────────────────────────────────────────────

func TestModuleCacheDir_NotEmpty(t *testing.T) {
	dir := ModuleCacheDir()
	if dir == "" {
		t.Error("ModuleCacheDir() should return a non-empty path")
	}
	t.Logf("GOMODCACHE = %s", dir)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "go.mod")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	f.Close()
	return f.Name()
}
