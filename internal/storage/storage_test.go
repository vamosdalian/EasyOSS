package storage

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorage_CreateBucket(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test creating a bucket
	err = storage.CreateBucket("test-bucket")
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Verify bucket directory exists
	bucketPath := filepath.Join(tmpDir, "test-bucket")
	if _, err := os.Stat(bucketPath); os.IsNotExist(err) {
		t.Errorf("Bucket directory was not created")
	}

	// Test creating duplicate bucket
	err = storage.CreateBucket("test-bucket")
	if err == nil {
		t.Error("Expected error when creating duplicate bucket")
	}
}

func TestLocalStorage_DeleteBucket(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create bucket first
	err = storage.CreateBucket("test-bucket")
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Delete bucket
	err = storage.DeleteBucket("test-bucket")
	if err != nil {
		t.Fatalf("Failed to delete bucket: %v", err)
	}

	// Verify bucket directory was removed
	bucketPath := filepath.Join(tmpDir, "test-bucket")
	if _, err := os.Stat(bucketPath); !os.IsNotExist(err) {
		t.Errorf("Bucket directory was not deleted")
	}
}

func TestLocalStorage_HeadBucket(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test non-existent bucket
	err = storage.HeadBucket("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent bucket")
	}

	// Create bucket and test again
	err = storage.CreateBucket("test-bucket")
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	err = storage.HeadBucket("test-bucket")
	if err != nil {
		t.Errorf("Expected no error for existing bucket: %v", err)
	}
}

func TestLocalStorage_ListBuckets(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create some buckets
	storage.CreateBucket("bucket-a")
	storage.CreateBucket("bucket-b")
	storage.CreateBucket("bucket-c")

	buckets, err := storage.ListBuckets()
	if err != nil {
		t.Fatalf("Failed to list buckets: %v", err)
	}

	if len(buckets) != 3 {
		t.Errorf("Expected 3 buckets, got %d", len(buckets))
	}
}

func TestLocalStorage_PutObject(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create bucket
	err = storage.CreateBucket("test-bucket")
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Put object
	data := []byte("Hello, World!")
	reader := bytes.NewReader(data)
	meta, err := storage.PutObject("test-bucket", "test.txt", reader, int64(len(data)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Failed to put object: %v", err)
	}

	if meta.Key != "test.txt" {
		t.Errorf("Expected key 'test.txt', got '%s'", meta.Key)
	}

	if meta.Size != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), meta.Size)
	}

	if meta.ContentType != "text/plain" {
		t.Errorf("Expected content type 'text/plain', got '%s'", meta.ContentType)
	}
}

func TestLocalStorage_GetObject(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create bucket and put object
	storage.CreateBucket("test-bucket")
	data := []byte("Hello, World!")
	storage.PutObject("test-bucket", "test.txt", bytes.NewReader(data), int64(len(data)), "text/plain", nil)

	// Get object
	reader, meta, err := storage.GetObject("test-bucket", "test.txt")
	if err != nil {
		t.Fatalf("Failed to get object: %v", err)
	}
	defer reader.Close()

	// Read content
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read object: %v", err)
	}

	if !bytes.Equal(content, data) {
		t.Errorf("Content mismatch: expected '%s', got '%s'", data, content)
	}

	if meta.Size != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), meta.Size)
	}
}

func TestLocalStorage_DeleteObject(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create bucket and put object
	storage.CreateBucket("test-bucket")
	data := []byte("Hello, World!")
	storage.PutObject("test-bucket", "test.txt", bytes.NewReader(data), int64(len(data)), "text/plain", nil)

	// Delete object
	err = storage.DeleteObject("test-bucket", "test.txt")
	if err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}

	// Verify object was deleted
	_, _, err = storage.GetObject("test-bucket", "test.txt")
	if err == nil {
		t.Error("Expected error when getting deleted object")
	}
}

func TestLocalStorage_ListObjects(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create bucket and put objects
	storage.CreateBucket("test-bucket")
	storage.PutObject("test-bucket", "file1.txt", bytes.NewReader([]byte("1")), 1, "text/plain", nil)
	storage.PutObject("test-bucket", "file2.txt", bytes.NewReader([]byte("2")), 1, "text/plain", nil)
	storage.PutObject("test-bucket", "dir/file3.txt", bytes.NewReader([]byte("3")), 1, "text/plain", nil)

	// List all objects
	objects, _, err := storage.ListObjects("test-bucket", "", "", 1000)
	if err != nil {
		t.Fatalf("Failed to list objects: %v", err)
	}

	if len(objects) != 3 {
		t.Errorf("Expected 3 objects, got %d", len(objects))
	}

	// List with delimiter
	objects, prefixes, err := storage.ListObjects("test-bucket", "", "/", 1000)
	if err != nil {
		t.Fatalf("Failed to list objects with delimiter: %v", err)
	}

	if len(objects) != 2 {
		t.Errorf("Expected 2 objects, got %d", len(objects))
	}

	if len(prefixes) != 1 || prefixes[0] != "dir/" {
		t.Errorf("Expected prefix 'dir/', got %v", prefixes)
	}
}

func TestLocalStorage_ObjectWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create bucket
	storage.CreateBucket("test-bucket")

	// Put object with metadata
	metadata := map[string]string{
		"author": "test",
		"type":   "document",
	}
	data := []byte("content")
	storage.PutObject("test-bucket", "test.txt", bytes.NewReader(data), int64(len(data)), "text/plain", metadata)

	// Get object and verify metadata
	_, meta, err := storage.GetObject("test-bucket", "test.txt")
	if err != nil {
		t.Fatalf("Failed to get object: %v", err)
	}

	if meta.Metadata["author"] != "test" {
		t.Errorf("Expected metadata author 'test', got '%s'", meta.Metadata["author"])
	}

	if meta.Metadata["type"] != "document" {
		t.Errorf("Expected metadata type 'document', got '%s'", meta.Metadata["type"])
	}
}
