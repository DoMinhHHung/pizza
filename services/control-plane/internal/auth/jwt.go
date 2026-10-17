package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserIDKey contextKey = "userID"

type Claims struct {
	Email  string `json:"email"`
	Status string `json:"status"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

type JWTValidator struct {
	secret []byte
}

func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{secret: []byte(secret)}
}

func (v *JWTValidator) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		userID, err := v.authenticateFromMD(ctx)
		if err != nil {
			return nil, err
		}
		ctx = context.WithValue(ctx, UserIDKey, userID)
		return handler(ctx, req)
	}
}

func (v *JWTValidator) authenticateFromMD(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing authorization")
	}
	return v.ParseAccessToken(values[0])
}

func (v *JWTValidator) ParseAccessToken(raw string) (uuid.UUID, error) {
	tokenStr := strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
		tokenStr = strings.TrimSpace(tokenStr[7:])
	}
	if tokenStr == "" {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing authorization")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.secret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}
	if claims.Type != "" && claims.Type != "access" {
		return uuid.Nil, status.Error(codes.Unauthenticated, "not an access token")
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "invalid subject")
	}
	return userID, nil
}

func (v *JWTValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := v.ParseAccessToken(r.Header.Get("Authorization"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "user not in context")
	}
	return id, nil
}

func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}
