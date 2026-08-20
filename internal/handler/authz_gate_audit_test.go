package handler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoInlineAdminGate is the static counterpart to the chi.Walk test in
// internal/server. It walks the AST of every file in this package and
// asserts that no handler returns 403 FORBIDDEN based on an inline
// `claims.Role != model.RoleAdmin` check.
//
// The two documented exceptions (#666) keep their inline admin/role logic
// because they implement CONTENT REDACTION (a different response shape for
// non-admins), not authz rejection:
//
//	internal/handler/backup.go:GetBackupProvider    — MEDIUM-017 returns {provider:null} to viewers
//	internal/handler/config.go:GetConfig           — MEDIUM-015 redacts passwords + env values
//	internal/handler/system.go:GetSystemInfo        — MEDIUM-018 blanks hostname for viewers
//
// The list below is referenced by exact (file, line) anchors. When a new
// exception is added intentionally, append it here AND in the PR
// description so reviewers can audit it.
var documentedInlineExceptions = map[string]string{
	"internal/handler/backup.go": "MEDIUM-017: GetBackupProvider degrades response for non-admins",
	"internal/handler/config.go": "MEDIUM-015: GetConfig redacts secrets for non-admins",
	"internal/handler/system.go": "MEDIUM-018: GetSystemInfo redacts hostname for non-admins",
}

// TestNoInlineAdminGate asserts the AST of internal/handler/ contains no
// 403 FORBIDDEN response preceded by a `claims.Role != model.RoleAdmin`
// branch, except in the documented redaction exceptions above.
//
// This catches the failure mode #666 was created to prevent: someone adding
// a new endpoint and reaching for the "easy" inline role check instead of
// wiring middleware.RequireAdmin on the route.
func TestNoInlineAdminGate(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if !strings.HasSuffix(file, ".go") {
			continue
		}

		astFile, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}

		// Look for IfStmt whose condition references claims.Role !=
		// model.RoleAdmin, and which contains a RespondError(StatusForbidden,
		// "FORBIDDEN", ...) call. That's the forbidden shape.
		ast.Inspect(astFile, func(n ast.Node) bool {
			ifStmt, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}
			if !guardsRoleRejection(ifStmt.Cond) {
				return true
			}

			// If we got here, this if-statement guards a Role != Admin
			// check. Walk its body for a 403 RespondError.
			if hasForbiddenRespondError(ifStmt.Body) {
				relPath := filepath.Join("internal", "handler", file)
				if _, allowed := documentedInlineExceptions[relPath]; !allowed {
					t.Errorf("%s: inline Role != model.RoleAdmin rejection detected. Move to middleware.RequireAdmin on the route. If this is a redaction (different response shape for non-admin), update documentedInlineExceptions and the PR description (#666).", relPath)
				}
			}
			return true
		})
	}
}

// guardsRoleRejection returns true when expr is a comparison against
// `claims.Role != model.RoleAdmin` (the canonical inline-gate shape).
// We don't try to be exhaustive — the goal is to catch the common shape
// before it ships, not to silence every creative variant.
func guardsRoleRejection(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	if !refsClaimsRole(bin.X) && !refsClaimsRole(bin.Y) {
		return false
	}
	other := bin.X
	if refsClaimsRole(bin.X) {
		other = bin.Y
	}
	return refsModelRoleAdmin(other)
}

func refsClaimsRole(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := x.X.(*ast.Ident)
	if !ok || id.Name != "claims" {
		return false
	}
	if x.Sel.Name != "Role" {
		return false
	}
	return sel.Sel.Name == "Role"
}

func refsModelRoleAdmin(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != "model" {
		return false
	}
	return sel.Sel.Name == "RoleAdmin"
}

// hasForbiddenRespondError walks a block (or single stmt) looking for a
// model.RespondError call with status http.StatusForbidden and code
// "FORBIDDEN". That's the canonical inline-gate response.
func hasForbiddenRespondError(body ast.Node) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isModelRespondError(call) {
			return true
		}
		// Arg shape: (w, status, code, msg). status is http.StatusForbidden;
		// code is the string literal "FORBIDDEN".
		if len(call.Args) < 3 {
			return true
		}
		if !isHTTPStatusForbidden(call.Args[1]) {
			return true
		}
		if !isStringLiteral(call.Args[2], "FORBIDDEN") {
			return true
		}
		found = true
		return false
	})
	return found
}

func isModelRespondError(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok || id.Name != "model" {
		return false
	}
	return sel.Sel.Name == "RespondError"
}

func isHTTPStatusForbidden(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != "http" {
		return false
	}
	return sel.Sel.Name == "StatusForbidden"
}

func isStringLiteral(expr ast.Expr, want string) bool {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	return lit.Value == `"`+want+`"`
}