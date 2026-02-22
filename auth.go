package main

import (
	"SuperChat/internal/httpapi"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Hardcoded signing secret per project requirement (not env/flag driven).
var jwtSecret = []byte("f6f8a9e06f3f6a4f0f95ce9a7d2f3d58b7f0a6c98f9469d0f64bb6f0a3f08b22")

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	jwt.RegisteredClaims
}

// generateToken creates a new JWT token for a user
func generateToken(user *User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Name:     user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   user.Name,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// validateToken validates a JWT token and returns the claims
func validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// checkPassword checks if a password matches a hash
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// authMiddleware is middleware that checks for valid JWT token
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Bearer token required", http.StatusUnauthorized)
			return
		}

		claims, err := validateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "name", claims.Name)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// getUserFromContext extracts user information from request context
func getUserFromContext(r *http.Request) (string, string) {
	userID := r.Context().Value("user_id").(string)
	username := r.Context().Value("username").(string)
	return userID, username
}

// getNameFromContext returns the display name if set, otherwise the username.
func getNameFromContext(r *http.Request) string {
	if v := r.Context().Value("name"); v != nil {
		if name, ok := v.(string); ok && name != "" {
			return name
		}
	}
	if v := r.Context().Value("username"); v != nil {
		if uname, ok := v.(string); ok {
			return uname
		}
	}
	return ""
}

// writeJSONResponse writes a JSON response with proper headers
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	httpapi.WriteJSON(w, statusCode, data)
}

// writeErrorResponse writes an error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	httpapi.WriteError(w, statusCode, message)
}

// writeSuccessResponse writes a success response
func writeSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	httpapi.WriteSuccess(w, data, message)
}
