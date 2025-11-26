package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_WithConfigFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
s3_port: 8000
web_port: 8001
data_path: /custom/data
meta_path: /custom/meta
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.S3Port != 8000 {
		t.Errorf("Expected S3Port 8000, got %d", cfg.S3Port)
	}
	if cfg.WebPort != 8001 {
		t.Errorf("Expected WebPort 8001, got %d", cfg.WebPort)
	}
	if cfg.DataPath != "/custom/data" {
		t.Errorf("Expected DataPath /custom/data, got %s", cfg.DataPath)
	}
	if cfg.MetaPath != "/custom/meta" {
		t.Errorf("Expected MetaPath /custom/meta, got %s", cfg.MetaPath)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	envVars := []string{"EASYOSS_S3_PORT", "EASYOSS_WEB_PORT", "EASYOSS_DATA_PATH", "EASYOSS_META_PATH"}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	// Test loading without config file (should use defaults from env-default tags)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}

	if cfg.S3Port != 9000 {
		t.Errorf("Expected default S3Port 9000, got %d", cfg.S3Port)
	}
	if cfg.WebPort != 9001 {
		t.Errorf("Expected default WebPort 9001, got %d", cfg.WebPort)
	}
	if cfg.DataPath != "./data" {
		t.Errorf("Expected default DataPath ./data, got %s", cfg.DataPath)
	}
	// MetaPath should be set to DataPath/.meta when empty
	expectedMetaPath := filepath.Join(cfg.DataPath, ".meta")
	if cfg.MetaPath != expectedMetaPath {
		t.Errorf("Expected MetaPath %s, got %s", expectedMetaPath, cfg.MetaPath)
	}
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("EASYOSS_S3_PORT", "7000")
	os.Setenv("EASYOSS_WEB_PORT", "7001")
	os.Setenv("EASYOSS_DATA_PATH", "/env/data")
	os.Setenv("EASYOSS_META_PATH", "/env/meta")
	defer func() {
		os.Unsetenv("EASYOSS_S3_PORT")
		os.Unsetenv("EASYOSS_WEB_PORT")
		os.Unsetenv("EASYOSS_DATA_PATH")
		os.Unsetenv("EASYOSS_META_PATH")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config from env: %v", err)
	}

	if cfg.S3Port != 7000 {
		t.Errorf("Expected S3Port 7000, got %d", cfg.S3Port)
	}
	if cfg.WebPort != 7001 {
		t.Errorf("Expected WebPort 7001, got %d", cfg.WebPort)
	}
	if cfg.DataPath != "/env/data" {
		t.Errorf("Expected DataPath /env/data, got %s", cfg.DataPath)
	}
	if cfg.MetaPath != "/env/meta" {
		t.Errorf("Expected MetaPath /env/meta, got %s", cfg.MetaPath)
	}
}

func TestLoad_NonExistentCustomPath(t *testing.T) {
	// Try to load from a non-existent custom config path
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error when loading from non-existent custom path")
	}
}

func TestLoad_MetaPathDefault(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("EASYOSS_META_PATH")

	// Create a config file without meta_path
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
s3_port: 9000
web_port: 9001
data_path: /test/data
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// MetaPath should default to DataPath/.meta
	expectedMetaPath := "/test/data/.meta"
	if cfg.MetaPath != expectedMetaPath {
		t.Errorf("Expected MetaPath %s, got %s", expectedMetaPath, cfg.MetaPath)
	}
}
