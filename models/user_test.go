package models

import (
	"context"
	"testing"
	"time"
)

// ─── Password helpers ─────────────────────────────────────────────────────────

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}

	if !CheckPasswordHash("secret123", hash) {
		t.Error("CheckPasswordHash: correct password should return true")
	}
	if CheckPasswordHash("wrongpassword", hash) {
		t.Error("CheckPasswordHash: wrong password should return false")
	}
}

func TestHashPassword_DifferentEachCall(t *testing.T) {
	h1, _ := HashPassword("mypassword")
	h2, _ := HashPassword("mypassword")
	// bcrypt produces different salted hashes each time
	if h1 == h2 {
		t.Error("expected different hashes for same password (bcrypt adds salt)")
	}
}

// ─── JWT helpers ──────────────────────────────────────────────────────────────

const testAdminUser = "admin"

func TestGenerateAndValidateToken(t *testing.T) {
	tokenStr, err := GenerateToken(testAdminUser, testAdminUser)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if tokenStr == "" {
		t.Error("expected non-empty token string")
	}

	claims, err := ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.Username != testAdminUser {
		t.Errorf("Username: got %q, want %q", claims.Username, testAdminUser)
	}
	if claims.Role != testAdminUser {
		t.Errorf("Role: got %q, want %q", claims.Role, testAdminUser)
	}
	if claims.Issuer != "DevTicket" {
		t.Errorf("Issuer: got %q, want %q", claims.Issuer, "DevTicket")
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	_, err := ValidateToken("not.a.valid.token")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
}

func TestValidateToken_TamperedToken(t *testing.T) {
	// Valid token with a different signing key (manually crafted garbage)
	_, err := ValidateToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIn0.INVALIDSIGNATURE")
	if err == nil {
		t.Error("expected error for tampered token, got nil")
	}
}

func TestGenerateToken_UserRole(t *testing.T) {
	token, err := GenerateToken("johndoe", "user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.Role != "user" {
		t.Errorf("expected role 'user', got %q", claims.Role)
	}
}

// ─── Context helpers ──────────────────────────────────────────────────────────

func TestGetUserFromContext_WithClaims(t *testing.T) {
	expected := &JWTClaims{Username: testAlice, Role: "admin"}
	ctx := context.WithValue(context.Background(), UserContextKey, expected)

	got := GetUserFromContext(ctx)
	if got == nil {
		t.Fatal("expected claims, got nil")
	}
	if got.Username != testAlice {
		t.Errorf("Username: got %q, want %q", got.Username, testAlice)
	}
}

func TestGetUserFromContext_WithoutClaims(t *testing.T) {
	got := GetUserFromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil claims from empty context, got %+v", got)
	}
}

func TestGetUserFromContext_WrongType(t *testing.T) {
	// Store a wrong type under the key
	ctx := context.WithValue(context.Background(), UserContextKey, "not-a-claims-struct")
	got := GetUserFromContext(ctx)
	if got != nil {
		t.Error("expected nil for wrong value type, got non-nil")
	}
}

// ─── User DB tests ────────────────────────────────────────────────────────────

func TestCreateUserTable_AndSeedAdmin(t *testing.T) {
	db := setupTestDB(t)

	// CreateUserTable also seeds the default admin
	if err := CreateUserTable(db); err != nil {
		t.Fatalf("CreateUserTable: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users WHERE username='admin'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 admin user, got %d", count)
	}
}

func TestCreateUserTable_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	if err := CreateUserTable(db); err != nil {
		t.Fatalf("first CreateUserTable: %v", err)
	}
	// Calling it again should not fail or duplicate
	if err := CreateUserTable(db); err != nil {
		t.Fatalf("second CreateUserTable: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users WHERE username='admin'").Scan(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 admin after two calls, got %d", count)
	}
}

func TestAuthenticateUser_Success(t *testing.T) {
	db := setupTestDB(t)
	if err := CreateUserTable(db); err != nil {
		t.Fatalf("CreateUserTable: %v", err)
	}

	user, err := AuthenticateUser(db, testAdminUser, "admin123")
	if err != nil {
		t.Fatalf("AuthenticateUser with valid creds: %v", err)
	}
	if user.Username != testAdminUser {
		t.Errorf("Username: got %q, want %q", user.Username, testAdminUser)
	}
	if user.Role != testAdminUser {
		t.Errorf("Role: got %q, want %q", user.Role, testAdminUser)
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestAuthenticateUser_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	_ = CreateUserTable(db)

	_, err := AuthenticateUser(db, "admin", "wrongpassword")
	if err == nil {
		t.Error("expected error for wrong password, got nil")
	}
}

func TestAuthenticateUser_UnknownUser(t *testing.T) {
	db := setupTestDB(t)
	_ = CreateUserTable(db)

	_, err := AuthenticateUser(db, "nobody", "anything")
	if err == nil {
		t.Error("expected error for unknown user, got nil")
	}
}

func TestAuthenticateUser_CustomUser(t *testing.T) {
	db := setupTestDB(t)
	_ = CreateUserTable(db)

	// Insert a non-admin user
	hash, _ := HashPassword("pass123")
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		"INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?)",
		"johndoe", hash, "user", nowStr,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	user, err := AuthenticateUser(db, "johndoe", "pass123")
	if err != nil {
		t.Fatalf("AuthenticateUser: %v", err)
	}
	if user.Role != "user" {
		t.Errorf("Role: got %q, want %q", user.Role, "user")
	}
}
