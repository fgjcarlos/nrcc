package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

const (
	maxUploadSize = 2 << 20 // 2 MB
)

var allowedMIMETypes = map[string]bool{
	"image/png":     true,
	"image/jpeg":    true,
	"image/gif":     true,
	"image/svg+xml": true,
	"image/x-icon":  true,
	"image/vnd.microsoft.icon": true,
}

var validCategories = map[string]bool{
	"favicon": true,
	"header":  true,
	"login":   true,
}

// AssetService manages branding asset uploads.
type AssetService struct {
	assetsDir string
}

// NewAssetService creates a new AssetService storing files under dataDir/assets/.
func NewAssetService(dataDir string) AssetService {
	return AssetService{assetsDir: filepath.Join(dataDir, "assets")}
}

// Upload validates and stores an uploaded file, returning the asset metadata.
func (s AssetService) Upload(category string, file multipart.File, header *multipart.FileHeader) (model.Asset, error) {
	if !validCategories[category] {
		return model.Asset{}, fmt.Errorf("invalid category %q: must be one of favicon, header, login", category)
	}

	if header.Size > maxUploadSize {
		return model.Asset{}, fmt.Errorf("file too large: %d bytes (max %d)", header.Size, maxUploadSize)
	}

	// Read first 512 bytes to detect MIME type
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return model.Asset{}, fmt.Errorf("read file: %w", err)
	}
	mimeType := http.DetectContentType(buf[:n])

	// SVG detection: DetectContentType returns text/xml or text/plain for SVGs
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == ".svg" && (strings.HasPrefix(mimeType, "text/") || mimeType == "application/xml") {
		mimeType = "image/svg+xml"
	}
	// .ico detection
	if ext == ".ico" && mimeType == "application/octet-stream" {
		mimeType = "image/x-icon"
	}

	if !allowedMIMETypes[mimeType] {
		return model.Asset{}, fmt.Errorf("unsupported file type %q: allowed types are PNG, JPEG, GIF, SVG, ICO", mimeType)
	}

	// Seek back to start
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return model.Asset{}, fmt.Errorf("seek file: %w", err)
		}
	}

	// Generate unique ID and filename
	id := generateID()
	safeFilename := sanitizeFilename(header.Filename)
	storedFilename := id + "_" + safeFilename

	// Ensure category directory exists
	categoryDir := filepath.Join(s.assetsDir, category)
	if err := platform.EnsureDir(categoryDir); err != nil {
		return model.Asset{}, fmt.Errorf("create assets dir: %w", err)
	}

	// Write file
	destPath := filepath.Join(categoryDir, storedFilename)
	dst, err := os.Create(destPath)
	if err != nil {
		return model.Asset{}, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(destPath)
		return model.Asset{}, fmt.Errorf("write file: %w", err)
	}

	return model.Asset{
		ID:        id,
		Category:  category,
		Filename:  storedFilename,
		Original:  header.Filename,
		MIMEType:  mimeType,
		Size:      written,
		URL:       fmt.Sprintf("/assets/%s/%s", category, storedFilename),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// List returns all assets in a category.
func (s AssetService) List(category string) (model.AssetList, error) {
	if !validCategories[category] {
		return model.AssetList{}, fmt.Errorf("invalid category %q", category)
	}

	categoryDir := filepath.Join(s.assetsDir, category)
	if !platform.Exists(categoryDir) {
		return model.AssetList{Items: []model.Asset{}}, nil
	}

	entries, err := os.ReadDir(categoryDir)
	if err != nil {
		return model.AssetList{}, fmt.Errorf("read assets dir: %w", err)
	}

	var assets []model.Asset
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		name := entry.Name()
		// Extract ID from "id_originalname" format
		id := name
		if idx := strings.Index(name, "_"); idx > 0 {
			id = name[:idx]
		}

		assets = append(assets, model.Asset{
			ID:        id,
			Category:  category,
			Filename:  name,
			Original:  name[strings.Index(name, "_")+1:],
			MIMEType:  mimeForExt(filepath.Ext(name)),
			Size:      info.Size(),
			URL:       fmt.Sprintf("/assets/%s/%s", category, name),
			CreatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	// Sort by creation time descending
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].CreatedAt > assets[j].CreatedAt
	})

	if assets == nil {
		assets = []model.Asset{}
	}

	return model.AssetList{Items: assets}, nil
}

// Delete removes an asset by ID from a category.
func (s AssetService) Delete(category, id string) error {
	if !validCategories[category] {
		return fmt.Errorf("invalid category %q", category)
	}

	categoryDir := filepath.Join(s.assetsDir, category)
	entries, err := os.ReadDir(categoryDir)
	if err != nil {
		return fmt.Errorf("read assets dir: %w", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), id+"_") {
			return os.Remove(filepath.Join(categoryDir, entry.Name()))
		}
	}

	return fmt.Errorf("asset %q not found in category %q", id, category)
}

// AssetsDir returns the base assets directory path.
func (s AssetService) AssetsDir() string {
	return s.assetsDir
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func sanitizeFilename(name string) string {
	// Keep only the base name
	name = filepath.Base(name)
	// Replace path separators and dangerous chars
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"..", "_",
		" ", "_",
	)
	name = replacer.Replace(name)
	if name == "" || name == "." {
		name = "upload"
	}
	return name
}

func mimeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
