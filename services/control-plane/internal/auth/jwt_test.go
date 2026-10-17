package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testSecret = "test-secret-at-least-32-characters-long!!"

func signToken(t *testing.T, secret string, claims *Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestParseAccessToken_Valid(t *testing.T) {
	v := NewJWTValidator(testSecret)
	userID := uuid.New()
	jti := uuid.New().String()

	raw := signToken(t, testSecret, &Claims{
		Email:  "user@example.com",
		Status: "ACTIVE",
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})

	got, err := v.ParseAccessToken("Bearer " + raw)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if got != userID {
		t.Fatalf("userID: want %s got %s", userID, got)
	}
}

func TestParseAccessToken_WithoutBearerPrefix(t *testing.T) {
	v := NewJWTValidator(testSecret)
	userID := uuid.New()

	raw := signToken(t, testSecret, &Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	got, err := v.ParseAccessToken(raw)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if got != userID {
		t.Fatalf("userID: want %s got %s", userID, got)
	}
}

func TestParseAccessToken_Empty(t *testing.T) {
	v := NewJWTValidator(testSecret)
	_, err := v.ParseAccessToken("")
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestParseAccessToken_RejectRefreshType(t *testing.T) {
	v := NewJWTValidator(testSecret)
	raw := signToken(t, testSecret, &Claims{
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	_, err := v.ParseAccessToken("Bearer " + raw)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestParseAccessToken_Expired(t *testing.T) {
	v := NewJWTValidator(testSecret)
	raw := signToken(t, testSecret, &Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	})

	_, err := v.ParseAccessToken("Bearer " + raw)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	v := NewJWTValidator(testSecret)
	raw := signToken(t, "another-secret-at-least-32-characters-xx", &Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	_, err := v.ParseAccessToken("Bearer " + raw)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestParseAccessToken_InvalidSubject(t *testing.T) {
	v := NewJWTValidator(testSecret)
	raw := signToken(t, testSecret, &Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "not-a-uuid",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	_, err := v.ParseAccessToken("Bearer " + raw)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestAuthenticateFromMD_Success(t *testing.T) {
	v := NewJWTValidator(testSecret)
	userID := uuid.New()
	raw := signToken(t, testSecret, &Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer "+raw,
	))

	got, err := v.authenticateFromMD(ctx)
	if err != nil {
		t.Fatalf("authenticateFromMD: %v", err)
	}
	if got != userID {
		t.Fatalf("userID: want %s got %s", userID, got)
	}
}

func TestAuthenticateFromMD_MissingMetadata(t *testing.T) {
	v := NewJWTValidator(testSecret)
	_, err := v.authenticateFromMD(context.Background())
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestUserIDFromContext(t *testing.T) {
	userID := uuid.New()
	ctx := WithUserID(context.Background(), userID)

	got, err := UserIDFromContext(ctx)
	if err != nil {
		t.Fatalf("UserIDFromContext: %v", err)
	}
	if got != userID {
		t.Fatalf("want %s got %s", userID, got)
	}

	_, err = UserIDFromContext(context.Background())
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestMiddleware_Authorized(t *testing.T) {
	v := NewJWTValidator(testSecret)
	userID := uuid.New()
	raw := signToken(t, testSecret, &Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		got, err := UserIDFromContext(r.Context())
		if err != nil {
			t.Fatalf("UserIDFromContext in handler: %v", err)
		}
		if got != userID {
			t.Fatalf("want %s got %s", userID, got)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()

	v.Middleware(next).ServeHTTP(rr, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestMiddleware_Unauthorized(t *testing.T) {
	v := NewJWTValidator(testSecret)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs", nil)
	rr := httptest.NewRecorder()

	v.Middleware(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rr.Code)
	}
}
