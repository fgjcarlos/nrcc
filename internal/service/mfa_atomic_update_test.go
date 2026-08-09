package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestMfaService_BeginEnrollment_Concurrent(t *testing.T) {
	svc, _ := newMfaTestSetup(t)
	userID := svc.authSvc.GetUserByUsername("admin").ID
	const callers = 8

	secrets := make([]string, callers)
	urls := make([]string, callers)
	errs := runConcurrent(t, callers, func(i int) error {
		var err error
		secrets[i], urls[i], err = svc.BeginEnrollment(userID)
		return err
	})

	winners := 0
	for i, err := range errs {
		if err == nil {
			winners++
			if secrets[i] == "" || !strings.HasPrefix(urls[i], "otpauth://totp/") {
				t.Errorf("winner %d returned invalid enrollment material", i)
			}
			continue
		}
		if !errors.Is(err, ErrMfaAlreadyEnrolled) {
			t.Errorf("caller %d: got %v, want ErrMfaAlreadyEnrolled", i, err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful enrollments = %d, want 1", winners)
	}

	data, err := svc.store.Read()
	if err != nil {
		t.Fatalf("read mfa store: %v", err)
	}
	if len(data.Enrollments) != 1 || !data.Enrollments[0].Pending {
		t.Fatalf("persisted enrollments = %+v, want one pending row", data.Enrollments)
	}
}

func TestMfaService_ConfirmEnrollment_Concurrent(t *testing.T) {
	svc, _ := newMfaTestSetup(t)
	userID := svc.authSvc.GetUserByUsername("admin").ID
	secret, _, err := svc.BeginEnrollment(userID)
	if err != nil {
		t.Fatalf("begin enrollment: %v", err)
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP code: %v", err)
	}
	const callers = 4

	recoveryCodes := make([][]string, callers)
	errs := runConcurrent(t, callers, func(i int) error {
		var err error
		recoveryCodes[i], err = svc.ConfirmEnrollment(userID, code)
		return err
	})

	winners := 0
	for i, err := range errs {
		if err == nil {
			winners++
			if len(recoveryCodes[i]) != MfaRecoveryCodeCount {
				t.Errorf("winner %d recovery-code count = %d, want %d", i, len(recoveryCodes[i]), MfaRecoveryCodeCount)
			}
			continue
		}
		if !errors.Is(err, ErrMfaAlreadyEnrolled) {
			t.Errorf("caller %d: got %v, want ErrMfaAlreadyEnrolled", i, err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful confirmations = %d, want 1", winners)
	}

	status, err := svc.Status(userID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !status.Enabled || status.RecoveryCodesRemaining != MfaRecoveryCodeCount {
		t.Fatalf("status = %+v, want enabled with %d recovery codes", status, MfaRecoveryCodeCount)
	}
}

func TestMfaService_Disable_Concurrent(t *testing.T) {
	svc, _ := newMfaTestSetup(t)
	userID := svc.authSvc.GetUserByUsername("admin").ID
	secret, _, err := svc.BeginEnrollment(userID)
	if err != nil {
		t.Fatalf("begin enrollment: %v", err)
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP code: %v", err)
	}
	if _, err := svc.ConfirmEnrollment(userID, code); err != nil {
		t.Fatalf("confirm enrollment: %v", err)
	}
	const callers = 8

	errs := runConcurrent(t, callers, func(_ int) error {
		return svc.Disable(userID, userID, "password123", false)
	})

	winners := 0
	for i, err := range errs {
		if err == nil {
			winners++
			continue
		}
		if !errors.Is(err, ErrMfaNotEnrolled) {
			t.Errorf("caller %d: got %v, want ErrMfaNotEnrolled", i, err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful disables = %d, want 1", winners)
	}

	status, err := svc.Status(userID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Enabled {
		t.Fatal("MFA remains enabled after concurrent disable")
	}
}

func TestMfaService_ConsumeRecoveryCode_Concurrent(t *testing.T) {
	svc, _ := newMfaTestSetup(t)
	userID := svc.authSvc.GetUserByUsername("admin").ID
	secret, _, err := svc.BeginEnrollment(userID)
	if err != nil {
		t.Fatalf("begin enrollment: %v", err)
	}
	totpCode, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP code: %v", err)
	}
	recoveryCodes, err := svc.ConfirmEnrollment(userID, totpCode)
	if err != nil {
		t.Fatalf("confirm enrollment: %v", err)
	}
	const callers = 8

	consumed := make([]bool, callers)
	errs := runConcurrent(t, callers, func(i int) error {
		var err error
		consumed[i], err = svc.ConsumeRecoveryCode(userID, recoveryCodes[0])
		return err
	})

	winners := 0
	for i, err := range errs {
		if consumed[i] {
			winners++
			if err != nil {
				t.Errorf("winner %d returned error: %v", i, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), "code already used") {
			t.Errorf("caller %d: got %v, want code already used error", i, err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful recovery-code consumers = %d, want exactly 1", winners)
	}

	ok, err := svc.ConsumeRecoveryCode(userID, recoveryCodes[0])
	if ok || err == nil || !strings.Contains(err.Error(), "code already used") {
		t.Fatalf("reuse result = (%v, %v), want false and code already used error", ok, err)
	}
	status, err := svc.Status(userID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.RecoveryCodesRemaining != MfaRecoveryCodeCount-1 {
		t.Fatalf("remaining recovery codes = %d, want %d", status.RecoveryCodesRemaining, MfaRecoveryCodeCount-1)
	}
}
