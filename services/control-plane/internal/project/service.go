package project

import (
	"context"
	"regexp"
	"strings"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/org"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	exists, err := s.repo.ExistsByOrgAndSlug(ctx, orgID, slug)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check slug")
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "project slug already exists in this organization")
	}

	p, err := s.repo.Create(ctx, orgID, name, slug)
	if err != nil {
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
