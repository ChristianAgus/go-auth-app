package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"auth-app/internal/service"
)

const testSecret = "test-secret-test-secret-test-secret-1234"

func makeToken(t *testing.T, sub string, exp time.Time) string {
	t.Helper()
	claims := jwt.RegisteredClaims{Subject: sub, ExpiresAt: jwt.NewNumericDate(exp)}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAuth(t *testing.T) {
	// ParseToken nggak butuh database, jadi store-nya boleh nil
	authSvc := service.NewAuthService(nil, testSecret, time.Hour)

	valid := makeToken(t, "42", time.Now().Add(time.Hour))
	expired := makeToken(t, "42", time.Now().Add(-time.Minute))

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"tanpa header", "", http.StatusUnauthorized},
		{"tanpa token", "Bearer", http.StatusUnauthorized},
		{"skema salah", "Basic " + valid, http.StatusUnauthorized},
		{"token sampah", "Bearer sampah", http.StatusUnauthorized},
		{"token expired", "Bearer " + expired, http.StatusUnauthorized},
		{"token valid", "Bearer " + valid, http.StatusOK},
		{"bearer huruf kecil", "bearer " + valid, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotID uint64
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotID, _ = UserIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			Auth(authSvc)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, harusnya %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && gotID != 42 {
				t.Errorf("user ID di context = %d, harusnya 42", gotID)
			}
		})
	}
}