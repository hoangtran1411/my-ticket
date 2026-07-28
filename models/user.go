package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var JWTSecretKey = []byte("super-secret-devticket-jwt-key-2026")

type contextKey string

const UserContextKey contextKey = "user_claims"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "admin" or "user"
	CreatedAt    time.Time `json:"created_at"`
}

type JWTClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(username, role string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &JWTClaims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "DevTicket",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecretKey)
}

func ValidateToken(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return JWTSecretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	return claims, nil
}

func GetUserFromContext(ctx context.Context) *JWTClaims {
	if claims, ok := ctx.Value(UserContextKey).(*JWTClaims); ok {
		return claims
	}
	return nil
}

func IsAdminFromRequest(r *http.Request) bool {
	cookie, err := r.Cookie("jwt_token")
	if err != nil {
		return false
	}
	claims, err := ValidateToken(cookie.Value)
	if err != nil {
		return false
	}
	return claims.Role == "admin"
}

func CreateUserTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);`
	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	// Create default admin user if not exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	if err == nil && count == 0 {
		hashed, _ := HashPassword("admin123")
		nowStr := time.Now().Format("2006-01-02 15:04:05")
		_, _ = db.Exec("INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?)", "admin", hashed, "admin", nowStr)
	}

	return nil
}

func AuthenticateUser(db *sql.DB, username, password string) (*User, error) {
	var user User
	var createdAtStr string
	query := `SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?`
	err := db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &createdAtStr)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	if !CheckPasswordHash(password, user.PasswordHash) {
		return nil, errors.New("invalid username or password")
	}

	user.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	return &user, nil
}
