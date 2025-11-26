package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ObjectMeta represents metadata for an object
type ObjectMeta struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	ETag         string            `json:"etag"`
	LastModified time.Time         `json:"last_modified"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// BucketMeta represents metadata for a bucket
type BucketMeta struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// LocalStorage implements a local file system based storage
type LocalStorage struct {
	basePath string
	metaPath string
	mu       sync.RWMutex
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	return NewLocalStorageWithMeta(basePath, filepath.Join(basePath, ".meta"))
}

// NewLocalStorageWithMeta creates a new local storage instance with custom meta path
func NewLocalStorageWithMeta(basePath, metaPath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	if err := os.MkdirAll(metaPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create meta directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		metaPath: metaPath,
	}, nil
}

// bucketPath returns the path for a bucket
func (ls *LocalStorage) bucketPath(bucket string) string {
	return filepath.Join(ls.basePath, bucket)
}

// objectPath returns the path for an object
func (ls *LocalStorage) objectPath(bucket, key string) string {
	return filepath.Join(ls.basePath, bucket, key)
}

// metaPath returns the path for bucket metadata
func (ls *LocalStorage) bucketMetaPath(bucket string) string {
	return filepath.Join(ls.metaPath, bucket+".json")
}

// objectMetaPath returns the path for object metadata
func (ls *LocalStorage) objectMetaPath(bucket, key string) string {
	return filepath.Join(ls.metaPath, bucket, key+".json")
}

// CreateBucket creates a new bucket
func (ls *LocalStorage) CreateBucket(name string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	bucketDir := ls.bucketPath(name)
	if _, err := os.Stat(bucketDir); err == nil {
		return fmt.Errorf("bucket already exists")
	}

	if err := os.MkdirAll(bucketDir, 0755); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	// Create bucket metadata directory
	metaDir := filepath.Join(ls.metaPath, name)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("failed to create bucket meta directory: %w", err)
	}

	// Save bucket metadata
	meta := BucketMeta{
		Name:      name,
		CreatedAt: time.Now(),
	}

	metaData, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal bucket metadata: %w", err)
	}

	if err := os.WriteFile(ls.bucketMetaPath(name), metaData, 0644); err != nil {
		return fmt.Errorf("failed to save bucket metadata: %w", err)
	}

	return nil
}

// DeleteBucket deletes a bucket
func (ls *LocalStorage) DeleteBucket(name string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	bucketDir := ls.bucketPath(name)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		return fmt.Errorf("bucket does not exist")
	}

	// Check if bucket is empty
	entries, err := os.ReadDir(bucketDir)
	if err != nil {
		return fmt.Errorf("failed to read bucket directory: %w", err)
	}

	if len(entries) > 0 {
		return fmt.Errorf("bucket is not empty")
	}

	// Delete bucket directory
	if err := os.Remove(bucketDir); err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	// Delete bucket metadata
	metaFile := ls.bucketMetaPath(name)
	_ = os.Remove(metaFile)

	// Delete bucket meta directory
	metaDir := filepath.Join(ls.metaPath, name)
	_ = os.RemoveAll(metaDir)

	return nil
}

// HeadBucket checks if a bucket exists
func (ls *LocalStorage) HeadBucket(name string) error {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	bucketDir := ls.bucketPath(name)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		return fmt.Errorf("bucket does not exist")
	}

	return nil
}

// ListBuckets lists all buckets
func (ls *LocalStorage) ListBuckets() ([]BucketMeta, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	entries, err := os.ReadDir(ls.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	var buckets []BucketMeta
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".meta" {
			continue
		}

		meta, err := ls.getBucketMeta(entry.Name())
		if err != nil {
			// If no metadata exists, create a default one
			info, _ := entry.Info()
			modTime := time.Now()
			if info != nil {
				modTime = info.ModTime()
			}
			meta = &BucketMeta{
				Name:      entry.Name(),
				CreatedAt: modTime,
			}
		}
		buckets = append(buckets, *meta)
	}

	return buckets, nil
}

// getBucketMeta reads bucket metadata from file
func (ls *LocalStorage) getBucketMeta(name string) (*BucketMeta, error) {
	metaFile := ls.bucketMetaPath(name)
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}

	var meta BucketMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// PutObject stores an object
func (ls *LocalStorage) PutObject(bucket, key string, data io.Reader, size int64, contentType string, metadata map[string]string) (*ObjectMeta, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// Check if bucket exists
	bucketDir := ls.bucketPath(bucket)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("bucket does not exist")
	}

	// Create object directory if needed
	objectFile := ls.objectPath(bucket, key)
	objectDir := filepath.Dir(objectFile)
	if err := os.MkdirAll(objectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create object directory: %w", err)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp(objectDir, ".tmp-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Copy data to temporary file and calculate hash
	written, err := io.Copy(tmpFile, data)
	if err != nil {
		return nil, fmt.Errorf("failed to write object data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temporary file: %w", err)
	}
	tmpFile = nil

	// Rename temporary file to final location
	if err := os.Rename(tmpPath, objectFile); err != nil {
		return nil, fmt.Errorf("failed to rename object file: %w", err)
	}

	// Generate ETag (MD5 hash would be proper, but for simplicity use file info)
	etag := fmt.Sprintf("\"%x-%x\"", time.Now().UnixNano(), written)

	// Create object metadata
	objMeta := &ObjectMeta{
		Key:          key,
		Size:         written,
		ContentType:  contentType,
		ETag:         etag,
		LastModified: time.Now(),
		Metadata:     metadata,
	}

	// Create metadata directory if needed
	metaFile := ls.objectMetaPath(bucket, key)
	metaDir := filepath.Dir(metaFile)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Save object metadata
	metaData, err := json.Marshal(objMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object metadata: %w", err)
	}

	if err := os.WriteFile(metaFile, metaData, 0644); err != nil {
		return nil, fmt.Errorf("failed to save object metadata: %w", err)
	}

	return objMeta, nil
}

// GetObject retrieves an object
func (ls *LocalStorage) GetObject(bucket, key string) (io.ReadCloser, *ObjectMeta, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	objectFile := ls.objectPath(bucket, key)
	if _, err := os.Stat(objectFile); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("object does not exist")
	}

	file, err := os.Open(objectFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open object: %w", err)
	}

	meta, err := ls.getObjectMeta(bucket, key)
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	return file, meta, nil
}

// HeadObject retrieves object metadata
func (ls *LocalStorage) HeadObject(bucket, key string) (*ObjectMeta, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	objectFile := ls.objectPath(bucket, key)
	if _, err := os.Stat(objectFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("object does not exist")
	}

	return ls.getObjectMeta(bucket, key)
}

// DeleteObject deletes an object
func (ls *LocalStorage) DeleteObject(bucket, key string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	objectFile := ls.objectPath(bucket, key)
	if _, err := os.Stat(objectFile); os.IsNotExist(err) {
		// S3 returns success even if object doesn't exist
		return nil
	}

	if err := os.Remove(objectFile); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	// Delete object metadata
	metaFile := ls.objectMetaPath(bucket, key)
	_ = os.Remove(metaFile)

	// Clean up empty directories
	ls.cleanupEmptyDirs(filepath.Dir(objectFile), ls.bucketPath(bucket))
	ls.cleanupEmptyDirs(filepath.Dir(metaFile), filepath.Join(ls.metaPath, bucket))

	return nil
}

// cleanupEmptyDirs removes empty directories up to stopAt
func (ls *LocalStorage) cleanupEmptyDirs(dir, stopAt string) {
	for dir != stopAt {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		if err := os.Remove(dir); err != nil {
			break
		}
		dir = filepath.Dir(dir)
	}
}

// ListObjects lists objects in a bucket
func (ls *LocalStorage) ListObjects(bucket, prefix, delimiter string, maxKeys int) ([]ObjectMeta, []string, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	bucketDir := ls.bucketPath(bucket)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("bucket does not exist")
	}

	var objects []ObjectMeta
	prefixes := make(map[string]bool)

	err := filepath.Walk(bucketDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(bucketDir, path)
		if err != nil {
			return nil
		}

		// Convert to forward slashes for S3 compatibility
		key := filepath.ToSlash(relPath)

		// Check prefix filter
		if prefix != "" && len(key) >= len(prefix) {
			if key[:len(prefix)] != prefix {
				return nil
			}
		} else if prefix != "" && len(key) < len(prefix) {
			return nil
		}

		// Handle delimiter
		if delimiter != "" {
			suffixStart := len(prefix)
			if suffixStart < len(key) {
				suffix := key[suffixStart:]
				delimIndex := -1
				for i := 0; i < len(suffix); i++ {
					if string(suffix[i]) == delimiter {
						delimIndex = i
						break
					}
				}
				if delimIndex >= 0 {
					commonPrefix := key[:suffixStart+delimIndex+1]
					prefixes[commonPrefix] = true
					return nil
				}
			}
		}

		meta, err := ls.getObjectMeta(bucket, key)
		if err != nil {
			// Create metadata from file info if not available
			meta = &ObjectMeta{
				Key:          key,
				Size:         info.Size(),
				ContentType:  "application/octet-stream",
				ETag:         fmt.Sprintf("\"%x\"", info.ModTime().UnixNano()),
				LastModified: info.ModTime(),
			}
		}

		objects = append(objects, *meta)
		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Sort objects by key
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Key < objects[j].Key
	})

	// Extract common prefixes
	var commonPrefixes []string
	for p := range prefixes {
		commonPrefixes = append(commonPrefixes, p)
	}
	sort.Strings(commonPrefixes)

	// Apply maxKeys limit
	if maxKeys > 0 && len(objects) > maxKeys {
		objects = objects[:maxKeys]
	}

	return objects, commonPrefixes, nil
}

// getObjectMeta reads object metadata from file
func (ls *LocalStorage) getObjectMeta(bucket, key string) (*ObjectMeta, error) {
	metaFile := ls.objectMetaPath(bucket, key)
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}

	var meta ObjectMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
