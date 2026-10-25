package model

// RedactedSecret is the placeholder a client may send back to say
// "keep the stored value". The configuration API never returns a
// write-only secret, so a client that round-trips a response it just
// read would otherwise wipe the secret it never saw (issue #877).
const RedactedSecret = "[redacted]"

// redactedSecretMarkers lists every value the API accepts as "the secret
// was not sent". An empty value is part of the set because the UI
// submits an empty string when the operator leaves a write-only field
// blank.
var redactedSecretMarkers = []string{
	"",
	RedactedSecret,
	"[REDACTED]",
	"********",
}

// IsRedactedSecret reports whether value means "keep the stored secret"
// rather than "replace the stored secret".
//
// The literal "false" is NOT a marker: Node-RED uses it to disable
// credential encryption, so it must rotate the stored value.
func IsRedactedSecret(value string) bool {
	for _, marker := range redactedSecretMarkers {
		if value == marker {
			return true
		}
	}
	return false
}

// RedactedSecrets returns a copy of the configuration with the write-only
// secrets cleared: the credential-encryption secret and the TLS private-key
// passphrase.
//
// The copy is deliberate. Callers redact a response payload without
// mutating the configuration they just read from the store — the stored
// secrets must survive the read (issue #877). Non-secret TLS material
// (key/cert/ca paths and the port) is preserved so the configuration UI
// can still render the TLS section.
func (c NodeRedConfig) RedactedSecrets() NodeRedConfig {
	redacted := c
	redacted.CredentialSecret = ""
	if c.Https != nil {
		https := *c.Https
		https.Passphrase = ""
		redacted.Https = &https
	}
	return redacted
}
