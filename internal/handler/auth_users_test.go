package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestGetUsersContract(t *testing.T) {
	h, _ := setupAuthTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/users", nil)
	rec := httptest.NewRecorder()

	h.GetUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response model.ApiResponse[UserListResponse]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || len(response.Data.Users) != 1 {
		t.Fatalf("expected successful data.users response with one user, got %#v", response)
	}
}

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

func TestDeleteUser_RejectsCurrentOperator(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	admin := authSvc.GetUserByUsername("admin")
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/users/"+admin.ID, nil)
	req.SetPathValue("id", admin.ID)
	req = req.WithContext(context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{
		UserID: admin.ID, Username: admin.Username, Role: model.RoleAdmin,
	}))
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var response model.ApiErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != "CANNOT_DELETE_SELF" {
		t.Fatalf("expected CANNOT_DELETE_SELF, got %#v", response.Error)
	}
	if authSvc.GetUserByID(admin.ID) == nil {
		t.Fatal("current operator was deleted")
	}
}

func TestUpdateUser_RejectsCurrentOperatorDemotion(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	admin := authSvc.GetUserByUsername("admin")
	role := model.RoleViewer
	body, err := json.Marshal(UpdateUserRequest{Role: &role})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/auth/users/"+admin.ID, bytes.NewReader(body))
	req.SetPathValue("id", admin.ID)
	req = req.WithContext(context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{
		UserID: admin.ID, Username: admin.Username, Role: model.RoleAdmin,
	}))
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var response model.ApiErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != "CANNOT_CHANGE_OWN_ROLE" {
		t.Fatalf("expected CANNOT_CHANGE_OWN_ROLE, got %#v", response.Error)
	}
	if user := authSvc.GetUserByID(admin.ID); user == nil || user.Role != model.RoleAdmin {
		t.Fatalf("current operator role changed: %#v", user)
	}
}
