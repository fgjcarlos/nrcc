package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestDeleteUser_LastAdminReturnsForbidden(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	admin := authSvc.GetUserByUsername("admin")

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/users/"+admin.ID, nil)
	req.SetPathValue("id", admin.ID)
	req = req.WithContext(context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{
		UserID:   "requesting-admin-id",
		Username: "requesting-admin",
		Role:     model.RoleAdmin,
	}))
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for deleting last admin, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope model.ApiErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Error == nil || envelope.Error.Code != "CANNOT_DELETE_LAST_ADMIN" {
		t.Fatalf("expected CANNOT_DELETE_LAST_ADMIN error, got %#v", envelope.Error)
	}
}
