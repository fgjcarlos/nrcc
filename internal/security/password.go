package security

import (
	"fmt"
	"strings"
	"unicode"
)

// commonPasswords is a small list of the most commonly used passwords.
// Kept deliberately short — this is a local tool, not a bank.
var commonPasswords = map[string]struct{}{
	"password": {}, "12345678": {}, "123456789": {}, "1234567890": {}, // pragma: allowlist secret
	"qwerty123": {}, "password1": {}, "iloveyou": {}, "sunshine": {}, // pragma: allowlist secret
	"princess": {}, "football": {}, "charlie1": {}, "trustno1": {}, // pragma: allowlist secret
	"dragon12": {}, "baseball": {}, "abc12345": {}, "monkey12": {}, // pragma: allowlist secret
	"letmein1": {}, "shadow12": {}, "master12": {}, "qwertyui": {}, // pragma: allowlist secret
	"michael1": {}, "superman": {}, "1qaz2wsx": {}, "jennifer": {}, // pragma: allowlist secret
	"hunter12": {}, "thomas12": {}, "password123": {}, "admin123": {}, // pragma: allowlist secret
	"welcome1": {}, "passw0rd": {}, "starwars": {}, "whatever": {}, // pragma: allowlist secret
	"computer": {}, "corvette": {}, "12341234": {}, "88888888": {}, // pragma: allowlist secret
	"87654321": {}, "abcdefgh": {}, "11111111": {}, "22222222": {}, // pragma: allowlist secret
	"33333333": {}, "44444444": {}, "55555555": {}, "66666666": {}, // pragma: allowlist secret
	"77777777": {}, "99999999": {}, "00000000": {}, "qwerty12": {}, // pragma: allowlist secret
	"iloveu12": {}, "trustme1": {}, "changeme": {}, "admin1234": {}, // pragma: allowlist secret
	"password12": {}, "letmein12": {}, "welcome12": {}, "monkey123": {}, // pragma: allowlist secret
	"dragon123": {}, "master123": {}, "qwerty1234": {}, "password1234": {}, // pragma: allowlist secret
	"abc123456": {}, "654321abc": {}, "123abc456": {}, "pass1234": {}, // pragma: allowlist secret
	"test1234": {}, "hello123": {}, "p@ssw0rd": {}, "p@ssword": {}, // pragma: allowlist secret
	"Pa$$w0rd": {}, "asdfghjk": {}, "zxcvbnm1": {}, // pragma: allowlist secret
}

// PasswordValidationError contains structured information about why a password
// was rejected.
type PasswordValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *PasswordValidationError) Error() string {
	return e.Message
}

// ValidatePassword checks a password against the password policy:
//   - Minimum 8 characters
//   - At least 2 of: lowercase, uppercase, digits, special characters
//   - Not in the common passwords list
func ValidatePassword(password string) *PasswordValidationError {
	if len(password) < 8 {
		return &PasswordValidationError{
			Field:   "password",
			Message: "password must be at least 8 characters",
		}
	}

	if _, found := commonPasswords[strings.ToLower(password)]; found {
		return &PasswordValidationError{
			Field:   "password",
			Message: "password is too common, please choose a stronger one",
		}
	}

	classes := countCharClasses(password)
	if classes < 2 {
		return &PasswordValidationError{
			Field:   "password",
			Message: fmt.Sprintf("password must contain at least 2 character types (lowercase, uppercase, digits, special); found %d", classes),
		}
	}

	return nil
}

func countCharClasses(s string) int {
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, ch := range s {
		switch {
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsDigit(ch):
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	count := 0
	if hasLower {
		count++
	}
	if hasUpper {
		count++
	}
	if hasDigit {
		count++
	}
	if hasSpecial {
		count++
	}
	return count
}
