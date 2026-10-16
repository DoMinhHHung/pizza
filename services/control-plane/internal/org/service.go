package org

import (
	"context"
	"errors"
	"regexp"
	"strings"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Service struct {
	controlv1.UnimplementedOrganizationServiceServer
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateOrganization(ctx context.Context, req *controlv1.CreateOrganizationRequest) (*controlv1.Organization, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.GetName())
	slug := strings.TrimSpace(strings.ToLower(req.GetSlug()))

	if name == "" || len(name) > 100 {
		return nil, status.Error(codes.InvalidArgument, "name must be 1-100 characters")
	}
	if slug == "" || len(slug) > 50 || !slugRegex.MatchString(slug) {
		return nil, status.Error(codes.InvalidArgument, "slug must be lowercase alphanumeric with hyphens")
	}

	exists, err := s.repo.ExistsBySlug(ctx, slug)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check slug")
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "slug already taken")
	}

	org, err := s.repo.Create(ctx, name, slug, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create organization")
	}

	return toProto(org, RoleOwner), nil
}

func (s *Service) ListMyOrganizations(ctx context.Context, _ *controlv1.ListMyOrganizationsRequest) (*controlv1.ListMyOrganizationsResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list organizations")
	}

	resp := &controlv1.ListMyOrganizationsResponse{}
	for _, item := range items {
		resp.Organizations = append(resp.Organizations, toProto(&item.Organization, item.Role))
	}
	return resp, nil
}

func toProto(org *Organization, role Role) *controlv1.Organization {
	return &controlv1.Organization{
		Id:        org.ID.String(),
		Name:      org.Name,
		Slug:      org.Slug,
		Role:      string(role),
		CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt),
	}
}

var _ = errors.New
