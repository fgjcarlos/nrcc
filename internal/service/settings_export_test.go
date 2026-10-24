package service

import (
	"testing"
)

// TestExportSettingsObject_ReportsDeclaredKeys asserts that the sandbox
// export knows the exact top-level keys the source declares.
func TestExportSettingsObject_ReportsDeclaredKeys(t *testing.T) {
	entries, err := exportSettingsObject(`module.exports = {
  uiPort: 7777,
  editorTheme: { projects: { enabled: true } },
  env: { PLAIN: 'value' },
};
`)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	for _, key := range []string{"uiPort", "editorTheme", "env"} {
		if _, ok := entries[key]; !ok {
			t.Fatalf("the export does not report %q: %+v", key, entries)
		}
	}
	if _, ok := entries["flowFile"]; ok {
		t.Fatalf("the export reports a key the source does not declare: %+v", entries)
	}
}

// TestExportSettingsObject_BlocksHostAPI asserts that the inspector never
// grants a host API to the source.
func TestExportSettingsObject_BlocksHostAPI(t *testing.T) {
	if _, err := exportSettingsObject(`module.exports = { uiPort: require('net') };`); err == nil {
		t.Fatal("the inspector executed a blocked host API")
	}
}
