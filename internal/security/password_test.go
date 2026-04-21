package security

import "testing"

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pass    string
		wantErr bool
		errMsg  string
	}{
		{"too short", "Ab1!", false, ""},          // 4 chars
		{"too short", "Ab1!x", false, ""},         // 5 chars — still short
		{"exactly 7", "Abcdef1", true, ""},        // 7 chars
		{"min length ok", "Abcdefg1", false, ""}, // 8 chars, 3 classes
		{"only lowercase", "abcdefgh", true, ""},
		{"only digits", "12345678", true, "too common"},
		{"lower+digit", "abcdef12", false, ""},
		{"upper+digit", "ABCDEF12", false, ""},
		{"lower+special", "abcdef!@", false, ""},
		{"common password", "password123", true, "too common"},
		{"common password case", "Password123", true, "too common"},
		{"common changeme", "changeme", true, "too common"},
		{"strong password", "MyP@ss2025!", false, ""},
		{"unicode lower+digit", "hellowld42", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.pass, func(t *testing.T) {
			err := ValidatePassword(tt.pass)
			// Fix: short passwords should error
			if len(tt.pass) < 8 {
				if err == nil {
					t.Fatalf("ValidatePassword(%q) = nil, want error for short password", tt.pass)
				}
				return
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidatePassword(%q) = nil, want error", tt.pass)
				}
				if tt.errMsg != "" && !strContains(err.Message, tt.errMsg) {
					t.Fatalf("ValidatePassword(%q) error = %q, want to contain %q", tt.pass, err.Message, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("ValidatePassword(%q) = %v, want nil", tt.pass, err)
				}
			}
		})
	}
}

func strContains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
