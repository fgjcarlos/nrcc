package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/security"
)

// SupportBundleService generates support bundles for diagnostics and support
type SupportBundleService struct {
	dataDir   string
	logSvc    *LogService
	doctorSvc *DoctorService
	sanitizer *security.Sanitizer
}

// NewSupportBundleService creates a new SupportBundleService
func NewSupportBundleService(dataDir string, logSvc *LogService, doctorSvc *DoctorService, sanitizer *security.Sanitizer) *SupportBundleService {
	return &SupportBundleService{
		dataDir:   dataDir,
		logSvc:    logSvc,
		doctorSvc: doctorSvc,
		sanitizer: sanitizer,
	}
}

// Export generates a ZIP support bundle at destPath containing diagnostics
// Returns the path to the created ZIP and any error
func (s *SupportBundleService) Export(ctx context.Context) (string, error) {
	// Create support directory if it doesn't exist
	supportDir := filepath.Join(s.dataDir, "support")
	if err := os.MkdirAll(supportDir, 0700); err != nil {
		return "", fmt.Errorf("create support directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	zipName := fmt.Sprintf("nrcc-support-%s.zip", timestamp)
	zipPath := filepath.Join(supportDir, zipName)

	// Create ZIP file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("create zip file: %w", err)
	}
	defer zipFile.Close()

	writer := zip.NewWriter(zipFile)
	defer writer.Close()

	// Manifest
	manifest := model.SupportBundleManifest{
		FileName:    zipName,
		GeneratedAt: time.Now().UTC(),
		GeneratedBy: "nrcc-cli",
	}

	// 1. Add manifest.json
	if err := s.addFileToZip(writer, "manifest.json", func() ([]byte, error) {
		return json.MarshalIndent(manifest, "", "  ")
	}); err != nil {
		return "", fmt.Errorf("add manifest: %w", err)
	}

	// 2. Add doctor_report.json
	if s.doctorSvc != nil {
		if err := s.addFileToZip(writer, "doctor_report.json", func() ([]byte, error) {
			report := s.doctorSvc.Run(ctx)
			return json.MarshalIndent(report, "", "  ")
		}); err != nil {
			return "", fmt.Errorf("add doctor report: %w", err)
		}
	}

	// 3. Add app.log (last 500 lines or entire file, sanitized)
	logPath := filepath.Join(s.dataDir, "logs", "app.log")
	if err := s.addFileToZip(writer, "app.log", func() ([]byte, error) {
		return s.readAndSanitizeLog(logPath, 500)
	}); err != nil {
		return "", fmt.Errorf("add app.log: %w", err)
	}

	// 4. Add settings.json (sanitized)
	settingsPath := filepath.Join(s.dataDir, "nodered", "settings.js")
	if err := s.addFileToZip(writer, "settings.json", func() ([]byte, error) {
		return s.readAndSanitizeFile(settingsPath)
	}); err != nil {
		return "", fmt.Errorf("add settings: %w", err)
	}

	// 5. Add system_info.json
	if err := s.addFileToZip(writer, "system_info.json", func() ([]byte, error) {
		return s.getSystemInfo()
	}); err != nil {
		return "", fmt.Errorf("add system_info: %w", err)
	}

	return zipPath, nil
}

// Helper: add a file to the ZIP
func (s *SupportBundleService) addFileToZip(writer *zip.Writer, filename string, contentFn func() ([]byte, error)) error {
	content, err := contentFn()
	if err != nil {
		// Log error but don't fail the entire bundle
		return nil
	}

	header := &zip.FileHeader{
		Name:     filename,
		Modified: time.Now(),
	}
	f, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = f.Write(content)
	return err
}

// Helper: read and sanitize log file, return last N lines
func (s *SupportBundleService) readAndSanitizeLog(path string, maxLines int) ([]byte, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return []byte(""), nil // Log not found is OK
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	text := strings.Join(lines, "\n")
	sanitized := s.sanitizer.SanitizeString(text)
	return []byte(sanitized), nil
}

// Helper: read and sanitize a file
func (s *SupportBundleService) readAndSanitizeFile(path string) ([]byte, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return []byte(""), nil // File not found is OK
	}

	sanitized := s.sanitizer.SanitizeString(string(content))
	return []byte(sanitized), nil
}

// Helper: get system info
func (s *SupportBundleService) getSystemInfo() ([]byte, error) {
	hostname, _ := os.Hostname()

	info := map[string]interface{}{
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"cpus":      runtime.NumCPU(),
		"hostname":  hostname,
		"timestamp": time.Now().UTC(),
		"goVersion": runtime.Version(),
	}

	return json.MarshalIndent(info, "", "  ")
}
