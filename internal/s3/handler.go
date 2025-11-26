package s3

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/vamosdalian/EasyOSS/internal/storage"
)

// S3Handler handles S3 API requests
type S3Handler struct {
	storage *storage.LocalStorage
}

// NewS3Handler creates a new S3 handler
func NewS3Handler(storage *storage.LocalStorage) *S3Handler {
	return &S3Handler{
		storage: storage,
	}
}

// XML structures for S3 responses

// ListAllMyBucketsResult represents the response for ListBuckets
type ListAllMyBucketsResult struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Xmlns   string   `xml:"xmlns,attr"`
	Owner   Owner    `xml:"Owner"`
	Buckets Buckets  `xml:"Buckets"`
}

// Owner represents bucket owner
type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

// Buckets represents a list of buckets
type Buckets struct {
	Bucket []Bucket `xml:"Bucket"`
}

// Bucket represents a single bucket
type Bucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

// ListBucketResult represents the response for ListObjects
type ListBucketResult struct {
	XMLName        xml.Name         `xml:"ListBucketResult"`
	Xmlns          string           `xml:"xmlns,attr"`
	Name           string           `xml:"Name"`
	Prefix         string           `xml:"Prefix"`
	Delimiter      string           `xml:"Delimiter,omitempty"`
	MaxKeys        int              `xml:"MaxKeys"`
	IsTruncated    bool             `xml:"IsTruncated"`
	Contents       []Contents       `xml:"Contents"`
	CommonPrefixes []CommonPrefixes `xml:"CommonPrefixes,omitempty"`
}

// Contents represents an object in the list
type Contents struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

// CommonPrefixes represents common prefixes in object listing
type CommonPrefixes struct {
	Prefix string `xml:"Prefix"`
}

// Error represents an S3 error response
type Error struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource,omitempty"`
	RequestID string   `xml:"RequestId,omitempty"`
}

// CopyObjectResult represents the response for CopyObject
type CopyObjectResult struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	LastModified string   `xml:"LastModified"`
	ETag         string   `xml:"ETag"`
}

// ServeHTTP handles HTTP requests for S3 API
func (h *S3Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Parse bucket and key from path
	bucket, key := parsePath(path)

	switch {
	case bucket == "" && r.Method == http.MethodGet:
		// List buckets
		h.listBuckets(w, r)
	case bucket != "" && key == "":
		// Bucket operations
		switch r.Method {
		case http.MethodGet:
			h.listObjects(w, r, bucket)
		case http.MethodPut:
			h.createBucket(w, r, bucket)
		case http.MethodDelete:
			h.deleteBucket(w, r, bucket)
		case http.MethodHead:
			h.headBucket(w, r, bucket)
		default:
			h.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		}
	case bucket != "" && key != "":
		// Object operations
		switch r.Method {
		case http.MethodGet:
			h.getObject(w, r, bucket, key)
		case http.MethodPut:
			h.putObject(w, r, bucket, key)
		case http.MethodDelete:
			h.deleteObject(w, r, bucket, key)
		case http.MethodHead:
			h.headObject(w, r, bucket, key)
		default:
			h.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		}
	default:
		h.writeError(w, http.StatusBadRequest, "InvalidRequest", "Invalid request")
	}
}

// parsePath extracts bucket and key from URL path
func parsePath(path string) (bucket, key string) {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "", ""
	}

	parts := strings.SplitN(path, "/", 2)
	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}
	return bucket, key
}

// listBuckets handles GET / - List all buckets
func (h *S3Handler) listBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := h.storage.ListBuckets()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}

	result := ListAllMyBucketsResult{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner: Owner{
			ID:          "easyoss",
			DisplayName: "EasyOSS",
		},
	}

	for _, b := range buckets {
		result.Buckets.Bucket = append(result.Buckets.Bucket, Bucket{
			Name:         b.Name,
			CreationDate: b.CreatedAt.Format(time.RFC3339),
		})
	}

	h.writeXML(w, http.StatusOK, result)
}

// createBucket handles PUT /{bucket} - Create bucket
func (h *S3Handler) createBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	if !isValidBucketName(bucket) {
		h.writeError(w, http.StatusBadRequest, "InvalidBucketName", "Invalid bucket name")
		return
	}

	if err := h.storage.CreateBucket(bucket); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			h.writeError(w, http.StatusConflict, "BucketAlreadyExists", err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}

	w.Header().Set("Location", "/"+bucket)
	w.WriteHeader(http.StatusOK)
}

