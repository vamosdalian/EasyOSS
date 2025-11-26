package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vamosdalian/EasyOSS/internal/storage"
)

var (
	//go:embed templates/*.html
	templatesFS embed.FS

	//go:embed static/*
	staticFS embed.FS
)

// WebHandler handles web UI requests
type WebHandler struct {
	storage   *storage.LocalStorage
	templates *template.Template
}

// NewWebHandler creates a new web handler
func NewWebHandler(storage *storage.LocalStorage) (*WebHandler, error) {
	templates, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &WebHandler{
		storage:   storage,
		templates: templates,
	}, nil
}

// BucketData represents data for bucket template
type BucketData struct {
	Buckets []storage.BucketMeta
	Error   string
}

// ObjectData represents data for object template
type ObjectData struct {
	Bucket         string
	Prefix         string
	Objects        []storage.ObjectMeta
	CommonPrefixes []string
	ParentPrefix   string
	Error          string
}

// ServeHTTP handles HTTP requests for web UI
func (h *WebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Serve static files
	if strings.HasPrefix(path, "/static/") {
		h.serveStatic(w, r)
		return
	}

	switch {
	case path == "/" || path == "/web" || path == "/web/":
		h.listBuckets(w, r)
	case strings.HasPrefix(path, "/web/bucket/"):
		h.handleBucket(w, r)
	case strings.HasPrefix(path, "/web/object/"):
		h.handleObject(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveStatic serves static files
func (h *WebHandler) serveStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	data, err := staticFS.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set content type based on extension
	ext := filepath.Ext(path)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.Write(data)
}

// listBuckets displays all buckets
func (h *WebHandler) listBuckets(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		action := r.FormValue("action")
		bucketName := r.FormValue("bucket")

		switch action {
		case "create":
			if err := h.storage.CreateBucket(bucketName); err != nil {
				data := BucketData{Error: err.Error()}
				h.templates.ExecuteTemplate(w, "buckets.html", data)
				return
			}
		case "delete":
			if err := h.storage.DeleteBucket(bucketName); err != nil {
				data := BucketData{Error: err.Error()}
				h.templates.ExecuteTemplate(w, "buckets.html", data)
				return
			}
		}

		http.Redirect(w, r, "/web/", http.StatusSeeOther)
		return
	}

	buckets, err := h.storage.ListBuckets()
	if err != nil {
		data := BucketData{Error: err.Error()}
		h.templates.ExecuteTemplate(w, "buckets.html", data)
		return
	}

	data := BucketData{Buckets: buckets}
	h.templates.ExecuteTemplate(w, "buckets.html", data)
}

// handleBucket handles bucket operations
func (h *WebHandler) handleBucket(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/web/bucket/")
	bucket := strings.TrimSuffix(path, "/")
	prefix := r.URL.Query().Get("prefix")

	if r.Method == http.MethodPost {
		action := r.FormValue("action")

		switch action {
		case "upload":
			h.uploadObject(w, r, bucket, prefix)
			return
		case "delete":
			key := r.FormValue("key")
			if err := h.storage.DeleteObject(bucket, key); err != nil {
				data := ObjectData{Bucket: bucket, Error: err.Error()}
				h.templates.ExecuteTemplate(w, "objects.html", data)
				return
			}
		}

		redirectURL := "/web/bucket/" + bucket + "/"
		if prefix != "" {
			redirectURL += "?prefix=" + prefix
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	objects, commonPrefixes, err := h.storage.ListObjects(bucket, prefix, "/", 1000)
	if err != nil {
		data := ObjectData{Bucket: bucket, Error: err.Error()}
		h.templates.ExecuteTemplate(w, "objects.html", data)
		return
	}

	// Calculate parent prefix for navigation
	var parentPrefix string
	if prefix != "" {
		parts := strings.Split(strings.TrimSuffix(prefix, "/"), "/")
		if len(parts) > 1 {
			parentPrefix = strings.Join(parts[:len(parts)-1], "/") + "/"
		}
	}

	data := ObjectData{
		Bucket:         bucket,
		Prefix:         prefix,
		Objects:        objects,
		CommonPrefixes: commonPrefixes,
		ParentPrefix:   parentPrefix,
	}
	h.templates.ExecuteTemplate(w, "objects.html", data)
}

// uploadObject handles object upload
func (h *WebHandler) uploadObject(w http.ResponseWriter, r *http.Request, bucket, prefix string) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		data := ObjectData{Bucket: bucket, Error: "Failed to parse form: " + err.Error()}
		h.templates.ExecuteTemplate(w, "objects.html", data)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		data := ObjectData{Bucket: bucket, Error: "Failed to get file: " + err.Error()}
		h.templates.ExecuteTemplate(w, "objects.html", data)
		return
	}
	defer file.Close()

	// Generate object key
	key := prefix + header.Filename

	// Determine content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload object
	if _, err := h.storage.PutObject(bucket, key, file, header.Size, contentType, nil); err != nil {
		data := ObjectData{Bucket: bucket, Error: "Failed to upload: " + err.Error()}
		h.templates.ExecuteTemplate(w, "objects.html", data)
		return
	}

	redirectURL := "/web/bucket/" + bucket + "/"
	if prefix != "" {
		redirectURL += "?prefix=" + prefix
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// handleObject handles object download
func (h *WebHandler) handleObject(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/web/object/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	bucket := parts[0]
	key := parts[1]

	reader, meta, err := h.storage.GetObject(bucket, key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(meta.Size, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(key)))

	io.Copy(w, reader)
}
