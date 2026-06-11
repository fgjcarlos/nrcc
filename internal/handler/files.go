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
	// Parse multipart form with max 100MB size
	if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "No file provided")
		return
	}
	defer file.Close()

	// Validate filename (prevent path traversal)
	filename := filepath.Base(header.Filename)
	if strings.Contains(filename, "..") || filename == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid filename")
		return
	}

	uploadDir := filepath.Join(h.dataDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "UPLOAD_ERROR", err.Error())
		return
	}

	uploadPath := filepath.Join(uploadDir, filename)

	// Create file
	dst, err := os.Create(uploadPath)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "UPLOAD_ERROR", err.Error())
		return
	}
	defer dst.Close()

	// Copy file data
	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(uploadPath)
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

	filePath := filepath.Join(h.dataDir, "uploads", name)

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

	filePath := filepath.Join(h.dataDir, "uploads", name)
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
	http.ServeFile(w, r, filePath)
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
