package org_test

import (
	"context"
	"testing"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/org"
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

func TestOrganization_CreateListAndMembers(t *testing.T) {
	testutil.MigrateUp(t)
	pool := testutil.NewPool(t)

	repo := org.NewRepository(pool)
	svc := org.NewService(repo)

	ownerID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()
	slug := uniqueSlug("acme")

	ownerCtx := ctxUser(ownerID)
	memberCtx := ctxUser(memberID)
	outsiderCtx := ctxUser(outsiderID)

	created, err := svc.CreateOrganization(ownerCtx, &controlv1.CreateOrganizationRequest{
		Name: "Acme Inc",
		Slug: slug,
	})
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}
	if created.GetName() != "Acme Inc" {
		t.Fatalf("name = %q", created.GetName())
	}
	if created.GetSlug() != slug {
		t.Fatalf("slug = %q", created.GetSlug())
	}
	if created.GetRole() != "OWNER" {
		t.Fatalf("role = %q", created.GetRole())
	}
	if created.GetId() == "" {
		t.Fatal("id is empty")
	}

	orgID := created.GetId()

	// duplicate slug
	_, err = svc.CreateOrganization(ownerCtx, &controlv1.CreateOrganizationRequest{
		Name: "Other",
		Slug: slug,
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate slug: want AlreadyExists, got %v", err)
	}

	// invalid slug
	_, err = svc.CreateOrganization(ownerCtx, &controlv1.CreateOrganizationRequest{
		Name: "Bad",
		Slug: "INVALID_SLUG",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid slug: want InvalidArgument, got %v", err)
	}

	list, err := svc.ListMyOrganizations(ownerCtx, &controlv1.ListMyOrganizationsRequest{})
	if err != nil {
		t.Fatalf("ListMyOrganizations: %v", err)
	}
	found := false
	for _, o := range list.GetOrganizations() {
		if o.GetId() == orgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("created org not in list")
	}

	// outsider cannot list this org as member list
	_, err = svc.ListMembers(outsiderCtx, &controlv1.ListMembersRequest{OrgId: orgID})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("outsider ListMembers: want PermissionDenied, got %v", err)
	}

	// add member
	m, err := svc.AddMember(ownerCtx, &controlv1.AddMemberRequest{
		OrgId:  orgID,
		UserId: memberID.String(),
		Role:   "MEMBER",
	})
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if m.GetUserId() != memberID.String() || m.GetRole() != "MEMBER" {
		t.Fatalf("unexpected member: %+v", m)
	}

	// duplicate member
	_, err = svc.AddMember(ownerCtx, &controlv1.AddMemberRequest{
		OrgId:  orgID,
		UserId: memberID.String(),
		Role:   "MEMBER",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate member: want AlreadyExists, got %v", err)
	}

	// member can list members
	members, err := svc.ListMembers(memberCtx, &controlv1.ListMembersRequest{OrgId: orgID})
	if err != nil {
		t.Fatalf("ListMembers as member: %v", err)
	}
	if len(members.GetMembers()) < 2 {
		t.Fatalf("want >= 2 members, got %d", len(members.GetMembers()))
	}

	// member cannot add another member
	_, err = svc.AddMember(memberCtx, &controlv1.AddMemberRequest{
		OrgId:  orgID,
		UserId: uuid.New().String(),
		Role:   "MEMBER",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("member AddMember: want PermissionDenied, got %v", err)
	}

	// promote to ADMIN
	updated, err := svc.UpdateMemberRole(ownerCtx, &controlv1.UpdateMemberRoleRequest{
		OrgId:  orgID,
		UserId: memberID.String(),
		Role:   "ADMIN",
	})
	if err != nil {
		t.Fatalf("UpdateMemberRole: %v", err)
	}
	if updated.GetRole() != "ADMIN" {
		t.Fatalf("role after update = %q", updated.GetRole())
	}

	// remove member
	_, err = svc.RemoveMember(ownerCtx, &controlv1.RemoveMemberRequest{
		OrgId:  orgID,
		UserId: memberID.String(),
	})
	if err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	// cannot remove last owner
	_, err = svc.RemoveMember(ownerCtx, &controlv1.RemoveMemberRequest{
		OrgId:  orgID,
		UserId: ownerID.String(),
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("remove last owner: want FailedPrecondition, got %v", err)
	}
}

func TestOrganization_Unauthenticated(t *testing.T) {
	testutil.MigrateUp(t)
	pool := testutil.NewPool(t)
	svc := org.NewService(org.NewRepository(pool))

	_, err := svc.CreateOrganization(context.Background(), &controlv1.CreateOrganizationRequest{
		Name: "X",
		Slug: uniqueSlug("x"),
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}
