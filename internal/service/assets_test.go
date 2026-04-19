package service

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

func TestAssetUploadAndList(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	svc := NewAssetService(dataDir)

	// Create a small PNG file (1x1 pixel)
	pngData := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG header
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}

	file, header := createMultipartFile(t, "test.png", pngData)
	asset, err := svc.Upload("favicon", file, header)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if asset.Category != "favicon" {
		t.Errorf("Category = %q, want %q", asset.Category, "favicon")
	}
	if asset.Original != "test.png" {
		t.Errorf("Original = %q, want %q", asset.Original, "test.png")
	}
	if asset.MIMEType != "image/png" {
		t.Errorf("MIMEType = %q, want %q", asset.MIMEType, "image/png")
	}
	if asset.URL == "" {
		t.Error("URL should not be empty")
	}

	// List
	list, err := svc.List("favicon")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("List() items = %d, want 1", len(list.Items))
	}
	if list.Items[0].ID != asset.ID {
		t.Errorf("listed asset ID = %q, want %q", list.Items[0].ID, asset.ID)
	}

	// Delete
	if err := svc.Delete("favicon", asset.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	list, err = svc.List("favicon")
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("List() after delete items = %d, want 0", len(list.Items))
	}
}

func TestAssetUploadInvalidCategory(t *testing.T) {
	t.Parallel()
	svc := NewAssetService(t.TempDir())

	file, header := createMultipartFile(t, "test.png", []byte{0x89, 0x50, 0x4e, 0x47})
	_, err := svc.Upload("invalid", file, header)
	if err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestAssetUploadTooLarge(t *testing.T) {
	t.Parallel()
	svc := NewAssetService(t.TempDir())

	// Create a file over 2MB
	bigData := make([]byte, 3<<20)
	file, header := createMultipartFile(t, "big.png", bigData)
	_, err := svc.Upload("favicon", file, header)
	if err == nil {
		t.Error("expected error for oversized file")
	}
}

func TestAssetUploadInvalidMIME(t *testing.T) {
	t.Parallel()
	svc := NewAssetService(t.TempDir())

	file, header := createMultipartFile(t, "test.txt", []byte("hello world this is plain text"))
	_, err := svc.Upload("favicon", file, header)
	if err == nil {
		t.Error("expected error for non-image file")
	}
}

func TestAssetListEmptyCategory(t *testing.T) {
	t.Parallel()
	svc := NewAssetService(t.TempDir())

	list, err := svc.List("header")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected empty list, got %d items", len(list.Items))
	}
}

func TestAssetDeleteNotFound(t *testing.T) {
	t.Parallel()
	svc := NewAssetService(t.TempDir())

	// Create the category dir so the listing doesn't fail
	os.MkdirAll(filepath.Join(svc.AssetsDir(), "favicon"), 0o755)

	err := svc.Delete("favicon", "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent asset")
	}
}

func createMultipartFile(t *testing.T, filename string, data []byte) (multipart.File, *multipart.FileHeader) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(data) + 1024))
	if err != nil {
		t.Fatalf("ReadForm() error = %v", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		t.Fatal("no file found in form")
	}

	f, err := files[0].Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closer, ok := f.(io.Closer); ok {
			closer.Close()
		}
	})

	return f, files[0]
}
