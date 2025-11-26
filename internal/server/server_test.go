package server

import (
	"testing"
)

func TestConfig_SeparatePorts(t *testing.T) {
	// Test that different S3 and Web ports are recognized as separate
	config := &Config{
		S3Port:   9000,
		WebPort:  9001,
		DataPath: t.TempDir(),
	}

	srv, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if !srv.separatePorts {
		t.Error("Expected separatePorts to be true when ports differ")
	}
}

func TestConfig_CombinedPort(t *testing.T) {
	// Test that same S3 and Web ports are recognized as combined
	config := &Config{
		S3Port:   9000,
		WebPort:  9000,
		DataPath: t.TempDir(),
	}

	srv, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if srv.separatePorts {
		t.Error("Expected separatePorts to be false when ports are the same")
	}
}

func TestConfig_MetaPath(t *testing.T) {
	// Test that custom meta path is used
	tmpDir := t.TempDir()
	metaDir := t.TempDir()

	config := &Config{
		S3Port:   9000,
		WebPort:  9000,
		DataPath: tmpDir,
		MetaPath: metaDir,
	}

	_, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server with custom meta path: %v", err)
	}
}

func TestConfig_DefaultMetaPath(t *testing.T) {
	// Test that default meta path is used when not specified
	tmpDir := t.TempDir()

	config := &Config{
		S3Port:   9000,
		WebPort:  9000,
		DataPath: tmpDir,
		MetaPath: "",
	}

	_, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server with default meta path: %v", err)
	}
}
