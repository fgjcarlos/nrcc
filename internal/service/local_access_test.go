package service

import (
	"fmt"
	"testing"
)

func TestLocalAccessServiceDetectWithoutPortless(t *testing.T) {
	t.Parallel()

	svc := NewLocalAccessService(3000)
	svc.runner = fakeRunner{errors: map[string]error{"lookpath portless": fmt.Errorf("missing")}}

	status := svc.Detect()
	if status.PortlessAvailable {
		t.Fatal("expected portless to be unavailable")
	}
	if status.URL != "http://127.0.0.1:3000" {
		t.Fatalf("unexpected fallback URL: %s", status.URL)
	}
	if status.Mode != "direct" {
		t.Fatalf("unexpected mode: %s", status.Mode)
	}
}

func TestLocalAccessServiceEnsureConfigured(t *testing.T) {
	t.Parallel()

	svc := NewLocalAccessService(3000)
	svc.runner = fakeRunner{
		outputs: map[string]string{
			"portless proxy start":             "started",
			"portless alias nrcc 3000 --force": "aliased",
		},
	}

	status := svc.EnsureConfigured()
	if !status.Configured || !status.Operational {
		t.Fatalf("expected configured portless status, got %+v", status)
	}
	if status.Mode != "portless" {
		t.Fatalf("unexpected mode: %s", status.Mode)
	}
	if status.URL != "https://nrcc.localhost" {
		t.Fatalf("unexpected URL: %s", status.URL)
	}

	stored := svc.Status()
	if stored != status {
		t.Fatalf("stored status mismatch: %+v != %+v", stored, status)
	}
}
