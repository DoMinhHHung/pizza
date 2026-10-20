package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

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

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWTValidator struct {
	jwksURL  string
	cacheTTL time.Duration
	client   *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewJWTValidator(jwksURL string, cacheTTL time.Duration) *JWTValidator {
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &JWTValidator{
		jwksURL:  jwksURL,
		cacheTTL: cacheTTL,
		client:   &http.Client{Timeout: 5 * time.Second},
		keys:     make(map[string]*rsa.PublicKey),
	}
}

func NewJWTValidatorWithKeys(keys map[string]*rsa.PublicKey) *JWTValidator {
	return &JWTValidator{
		cacheTTL:  time.Hour,
		keys:      keys,
		fetchedAt: time.Now(),
	}
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
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			kid, _ := t.Header["kid"].(string)
			return v.publicKey(kid)
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
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

func (v *JWTValidator) publicKey(kid string) (*rsa.PublicKey, error) {
	if err := v.ensureKeys(); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if kid != "" {
		if k, ok := v.keys[kid]; ok {
			return k, nil
		}
		return nil, fmt.Errorf("unknown kid %q", kid)
	}
	if len(v.keys) == 1 {
		for _, k := range v.keys {
			return k, nil
		}
	}
	return nil, fmt.Errorf("missing kid and multiple keys")
}

func (v *JWTValidator) ensureKeys() error {
	v.mu.RLock()
	fresh := len(v.keys) > 0 && time.Since(v.fetchedAt) < v.cacheTTL
	v.mu.RUnlock()
	if fresh {
		return nil
	}
	return v.refreshKeys()
}

func (v *JWTValidator) refreshKeys() error {
	if v.jwksURL == "" {
		v.mu.RLock()
		ok := len(v.keys) > 0
		v.mu.RUnlock()
		if ok {
			return nil
		}
		return fmt.Errorf("jwks url not configured")
	}
	resp, err := v.client.Get(v.jwksURL)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks status %d", resp.StatusCode)
	}
	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}
	next := make(map[string]*rsa.PublicKey)
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.N == "" || k.E == "" {
			continue
		}
		pub, err := rsaPublicFromJWK(k.N, k.E)
		if err != nil {
			return err
		}
		kid := k.Kid
		if kid == "" {
			kid = "default"
		}
		next[kid] = pub
	}
	if len(next) == 0 {
		return fmt.Errorf("jwks has no rsa keys")
	}
	v.mu.Lock()
	v.keys = next
	v.fetchedAt = time.Now()
	v.mu.Unlock()
	return nil
}

func rsaPublicFromJWK(nStr, eStr string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("jwk n: %w", err)
	}
	eb, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("jwk e: %w", err)
	}
	n := new(big.Int).SetBytes(nb)
	e := new(big.Int).SetBytes(eb)
	if !e.IsInt64() {
		return nil, fmt.Errorf("jwk e too large")
	}
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
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
