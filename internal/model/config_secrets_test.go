package model

import "testing"

func TestNodeRedConfig_RedactedSecrets_ClearsWriteOnlySecrets(t *testing.T) {
	cfg := NodeRedConfig{
		Port:             1880,
		CredentialSecret: "audit-credential-fixture",
		Https: &HttpsConfig{
			Key:        "/tmp/key.pem",
			Cert:       "/tmp/cert.pem",
			Port:       1883,
			Passphrase: "audit-tls-fixture",
		},
	}

	redacted := cfg.RedactedSecrets()

	if redacted.CredentialSecret != "" {
		t.Errorf("credentialSecret = %q, want it redacted", redacted.CredentialSecret)
	}
	if redacted.Https == nil {
		t.Fatal("https block must be preserved")
	}
	if redacted.Https.Passphrase != "" {
		t.Errorf("https.passphrase = %q, want it redacted", redacted.Https.Passphrase)
	}
	if redacted.Https.Key != "/tmp/key.pem" || redacted.Https.Cert != "/tmp/cert.pem" {
		t.Errorf("non-secret TLS material must be preserved; got %+v", redacted.Https)
	}
	if redacted.Https.Port != 1883 {
		t.Errorf("https.port = %d, want it preserved", redacted.Https.Port)
	}
	if redacted.Port != 1880 {
		t.Errorf("port = %d, want it preserved", redacted.Port)
	}
}

func TestNodeRedConfig_RedactedSecrets_DoesNotMutateReceiver(t *testing.T) {
	cfg := NodeRedConfig{
		CredentialSecret: "audit-credential-fixture",
		Https:            &HttpsConfig{Passphrase: "audit-tls-fixture"},
	}

	_ = cfg.RedactedSecrets()

	if cfg.CredentialSecret != "audit-credential-fixture" {
		t.Errorf("receiver credentialSecret = %q, want it untouched", cfg.CredentialSecret)
	}
	if cfg.Https == nil || cfg.Https.Passphrase != "audit-tls-fixture" {
		t.Errorf("receiver https.passphrase = %+v, want it untouched", cfg.Https)
	}
}

func TestNodeRedConfig_RedactedSecrets_HandlesNilHTTPS(t *testing.T) {
	cfg := NodeRedConfig{Port: 1880}

	redacted := cfg.RedactedSecrets()

	if redacted.Https != nil {
		t.Errorf("https = %+v, want nil", redacted.Https)
	}
}

func TestIsRedactedSecret(t *testing.T) {
	for _, value := range []string{"", RedactedSecret, "[REDACTED]", "********"} {
		if !IsRedactedSecret(value) {
			t.Errorf("IsRedactedSecret(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"audit-credential-fixture", "false", "secret"} {
		if IsRedactedSecret(value) {
			t.Errorf("IsRedactedSecret(%q) = true, want false", value)
		}
	}
}
