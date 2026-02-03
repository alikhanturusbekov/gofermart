package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

func TestAuthMiddleware(t *testing.T) {
	secretKey := "test-secret"

	// A handler that just writes "ok"
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, ok := UserIDFromContext(r.Context()); !ok {
			t.Error("userID not set in context")
		}
	})

	validUserID := uuid.New().String()
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": validUserID,
	})
	validTokenStr, _ := validToken.SignedString([]byte(secretKey))

	tests := []struct {
		name         string
		authHeader   string
		expectedCode int
	}{
		{"No Authorization header", "", http.StatusUnauthorized},
		{"Malformed header", "InvalidToken", http.StatusUnauthorized},
		{"Bearer with empty token", "Bearer ", http.StatusUnauthorized},
		{"Invalid token", "Bearer abc.def.ghi", http.StatusUnauthorized},
		{"Wrong signing method", func() string {
			tk := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"user_id": validUserID})
			s, _ := tk.SignedString([]byte(secretKey))
			return "Bearer " + s
		}(), http.StatusUnauthorized},
		{"Missing user_id claim", func() string {
			tk := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"foo": "bar"})
			s, _ := tk.SignedString([]byte(secretKey))
			return "Bearer " + s
		}(), http.StatusUnauthorized},
		{"Invalid UUID in user_id", func() string {
			tk := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": "not-a-uuid"})
			s, _ := tk.SignedString([]byte(secretKey))
			return "Bearer " + s
		}(), http.StatusUnauthorized},
		{"Valid token", "Bearer " + validTokenStr, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			mw := AuthMiddleware(secretKey)(okHandler)
			mw.ServeHTTP(rec, req)

			if rec.Code != tt.expectedCode {
				t.Errorf("got status %d, want %d", rec.Code, tt.expectedCode)
			}
		})
	}
}
