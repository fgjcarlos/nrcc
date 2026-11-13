package service

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// Node-RED's editor theme stats editorTheme image values as filesystem
// paths, so an NRCC upload must reach settings.js as the absolute path of
// the stored file, while the mirror keeps the "/uploads/<name>" URL the UI
// previews.

func newThemeUploadsService(t *testing.T, withSettings bool) (*ConfigService, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "uploads"), 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"header.png", "legacy.png"} {
		if err := os.WriteFile(filepath.Join(dir, "uploads", name), []byte("png"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if withSettings {
		existing := "module.exports = {\n  uiPort: 1880,\n  editorTheme: {\n    page: { title: \"Old\" },\n  },\n}\n"
		if err := os.WriteFile(filepath.Join(dir, "settings.js"), []byte(existing), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return NewIsolatedConfigService(dir), dir
}

func themeWithImage(image string) map[string]any {
	return map[string]any{
		"page":         map[string]any{"title": "Brand", "favicon": image},
		"header":       map[string]any{"title": "Ops", "image": image},
		"login":        map[string]any{"image": image},
		"deployButton": map[string]any{"type": "simple", "label": "Go", "icon": image},
	}
}

func TestSave_EditorThemeUploadImages_RenderFilesystemPaths(t *testing.T) {
	cases := []struct {
		name  string
		image string
		// want is the rendered value; "<uploads>" is replaced by the
		// absolute uploads directory of the test service.
		want string
	}{
		{"upload url", "/uploads/header.png", "<uploads>/header.png"},
		{"legacy stripped upload", "/legacy.png", "<uploads>/legacy.png"},
		{"legacy name without upload stays", "/missing.png", "/missing.png"},
		{"absolute path elsewhere stays", "/opt/brand/logo.png", "/opt/brand/logo.png"},
		{"http url stays", "https://cdn.example.com/logo.png", "https://cdn.example.com/logo.png"},
		{"traversal is neutralized", "/uploads/../../etc/passwd", "<uploads>/passwd"},
	}
	for _, mode := range []struct {
		name         string
		withSettings bool
	}{{"patch", true}, {"first write", false}} {
		for _, tc := range cases {
			t.Run(mode.name+"/"+tc.name, func(t *testing.T) {
				svc, dir := newThemeUploadsService(t, mode.withSettings)
				cfg := model.DefaultNodeRedConfig()
				cfg.EditorTheme = themeWithImage(tc.image)
				if err := svc.Save(cfg); err != nil {
					t.Fatalf("Save: %v", err)
				}
				got := readFileString(t, filepath.Join(dir, "settings.js"))
				want := strconv.Quote(strings.ReplaceAll(tc.want, "<uploads>", filepath.Join(dir, "uploads")))
				for _, fragment := range []string{
					"title: \"Brand\", favicon: " + want,
					"header: { title: \"Ops\", image: " + want,
					"login: { image: " + want + " }",
					"label: \"Go\", icon: " + want,
				} {
					if !strings.Contains(got, fragment) {
						t.Errorf("settings.js missing %q:\n%s", fragment, got)
					}
				}
				if strings.Contains(got, "/etc/passwd") {
					t.Errorf("traversal reached settings.js:\n%s", got)
				}
			})
		}
	}
}

func TestEditorThemeUploadImages_ReadBackAsUploadURL(t *testing.T) {
	svc, dir := newThemeUploadsService(t, true)
	cfg := model.DefaultNodeRedConfig()
	cfg.EditorTheme = map[string]any{
		"header": map[string]any{"title": "Ops", "image": "/uploads/header.png"},
		"page":   map[string]any{"favicon": "/legacy.png"},
		"login":  map[string]any{"image": "https://cdn.example.com/login.png"},
	}
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertThemeImages(t, svc, "/uploads/header.png", "/uploads/legacy.png", "https://cdn.example.com/login.png")

	// A raw settings.js carrying the absolute upload path is mirrored back
	// as the upload URL the UI previews.
	raw := "module.exports = {\n  uiPort: 1880,\n  editorTheme: {\n" +
		"    header: { title: \"Ops\", image: " + strconv.Quote(filepath.Join(dir, "uploads", "header.png")) + " },\n" +
		"    page: { favicon: " + strconv.Quote(filepath.Join(dir, "uploads", "legacy.png")) + " },\n" +
		"    login: { image: \"/opt/brand/login.png\" },\n" +
		"  },\n}\n"
	if _, err := svc.SaveRawSettings(raw); err != nil {
		t.Fatalf("SaveRawSettings: %v", err)
	}
	assertThemeImages(t, svc, "/uploads/header.png", "/uploads/legacy.png", "/opt/brand/login.png")
}

func assertThemeImages(t *testing.T, svc *ConfigService, header, favicon, login string) {
	t.Helper()
	got, err := svc.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	theme, _ := got.EditorTheme.(map[string]any)
	section := func(key string) map[string]any { m, _ := theme[key].(map[string]any); return m }
	if v := section("header")["image"]; v != header {
		t.Errorf("header.image = %v, want %q", v, header)
	}
	if v := section("page")["favicon"]; v != favicon {
		t.Errorf("page.favicon = %v, want %q", v, favicon)
	}
	if v := section("login")["image"]; v != login {
		t.Errorf("login.image = %v, want %q", v, login)
	}
}
