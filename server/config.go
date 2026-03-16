package main

import (
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	TemplateDir string
	TypstPath   string
	Port        string
}

func LoadConfig() (*Config, error) {
	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		return nil, errors.New("TEMPLATE_DIR environment variable is required")
	}

	absPath, err := filepath.Abs(templateDir)
	if err != nil {
		return nil, errors.New("invalid TEMPLATE_DIR path")
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, errors.New("TEMPLATE_DIR does not exist")
	}
	if !info.IsDir() {
		return nil, errors.New("TEMPLATE_DIR must be a directory")
	}

	typstPath := os.Getenv("TYPST_PATH")
	if typstPath == "" {
		typstPath = "/usr/local/bin/typst"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		TemplateDir: absPath,
		TypstPath:   typstPath,
		Port:        port,
	}, nil
}
