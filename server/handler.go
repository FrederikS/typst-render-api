package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
)

var templateNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Handler struct {
	Config *Config
	Logger *slog.Logger
}

type RenderRequest map[string]interface{}

func NewHandler(cfg *Config) *Handler {
	return &Handler{
		Config: cfg,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}

func (h *Handler) RenderTemplate(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if !templateNameRegex.MatchString(name) {
		http.Error(w, "invalid template name", http.StatusBadRequest)
		return
	}

	templatePath := filepath.Join(h.Config.TemplateDir, name+".typ")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	var reqBody RenderRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	staticDataPath := filepath.Join(h.Config.TemplateDir, name+".json")
	staticData := make(map[string]interface{})
	if _, err := os.Stat(staticDataPath); err == nil {
		data, err := os.ReadFile(staticDataPath)
		if err != nil {
			h.Logger.Error("failed to read static data", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(data, &staticData); err != nil {
			h.Logger.Error("failed to parse static data", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	mergedData := mergeData(staticData, reqBody)

	jsonData, err := json.Marshal(mergedData)
	if err != nil {
		h.Logger.Error("failed to marshal data", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tmpDir, err := os.MkdirTemp("", "typst-*")
	if err != nil {
		h.Logger.Error("failed to create temp dir", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	// outputPath := filepath.Join(tmpDir, "output.pdf")
	outputPath := filepath.Join(tmpDir, "output.pdf")

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	h.Logger.Debug("json:", "data", string(jsonData))

	cmd := exec.CommandContext(ctx, h.Config.TypstPath, "compile",
		templatePath, outputPath,
		"--input", "data="+string(jsonData),
		"--root", h.Config.TemplateDir,
	)

	h.Logger.Debug("typst command", "args", cmd.Args)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.Logger.Error("typst failed", "error", err, "stderr", stderr.String())
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	// Check file exists and size
	info, err := os.Stat(outputPath)
	if err != nil {
		h.Logger.Error("output file not created", "error", err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}
	h.Logger.Debug("pdf file info", "size", info.Size(), "path", outputPath)

	pdfData, err := os.ReadFile(outputPath)
	if err != nil {
		h.Logger.Error("failed to read output", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.Logger.Info("Wrote pdf", "template", name)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", name))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfData)
}

func mergeData(static, request map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range static {
		result[k] = v
	}

	for k, v := range request {
		if v == nil {
			delete(result, k)
		} else if existing, exists := result[k]; exists {
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if requestMap, ok := v.(map[string]interface{}); ok {
					result[k] = mergeData(existingMap, requestMap)
				} else {
					result[k] = v
				}
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}
