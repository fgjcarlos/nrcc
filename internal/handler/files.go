package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/go-chi/chi/v5"
)

const maxPublicImageBytes = 2 * 1024 * 1024

// FilesHandler handles file upload/management endpoints
type FilesHandler struct {
	dataDir string
	audit   *audit.Service
}

// NewFilesHandler creates a new files handler
func NewFilesHandler(dataDir string) *FilesHandler {
	return &FilesHandler{dataDir: dataDir}
}

// SetAuditService injects the audit logger.
func (h *FilesHandler) SetAuditService(a *audit.Service) { h.audit = a }

// PostUpload uploads a file
// POST /api/files/upload
func (h *FilesHandler) PostUpload(w http.ResponseWriter, r *http.Request) {
	// Cap total request body to 100 MiB to prevent memory exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)
	// Spool up to 1 MiB per file in memory; anything larger spills to a temp file.
	// #nosec G120 -- body is bounded by http.MaxBytesReader above; maxMemory is 1 MiB so the residual in-memory footprint is well under the cap.
	if err := r.ParseMultipartForm(1048576); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "No file provided")
		return
	}
	defer func() { _ = file.Close() }()

	// Validate filename (prevent path traversal)
	filename := filepath.Base(header.Filename)
	if strings.Contains(filename, "..") || filename == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid filename")
		return
	}

	uploadDir := filepath.Join(h.dataDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "UPLOAD_ERROR", err.Error())
		return
	}

	uploadPath := filepath.Join(uploadDir, filename)

	// Create file
	// #nosec G304 -- filename is validated against path traversal and base name upstream; uploadPath is rooted to dataDir/uploads.
	dst, err := os.Create(uploadPath)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "UPLOAD_ERROR", err.Error())
		return
	}
	defer func() { _ = dst.Close() }()

	// Copy file data
	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(uploadPath)
		model.RespondError(w, http.StatusInternalServerError, "UPLOAD_ERROR", err.Error())
		return
	}

	if h.audit != nil {
		h.audit.Log(r, "", "FILE_UPLOAD", filename, "ok", nil)
	}
	model.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"filename": filename,
		"path":     "/uploads/" + filename,
	})
}

// DeleteFile deletes a file
// DELETE /api/files/{name}
func (h *FilesHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Validate filename
	if strings.Contains(name, "..") || name == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid filename")
		return
	}

	// #nosec G703 -- name is validated against ".." and empty upstream; the resulting path is rooted to dataDir/uploads.
	filePath := filepath.Join(h.dataDir, "uploads", name)

	// #nosec G703 -- filePath is rooted to dataDir/uploads and validated against path traversal upstream.
	if err := os.Remove(filePath); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error())
		return
	}

	if h.audit != nil {
		h.audit.Log(r, "", "FILE_DELETE", name, "ok", nil)
	}
	w.WriteHeader(http.StatusNoContent)
}

// DownloadFile downloads an uploaded file
// GET /api/files/{name}/download
func (h *FilesHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if strings.Contains(name, "..") || name == "" || filepath.Base(name) != name {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid filename")
		return
	}

	// #nosec G703 -- name is validated against ".." and base-mismatch upstream; the resulting path is rooted to dataDir/uploads.
	filePath := filepath.Join(h.dataDir, "uploads", name)
	// #nosec G703 -- filePath is rooted to dataDir/uploads and validated against path traversal upstream.
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "DOWNLOAD_ERROR", err.Error())
		return
	}

	if info.IsDir() {
		model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+strings.ReplaceAll(name, "\"", "")+"\"")
	// #nosec G703 -- filePath is rooted to dataDir/uploads and validated against path traversal upstream.
	http.ServeFile(w, r, filePath)
}

// ServeImage serves an uploaded image without authentication so Node-RED's
// editor can load favicon/header/login artwork referenced from settings.js.
// Only actual image payloads up to the same 2 MiB UI limit are exposed; other
// uploaded files remain behind the authenticated download endpoint.
func (h *FilesHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if strings.Contains(name, "..") || name == "" || filepath.Base(name) != name {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid filename")
		return
	}
	filePath := filepath.Join(h.dataDir, "uploads", name)
	file, err := os.Open(filePath) // #nosec G304 -- path is rooted and name is base-validated above.
	if err != nil {
		if os.IsNotExist(err) {
			model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Image not found")
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "IMAGE_ERROR", err.Error())
		return
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || info.IsDir() || info.Size() > maxPublicImageBytes {
		model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Image not found")
		return
	}
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		model.RespondError(w, http.StatusInternalServerError, "IMAGE_ERROR", err.Error())
		return
	}
	contentType := http.DetectContentType(header[:n])
	if !strings.HasPrefix(contentType, "image/") {
		model.RespondError(w, http.StatusUnsupportedMediaType, "NOT_AN_IMAGE", "Uploaded file is not an image")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "IMAGE_ERROR", err.Error())
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

// GetList lists uploaded files
// GET /api/files
func (h *FilesHandler) GetList(w http.ResponseWriter, r *http.Request) {
	uploadDir := filepath.Join(h.dataDir, "uploads")

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			model.RespondJSON(w, http.StatusOK, []interface{}{})
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}

	var files []interface{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, map[string]interface{}{
			"name":    entry.Name(),
			"size":    info.Size(),
			"modTime": info.ModTime().Unix(),
		})
	}

	model.RespondJSON(w, http.StatusOK, files)
}
