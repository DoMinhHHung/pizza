package auth

import (
	"context"
	"fmt"
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
		userID, err := v.authenticate(ctx)
		if err != nil {
			return nil, err
		}
		ctx = context.WithValue(ctx, UserIDKey, userID)
		return handler(ctx, req)
	}
}

func (v *JWTValidator) authenticate(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing authorization")
	}

	tokenStr := values[0]
	if strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
		tokenStr = strings.TrimSpace(tokenStr[7:])
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return v.secret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, status.Error(codes.Unauthenticated, "invalid token")
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

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "user not in context")
	}
	return id, nil
}
