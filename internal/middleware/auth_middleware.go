package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

type contextKey string

const UserIDKey contextKey = "userID"

// AuthMiddleware validates JWT Bearer tokens and injects userID into context
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
				logger.Warn().Msg("Missing auth header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"Invalid authorization format"}`, http.StatusUnauthorized)
				logger.Warn().Msg("Invalid auth format")
				return
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate algorithm
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				logger.Warn().Err(err).Msg("Invalid token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"Invalid token claims"}`, http.StatusUnauthorized)
				logger.Warn().Msg("Invalid claims")
				return
			}

			userID, ok := claims["user_id"].(float64)
			if !ok {
				http.Error(w, `{"error":"Missing user_id in token"}`, http.StatusUnauthorized)
				logger.Warn().Msg("Missing user_id claim")
				return
			}

			// Inject userID into context
			ctx := context.WithValue(r.Context(), UserIDKey, int64(userID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts userID from request context
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}
