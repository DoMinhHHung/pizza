package org

import (
	"context"
	"errors"
	"regexp"
	"strings"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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

	org, err := s.repo.Create(ctx, name, slug, userID)
	if err != nil {
		if errors.Is(err, ErrSlugTaken) {
			return nil, status.Error(codes.AlreadyExists, "slug already taken")
		}
		return nil, status.Error(codes.Internal, "failed to create organization")
	}
	return toOrgProto(org, RoleOwner), nil
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
		resp.Organizations = append(resp.Organizations, toOrgProto(&item.Organization, item.Role))
	}
	return resp, nil
}

func (s *Service) ListMembers(ctx context.Context, req *controlv1.ListMembersRequest) (*controlv1.ListMembersResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}
	if _, err := s.requireMember(ctx, orgID, userID); err != nil {
		return nil, err
	}
	members, err := s.repo.ListMembers(ctx, orgID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list members")
	}
	resp := &controlv1.ListMembersResponse{}
	for i := range members {
		resp.Members = append(resp.Members, toMemberProto(&members[i]))
	}
	return resp, nil
}

func (s *Service) AddMember(ctx context.Context, req *controlv1.AddMemberRequest) (*controlv1.Member, error) {
	actorID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}
	targetID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	role, ok := ParseRole(strings.TrimSpace(strings.ToUpper(req.GetRole())))
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "role must be OWNER, ADMIN, or MEMBER")
	}
	actor, err := s.requireMember(ctx, orgID, actorID)
	if err != nil {
		return nil, err
	}
	if !actor.Role.CanManageMembers() {
		return nil, status.Error(codes.PermissionDenied, "insufficient role to add members")
	}
	if role == RoleOwner && actor.Role != RoleOwner {
		return nil, status.Error(codes.PermissionDenied, "only OWNER can add another OWNER")
	}
	if role == RoleAdmin && actor.Role != RoleOwner {
		return nil, status.Error(codes.PermissionDenied, "only OWNER can add ADMIN")
	}

	m, err := s.repo.AddMember(ctx, orgID, targetID, role)
	if err != nil {
		if errors.Is(err, ErrAlreadyMember) {
			return nil, status.Error(codes.AlreadyExists, "user is already a member")
		}
		return nil, status.Error(codes.Internal, "failed to add member")
	}
	return toMemberProto(m), nil
}

func (s *Service) UpdateMemberRole(ctx context.Context, req *controlv1.UpdateMemberRoleRequest) (*controlv1.Member, error) {
	actorID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}
	targetID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	newRole, ok := ParseRole(strings.TrimSpace(strings.ToUpper(req.GetRole())))
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "role must be OWNER, ADMIN, or MEMBER")
	}
	actor, err := s.requireMember(ctx, orgID, actorID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleOwner {
		return nil, status.Error(codes.PermissionDenied, "only OWNER can change roles")
	}
	if actor.Role != RoleOwner {
		return nil, status.Error(codes.PermissionDenied, "only OWNER can change roles")
	}

	updated, err := s.repo.UpdateMemberRoleGuarded(ctx, orgID, targetID, newRole)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return nil, status.Error(codes.NotFound, "member not found")
		case errors.Is(err, ErrLastOwner):
			return nil, status.Error(codes.FailedPrecondition, "cannot demote the last OWNER")
		default:
			return nil, status.Error(codes.Internal, "failed to update role")
		}
	}
	return toMemberProto(updated), nil
}

func (s *Service) RemoveMember(ctx context.Context, req *controlv1.RemoveMemberRequest) (*emptypb.Empty, error) {
	actorID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}
	targetID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	actor, err := s.requireMember(ctx, orgID, actorID)
	if err != nil {
		return nil, err
	}
	if !actor.Role.CanManageMembers() {
		return nil, status.Error(codes.PermissionDenied, "insufficient role to remove members")
	}
	target, err := s.repo.GetMember(ctx, orgID, targetID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, "member not found")
		}
		return nil, status.Error(codes.Internal, "failed to get member")
	}

	if target.Role == RoleAdmin && actor.Role != RoleOwner {
		return nil, status.Error(codes.PermissionDenied, "only OWNER can remove an ADMIN")
	}

	if err := s.repo.RemoveMemberGuarded(ctx, orgID, targetID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return nil, status.Error(codes.NotFound, "member not found")
		case errors.Is(err, ErrLastOwner):
			return nil, status.Error(codes.FailedPrecondition, "cannot remove the last OWNER")
		default:
			return nil, status.Error(codes.Internal, "failed to remove member")
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *Service) requireMember(ctx context.Context, orgID, userID uuid.UUID) (*Member, error) {
	m, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.PermissionDenied, "not a member of this organization")
		}
		return nil, status.Error(codes.Internal, "failed to check membership")
	}
	return m, nil
}

func toOrgProto(org *Organization, role Role) *controlv1.Organization {
	return &controlv1.Organization{
		Id:        org.ID.String(),
		Name:      org.Name,
		Slug:      org.Slug,
		Role:      string(role),
		CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt),
	}
}

func toMemberProto(m *Member) *controlv1.Member {
	return &controlv1.Member{
		OrgId:     m.OrgID.String(),
		UserId:    m.UserID.String(),
		Role:      string(m.Role),
		CreatedAt: timestamppb.New(m.CreatedAt),
	}
}

func (s *Service) GetOrganization(ctx context.Context, req *controlv1.GetOrganizationRequest) (*controlv1.Organization, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}
	member, err := s.requireMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	o, err := s.repo.GetByID(ctx, orgID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}
		return nil, status.Error(codes.Internal, "failed to get organization")
	}
	return toOrgProto(o, member.Role), nil
}

func (s *Service) UpdateOrganization(ctx context.Context, req *controlv1.UpdateOrganizationRequest) (*controlv1.Organization, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}
	member, err := s.requireMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if member.Role != RoleOwner && member.Role != RoleAdmin {
		return nil, status.Error(codes.PermissionDenied, "insufficient role to update organization")
	}

	name := strings.TrimSpace(req.GetName())
	slug := strings.TrimSpace(strings.ToLower(req.GetSlug()))
	if slug != "" && (len(slug) > 50 || !slugRegex.MatchString(slug)) {
		return nil, status.Error(codes.InvalidArgument, "invalid slug")
	}
	if name != "" && len(name) > 100 {
		return nil, status.Error(codes.InvalidArgument, "name too long")
	}

	o, err := s.repo.Update(ctx, orgID, name, slug)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return nil, status.Error(codes.NotFound, "organization not found")
		case errors.Is(err, ErrSlugTaken):
			return nil, status.Error(codes.AlreadyExists, "slug already taken")
		default:
			return nil, status.Error(codes.Internal, "failed to update organization")
		}
	}
	return toOrgProto(o, member.Role), nil
}

func (s *Service) DeleteOrganization(ctx context.Context, req *controlv1.DeleteOrganizationRequest) (*emptypb.Empty, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}
	member, err := s.requireMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if member.Role != RoleOwner {
		return nil, status.Error(codes.PermissionDenied, "only OWNER can delete organization")
	}
	if err := s.repo.Delete(ctx, orgID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete organization")
	}
	return &emptypb.Empty{}, nil
}