// deleteBucket handles DELETE /{bucket} - Delete bucket
func (h *S3Handler) deleteBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	if err := h.storage.DeleteBucket(bucket); err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			h.writeError(w, http.StatusNotFound, "NoSuchBucket", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not empty") {
			h.writeError(w, http.StatusConflict, "BucketNotEmpty", err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// headBucket handles HEAD /{bucket} - Check bucket existence
func (h *S3Handler) headBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	if err := h.storage.HeadBucket(bucket); err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// listObjects handles GET /{bucket} - List objects in bucket
func (h *S3Handler) listObjects(w http.ResponseWriter, r *http.Request, bucket string) {
	prefix := r.URL.Query().Get("prefix")
	delimiter := r.URL.Query().Get("delimiter")
	maxKeys := 1000 // Default max keys

	objects, commonPrefixes, err := h.storage.ListObjects(bucket, prefix, delimiter, maxKeys)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			h.writeError(w, http.StatusNotFound, "NoSuchBucket", err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}

	result := ListBucketResult{
		Xmlns:       "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:        bucket,
		Prefix:      prefix,
		Delimiter:   delimiter,
		MaxKeys:     maxKeys,
		IsTruncated: false,
	}

	for _, obj := range objects {
		result.Contents = append(result.Contents, Contents{
			Key:          obj.Key,
			LastModified: obj.LastModified.Format(time.RFC3339),
			ETag:         obj.ETag,
			Size:         obj.Size,
			StorageClass: "STANDARD",
		})
	}

	for _, prefix := range commonPrefixes {
		result.CommonPrefixes = append(result.CommonPrefixes, CommonPrefixes{
			Prefix: prefix,
		})
	}

	h.writeXML(w, http.StatusOK, result)
}

// getObject handles GET /{bucket}/{key} - Get object
func (h *S3Handler) getObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	reader, meta, err := h.storage.GetObject(bucket, key)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			h.writeError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size))
	w.Header().Set("ETag", meta.ETag)
	w.Header().Set("Last-Modified", meta.LastModified.Format(http.TimeFormat))

	// Set custom metadata headers
	for k, v := range meta.Metadata {
		w.Header().Set("x-amz-meta-"+k, v)
	}

	io.Copy(w, reader)
}

// putObject handles PUT /{bucket}/{key} - Put object
func (h *S3Handler) putObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Extract custom metadata
	metadata := make(map[string]string)
	for k, v := range r.Header {
		if strings.HasPrefix(strings.ToLower(k), "x-amz-meta-") {
			metaKey := strings.TrimPrefix(strings.ToLower(k), "x-amz-meta-")
			if len(v) > 0 {
				metadata[metaKey] = v[0]
			}
		}
	}

	meta, err := h.storage.PutObject(bucket, key, r.Body, r.ContentLength, contentType, metadata)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			h.writeError(w, http.StatusNotFound, "NoSuchBucket", err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}

	w.Header().Set("ETag", meta.ETag)
	w.WriteHeader(http.StatusOK)
}

// deleteObject handles DELETE /{bucket}/{key} - Delete object
func (h *S3Handler) deleteObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	if err := h.storage.DeleteObject(bucket, key); err != nil {
		h.writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// headObject handles HEAD /{bucket}/{key} - Get object metadata
func (h *S3Handler) headObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	meta, err := h.storage.HeadObject(bucket, key)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size))
	w.Header().Set("ETag", meta.ETag)
	w.Header().Set("Last-Modified", meta.LastModified.Format(http.TimeFormat))

	// Set custom metadata headers
	for k, v := range meta.Metadata {
		w.Header().Set("x-amz-meta-"+k, v)
	}

	w.WriteHeader(http.StatusOK)
}

// writeXML writes an XML response
func (h *S3Handler) writeXML(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(data)
}

// writeError writes an S3 error response
func (h *S3Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	errResp := Error{
		Code:    code,
		Message: message,
	}
	h.writeXML(w, status, errResp)
}

// isValidBucketName checks if a bucket name is valid
func isValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}

	// Must start with lowercase letter or number
	if !regexp.MustCompile(`^[a-z0-9]`).MatchString(name) {
		return false
	}

	// Must end with lowercase letter or number
	if !regexp.MustCompile(`[a-z0-9]$`).MatchString(name) {
		return false
	}

	// Only lowercase letters, numbers, and hyphens
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(name) {
		return false
	}

	// Cannot have consecutive periods or hyphens
	if regexp.MustCompile(`\.\.`).MatchString(name) || regexp.MustCompile(`--`).MatchString(name) {
		return false
	}

	return true
}
