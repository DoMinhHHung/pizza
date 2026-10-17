package project_test

import (
	"context"
	"testing"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/org"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/project"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/testutil"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ctxUser(id uuid.UUID) context.Context {
	return auth.WithUserID(context.Background(), id)
}

func uniqueSlug(prefix string) string {
	return prefix + "-" + uuid.New().String()[:8]
}

func setupOrg(t *testing.T, orgRepo *org.Repository, ownerID uuid.UUID) string {
	t.Helper()
	o, err := orgRepo.Create(context.Background(), "Test Org", uniqueSlug("org"), ownerID)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	return o.ID.String()
}

func TestProject_CRUDAndAuthz(t *testing.T) {
	testutil.MigrateUp(t)
	pool := testutil.NewPool(t)

	orgRepo := org.NewRepository(pool)
	projectRepo := project.NewRepository(pool)
	svc := project.NewService(projectRepo, orgRepo)

	ownerID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()

	ownerCtx := ctxUser(ownerID)
	memberCtx := ctxUser(memberID)
	outsiderCtx := ctxUser(outsiderID)

	orgID := setupOrg(t, orgRepo, ownerID)

	// add MEMBER
	if _, err := orgRepo.AddMember(context.Background(), uuid.MustParse(orgID), memberID, org.RoleMember); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	slug := uniqueSlug("api")
	created, err := svc.CreateProject(ownerCtx, &controlv1.CreateProjectRequest{
		OrgId: orgID,
		Name:  "API",
		Slug:  slug,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if created.GetName() != "API" || created.GetSlug() != slug {
		t.Fatalf("unexpected project: %+v", created)
	}
	projectID := created.GetId()

	// duplicate slug in same org
	_, err = svc.CreateProject(ownerCtx, &controlv1.CreateProjectRequest{
		OrgId: orgID,
		Name:  "API2",
		Slug:  slug,
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate slug: want AlreadyExists, got %v", err)
	}

	// outsider cannot create
	_, err = svc.CreateProject(outsiderCtx, &controlv1.CreateProjectRequest{
		OrgId: orgID,
		Name:  "Nope",
		Slug:  uniqueSlug("nope"),
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("outsider create: want PermissionDenied, got %v", err)
	}

	// list
	list, err := svc.ListProjects(memberCtx, &controlv1.ListProjectsRequest{OrgId: orgID})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(list.GetProjects()) < 1 {
		t.Fatal("expected at least 1 project")
	}

	// get
	got, err := svc.GetProject(memberCtx, &controlv1.GetProjectRequest{ProjectId: projectID})
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.GetId() != projectID {
		t.Fatalf("get id = %s", got.GetId())
	}

	// outsider get
	_, err = svc.GetProject(outsiderCtx, &controlv1.GetProjectRequest{ProjectId: projectID})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("outsider get: want PermissionDenied, got %v", err)
	}

	// MEMBER cannot update
	_, err = svc.UpdateProject(memberCtx, &controlv1.UpdateProjectRequest{
		ProjectId: projectID,
		Name:      "Hacked",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("member update: want PermissionDenied, got %v", err)
	}

	// OWNER update
	newSlug := uniqueSlug("api-v2")
	updated, err := svc.UpdateProject(ownerCtx, &controlv1.UpdateProjectRequest{
		ProjectId: projectID,
		Name:      "API v2",
		Slug:      newSlug,
	})
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	if updated.GetName() != "API v2" || updated.GetSlug() != newSlug {
		t.Fatalf("unexpected update: %+v", updated)
	}

	// MEMBER cannot delete
	_, err = svc.DeleteProject(memberCtx, &controlv1.DeleteProjectRequest{ProjectId: projectID})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("member delete: want PermissionDenied, got %v", err)
	}

	// OWNER delete
	_, err = svc.DeleteProject(ownerCtx, &controlv1.DeleteProjectRequest{ProjectId: projectID})
	if err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	// get after delete
	_, err = svc.GetProject(ownerCtx, &controlv1.GetProjectRequest{ProjectId: projectID})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("get deleted: want NotFound, got %v", err)
	}
}

func TestProject_Unauthenticated(t *testing.T) {
	testutil.MigrateUp(t)
	pool := testutil.NewPool(t)

	orgRepo := org.NewRepository(pool)
	svc := project.NewService(project.NewRepository(pool), orgRepo)

	_, err := svc.CreateProject(context.Background(), &controlv1.CreateProjectRequest{
		OrgId: uuid.New().String(),
		Name:  "X",
		Slug:  uniqueSlug("x"),
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestProject_InvalidIDs(t *testing.T) {
	testutil.MigrateUp(t)
	pool := testutil.NewPool(t)

	orgRepo := org.NewRepository(pool)
	svc := project.NewService(project.NewRepository(pool), orgRepo)
	ctx := ctxUser(uuid.New())

	_, err := svc.GetProject(ctx, &controlv1.GetProjectRequest{ProjectId: "not-uuid"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", err)
	}

	_, err = svc.CreateProject(ctx, &controlv1.CreateProjectRequest{
		OrgId: "not-uuid",
		Name:  "X",
		Slug:  uniqueSlug("x"),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", err)
	}
}
