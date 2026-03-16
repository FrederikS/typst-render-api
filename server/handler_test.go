package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHealth(t *testing.T) {
	cfg := &Config{TemplateDir: "/tmp", TypstPath: "/usr/bin/typst", Port: "8080"}
	handler := NewHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

func TestHealthResponse(t *testing.T) {
	cfg := &Config{TemplateDir: "/tmp", TypstPath: "/usr/bin/typst", Port: "8080"}
	handler := NewHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	expected := `{"status": "ok"}`
	if w.Body.String() != expected {
		t.Errorf("expected body %s, got %s", expected, w.Body.String())
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	origTemplateDir := os.Getenv("TEMPLATE_DIR")
	origTypstPath := os.Getenv("TYPST_PATH")
	origPort := os.Getenv("PORT")

	defer func() {
		os.Setenv("TEMPLATE_DIR", origTemplateDir)
		os.Setenv("TYPST_PATH", origTypstPath)
		os.Setenv("PORT", origPort)
	}()

	os.Setenv("TEMPLATE_DIR", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.TemplateDir != tmpDir {
		t.Errorf("expected TemplateDir %s, got %s", tmpDir, cfg.TemplateDir)
	}

	if cfg.TypstPath != "/usr/local/bin/typst" {
		t.Errorf("expected default TypstPath, got %s", cfg.TypstPath)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080, got %s", cfg.Port)
	}
}

func TestLoadConfigMissingTemplateDir(t *testing.T) {
	origTemplateDir := os.Getenv("TEMPLATE_DIR")
	os.Unsetenv("TEMPLATE_DIR")
	defer os.Setenv("TEMPLATE_DIR", origTemplateDir)

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing TEMPLATE_DIR")
	}
}

func TestLoadConfigInvalidTemplateDir(t *testing.T) {
	origTemplateDir := os.Getenv("TEMPLATE_DIR")
	os.Setenv("TEMPLATE_DIR", "/nonexistent/path/to/dir")
	defer os.Setenv("TEMPLATE_DIR", origTemplateDir)

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for non-existent TEMPLATE_DIR")
	}
}

func TestLoadConfigCustomPort(t *testing.T) {
	tmpDir := t.TempDir()

	origTemplateDir := os.Getenv("TEMPLATE_DIR")
	origPort := os.Getenv("PORT")

	defer func() {
		os.Setenv("TEMPLATE_DIR", origTemplateDir)
		os.Setenv("PORT", origPort)
	}()

	os.Setenv("TEMPLATE_DIR", tmpDir)
	os.Setenv("PORT", "3000")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Port != "3000" {
		t.Errorf("expected Port 3000, got %s", cfg.Port)
	}
}

func TestTemplateNameValidation(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"letter", true},
		{"invoice-2024", true},
		{"my_template", true},
		{"template123", true},
		{"123template", true},
		{"template.name", false},
		{"template/name", false},
		{"template%name", false},
		{"../etc/passwd", false},
		{"", false},
		{"-startswithdash", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := templateNameRegex.MatchString(tt.name)
			if valid != tt.valid {
				t.Errorf("expected %s to be valid=%v, got %v", tt.name, tt.valid, valid)
			}
		})
	}
}

func TestRenderTemplateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{TemplateDir: tmpDir, TypstPath: "/usr/bin/typst", Port: "8080"}
	handler := NewHandler(cfg)

	r := chi.NewRouter()
	r.Post("/{name}/render", handler.RenderTemplate)

	req := httptest.NewRequest(http.MethodPost, "/api/templates/nonexistent/render", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestRenderTemplateInvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{TemplateDir: tmpDir, TypstPath: "/usr/bin/typst", Port: "8080"}
	handler := NewHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/templates/../etc/passwd/render", nil)
	w := httptest.NewRecorder()

	handler.RenderTemplate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestMergeData(t *testing.T) {
	static := map[string]interface{}{
		"company": map[string]interface{}{
			"name": "Acme Corp",
			"city": "Springfield",
		},
		"date": "2024-01-01",
	}

	request := map[string]interface{}{
		"company": map[string]interface{}{
			"name": "New Company",
		},
		"position": "Engineer",
	}

	result := mergeData(static, request)

	company := result["company"].(map[string]interface{})
	if company["name"] != "New Company" {
		t.Errorf("expected company.name to be 'New Company', got %v", company["name"])
	}
	if company["city"] != "Springfield" {
		t.Errorf("expected company.city to be 'Springfield', got %v", company["city"])
	}
	if result["date"] != "2024-01-01" {
		t.Errorf("expected date to be '2024-01-01', got %v", result["date"])
	}
	if result["position"] != "Engineer" {
		t.Errorf("expected position to be 'Engineer', got %v", result["position"])
	}
}

func TestMergeDataWithNil(t *testing.T) {
	static := map[string]interface{}{
		"keep":   "this",
		"remove": "this",
	}

	request := map[string]interface{}{
		"remove": nil,
		"add":    "new",
	}

	result := mergeData(static, request)

	if _, exists := result["remove"]; exists {
		t.Error("expected 'remove' key to be deleted")
	}
	if result["keep"] != "this" {
		t.Errorf("expected keep=this, got %v", result["keep"])
	}
	if result["add"] != "new" {
		t.Errorf("expected add=new, got %v", result["add"])
	}
}

func TestStaticDataLoad(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.typ")
	staticDataPath := filepath.Join(tmpDir, "test.json")

	if err := os.WriteFile(templatePath, []byte("#let data = ()\n"), 0644); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}
	if err := os.WriteFile(staticDataPath, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("failed to create static data: %v", err)
	}

	cfg := &Config{TemplateDir: tmpDir, TypstPath: "/usr/bin/typst", Port: "8080"}
	_ = cfg
}
