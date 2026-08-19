package server

import (
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestCountEncryptedEntries covers the helper that powers the startup
// WARN logged when NRCC_ENCRYPTION_KEY is empty but the persisted
// config still contains Encrypted: true entries (#664). The function is
// a one-liner but the contract is the load-bearing part of the
// operator-facing warning, so it gets an explicit test instead of
// relying on a side effect of NewServerWithConfig.
func TestCountEncryptedEntries(t *testing.T) {
	tests := []struct {
		name string
		in   []model.EnvVar
		want int
	}{
		{name: "empty", in: nil, want: 0},
		{name: "no encrypted", in: []model.EnvVar{
			{Key: "A", Value: "1"},
			{Key: "B", Value: "2"},
		}, want: 0},
		{name: "all encrypted", in: []model.EnvVar{
			{Key: "A", Value: "x", Encrypted: true},
			{Key: "B", Value: "y", Encrypted: true},
			{Key: "C", Value: "z", Encrypted: true},
		}, want: 3},
		{name: "mixed", in: []model.EnvVar{
			{Key: "A", Value: "1"},
			{Key: "B", Value: "x", Encrypted: true},
			{Key: "C", Value: "y", Encrypted: true},
			{Key: "D", Value: "2"},
		}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countEncryptedEntries(tt.in); got != tt.want {
				t.Errorf("countEncryptedEntries = %d, want %d", got, tt.want)
			}
		})
	}
}
