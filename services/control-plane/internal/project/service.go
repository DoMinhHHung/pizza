package project

import (
	"context"
	"errors"
	"regexp"
	"strings"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/org"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Service struct {
	controlv1.UnimplementedProjectServiceServer
	repo    *Repository
	orgRepo *org.Repository
}

func NewService(repo *Repository, orgRepo *org.Repository) *Service {
	return &Service{repo: repo, orgRepo: orgRepo}
}

func (s *Service) CreateProject(ctx context.Context, req *controlv1.CreateProjectRequest) (*controlv1.Project, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}

	name := strings.TrimSpace(req.GetName())
	slug := strings.TrimSpace(strings.ToLower(req.GetSlug()))

	if name == "" || len(name) > 100 {
		return nil, status.Error(codes.InvalidArgument, "name must be 1-100 characters")
	}
	if slug == "" || len(slug) > 50 || !slugRegex.MatchString(slug) {
		return nil, status.Error(codes.InvalidArgument, "slug must be lowercase alphanumeric with hyphens")
	}

	ok, err := s.orgRepo.HasMember(ctx, orgID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check membership")
	}
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "not a member of this organization")
	}

	p, err := s.repo.Create(ctx, orgID, name, slug)
	if err != nil {
		if errors.Is(err, ErrSlugTaken) {
			return nil, status.Error(codes.AlreadyExists, "project slug already exists in this organization")
		}
		return nil, status.Error(codes.Internal, "failed to create project")
	}
	return toProto(p), nil
}

func (s *Service) ListProjects(ctx context.Context, req *controlv1.ListProjectsRequest) (*controlv1.ListProjectsResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	orgID, err := uuid.Parse(req.GetOrgId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}

	ok, err := s.orgRepo.HasMember(ctx, orgID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check membership")
	}
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "not a member of this organization")
	}

	items, err := s.repo.ListByOrgID(ctx, orgID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list projects")
	}

	resp := &controlv1.ListProjectsResponse{}
	for i := range items {
		resp.Projects = append(resp.Projects, toProto(&items[i]))
	}
	return resp, nil
}

func toProto(p *Project) *controlv1.Project {
	return &controlv1.Project{
		Id:        p.ID.String(),
		OrgId:     p.OrgID.String(),
		Name:      p.Name,
		Slug:      p.Slug,
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func (s *Service) GetProject(ctx context.Context, req *controlv1.GetProjectRequest) (*controlv1.Project, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	projectID, err := uuid.Parse(req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	p, err := s.repo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
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

	return toProto(p), nil
}

func (s *Service) UpdateProject(ctx context.Context, req *controlv1.UpdateProjectRequest) (*controlv1.Project, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	projectID, err := uuid.Parse(req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	p, err := s.repo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
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
	if member.Role != org.RoleOwner && member.Role != org.RoleAdmin {
		return nil, status.Error(codes.PermissionDenied, "insufficient role to update project")
	}

	name := strings.TrimSpace(req.GetName())
	slug := strings.TrimSpace(strings.ToLower(req.GetSlug()))

	if name != "" && len(name) > 100 {
		return nil, status.Error(codes.InvalidArgument, "name must be 1-100 characters")
	}
	if slug != "" && (len(slug) > 50 || !slugRegex.MatchString(slug)) {
		return nil, status.Error(codes.InvalidArgument, "slug must be lowercase alphanumeric with hyphens")
	}

	updated, err := s.repo.Update(ctx, projectID, name, slug)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return nil, status.Error(codes.NotFound, "project not found")
		case errors.Is(err, ErrSlugTaken):
			return nil, status.Error(codes.AlreadyExists, "project slug already exists in this organization")
		default:
			return nil, status.Error(codes.Internal, "failed to update project")
		}
	}
	return toProto(updated), nil
}

func (s *Service) DeleteProject(ctx context.Context, req *controlv1.DeleteProjectRequest) (*emptypb.Empty, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	projectID, err := uuid.Parse(req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	p, err := s.repo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
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
	if member.Role != org.RoleOwner && member.Role != org.RoleAdmin {
		return nil, status.Error(codes.PermissionDenied, "insufficient role to delete project")
	}

	if err := s.repo.Delete(ctx, projectID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, "project not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete project")
	}

	return &emptypb.Empty{}, nil
}
