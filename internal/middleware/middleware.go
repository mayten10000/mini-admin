package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"mini-admin/internal/utils"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				utils.ErrorJSON(w, http.StatusUnauthorized, "Missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid token claims")
				return
			}

			userID, ok := claims["sub"].(float64)
			if !ok {
				utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid token subject")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, int64(userID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) int64 {
	id, _ := r.Context().Value(UserIDKey).(int64)
	return id
}

func GenerateAccessToken(secret string, userID int64, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(ttl).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ActiveUserMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r)
			if userID == 0 {
				utils.ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			var status string
			err := db.QueryRow("SELECT status FROM users WHERE id=$1", userID).Scan(&status)
			if err != nil {
				utils.ErrorJSON(w, http.StatusUnauthorized, "User not found")
				return
			}
			if status != "active" {
				utils.ErrorJSON(w, http.StatusForbidden, "Account is disabled")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
