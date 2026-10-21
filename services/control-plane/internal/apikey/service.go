package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/org"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/project"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	controlv1.UnimplementedApiKeyServiceServer
	repo        *Repository
	projectRepo *project.Repository
	orgRepo     *org.Repository
}

func NewService(repo *Repository, projectRepo *project.Repository, orgRepo *org.Repository) *Service {
	return &Service{repo: repo, projectRepo: projectRepo, orgRepo: orgRepo}
}

func (s *Service) CreateApiKey(ctx context.Context, req *controlv1.CreateApiKeyRequest) (*controlv1.CreateApiKeyResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	projectID, err := uuid.Parse(req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" || len(name) > 100 {
		return nil, status.Error(codes.InvalidArgument, "name must be 1-100 characters")
	}

	p, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "project not found")
		}
		return nil, status.Error(codes.Internal, "failed to get project")
	}

	member, err := s.orgRepo.GetMember(ctx, p.OrgID, userID)
	if err != nil {
		if errors.Is(err, org.ErrNotFound) {
			return nil, status.Error(codes.PermissionDenied, "not a member of this organization")
		}
		return nil, status.Error(codes.Internal, "failed to check membership")
	}
	if !member.Role.CanManageMembers() && member.Role != org.RoleMember {
		return nil, status.Error(codes.PermissionDenied, "insufficient role")
	}

	secret, prefix, hash, err := generateSecret()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate key")
	}

	key := &ApiKey{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   hash,
		CreatedBy: userID,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, key); err != nil {
		return nil, status.Error(codes.Internal, "failed to store api key")
	}

	return &controlv1.CreateApiKeyResponse{
		ApiKey: toProto(key),
		Secret: secret,
	}, nil
}

func (s *Service) ListApiKeys(ctx context.Context, req *controlv1.ListApiKeysRequest) (*controlv1.ListApiKeysResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	projectID, err := uuid.Parse(req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	p, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "project not found")
		}
		return nil, status.Error(codes.Internal, "failed to get project")
	}

	ok, err := s.orgRepo.HasMember(ctx, p.OrgID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check membership")
	}
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "not a member of this organization")
	}

	keys, err := s.repo.ListByProjectID(ctx, projectID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list api keys")
	}

	resp := &controlv1.ListApiKeysResponse{}
	for i := range keys {
		resp.ApiKeys = append(resp.ApiKeys, toProto(&keys[i]))
	}
	return resp, nil
}

func (s *Service) RevokeApiKey(ctx context.Context, req *controlv1.RevokeApiKeyRequest) (*emptypb.Empty, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	keyID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	key, err := s.repo.GetByID(ctx, keyID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, "api key not found")
		}
		return nil, status.Error(codes.Internal, "failed to get api key")
	}

	p, err := s.projectRepo.GetByID(ctx, key.ProjectID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get project")
	}

	member, err := s.orgRepo.GetMember(ctx, p.OrgID, userID)
	if err != nil {
		if errors.Is(err, org.ErrNotFound) {
			return nil, status.Error(codes.PermissionDenied, "not a member of this organization")
		}
		return nil, status.Error(codes.Internal, "failed to check membership")
	}

	if !member.Role.CanManageMembers() && key.CreatedBy != userID {
		return nil, status.Error(codes.PermissionDenied, "can only revoke own keys unless OWNER/ADMIN")
	}

	if key.RevokedAt != nil {
		return &emptypb.Empty{}, nil
	}

	if err := s.repo.Revoke(ctx, keyID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, "api key not found or already revoked")
		}
		return nil, status.Error(codes.Internal, "failed to revoke api key")
	}
	return &emptypb.Empty{}, nil
}

func generateSecret() (secret, prefix, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", err
	}
	secret = "pk_live_" + base64.RawURLEncoding.EncodeToString(raw)
	if len(secret) >= 16 {
		prefix = secret[:16]
	} else {
		prefix = secret
	}
	sum := sha256.Sum256([]byte(secret))
	hash = hex.EncodeToString(sum[:])
	return secret, prefix, hash, nil
}

func toProto(k *ApiKey) *controlv1.ApiKey {
	out := &controlv1.ApiKey{
		Id:        k.ID.String(),
		ProjectId: k.ProjectID.String(),
		Name:      k.Name,
		KeyPrefix: k.KeyPrefix,
		CreatedBy: k.CreatedBy.String(),
		CreatedAt: timestamppb.New(k.CreatedAt),
	}
	if k.RevokedAt != nil {
		out.RevokedAt = timestamppb.New(*k.RevokedAt)
	}
	if k.LastUsedAt != nil {
		out.LastUsedAt = timestamppb.New(*k.LastUsedAt)
	}
	return out
}

func (s *Service) VerifySecret(ctx context.Context, raw string) (*ApiKey, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "pk_live_") {
		return nil, ErrNotFound
	}
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	key, err := s.repo.FindActiveByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	_ = s.repo.TouchLastUsed(ctx, key.ID)
	now := time.Now().UTC()
	key.LastUsedAt = &now
	return key, nil
}
