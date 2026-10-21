package apikey_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/apikey"
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

func setupProject(
	t *testing.T,
	orgRepo *org.Repository,
	projectRepo *project.Repository,
	ownerID uuid.UUID,
) (orgID, projectID string) {
	t.Helper()
	ctx := context.Background()
	o, err := orgRepo.Create(ctx, "API Key Org", uniqueSlug("ak-org"), ownerID)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	p, err := projectRepo.Create(ctx, o.ID, "API Key Project", uniqueSlug("ak-proj"))
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return o.ID.String(), p.ID.String()
}

func TestApiKey_CreateListRevokeAndVerify(t *testing.T) {
	testutil.MigrateUp(t)
	pool := testutil.NewPool(t)

	orgRepo := org.NewRepository(pool)
	projectRepo := project.NewRepository(pool)
	apiKeyRepo := apikey.NewRepository(pool)
	svc := apikey.NewService(apiKeyRepo, projectRepo, orgRepo)

	ownerID := uuid.New()
	memberID := uuid.New()
	otherMemberID := uuid.New()
	outsiderID := uuid.New()

	ownerCtx := ctxUser(ownerID)
	memberCtx := ctxUser(memberID)
	otherMemberCtx := ctxUser(otherMemberID)
	outsiderCtx := ctxUser(outsiderID)

	orgID, projectID := setupProject(t, orgRepo, projectRepo, ownerID)

	if _, err := orgRepo.AddMember(context.Background(), uuid.MustParse(orgID), memberID, org.RoleMember); err != nil {
		t.Fatalf("AddMember member: %v", err)
	}
	if _, err := orgRepo.AddMember(context.Background(), uuid.MustParse(orgID), otherMemberID, org.RoleMember); err != nil {
		t.Fatalf("AddMember other: %v", err)
	}

	// OWNER create
	created, err := svc.CreateApiKey(ownerCtx, &controlv1.CreateApiKeyRequest{
		ProjectId: projectID,
		Name:      "ci-owner",
	})
	if err != nil {
		t.Fatalf("CreateApiKey owner: %v", err)
	}
	secret := created.GetSecret()
	if !strings.HasPrefix(secret, "pk_live_") {
		t.Fatalf("secret format: %q", secret)
	}
	if created.GetApiKey().GetKeyPrefix() == "" || created.GetApiKey().GetName() != "ci-owner" {
		t.Fatalf("unexpected api key meta: %+v", created.GetApiKey())
	}
	keyID := created.GetApiKey().GetId()

	// MEMBER create
	created2, err := svc.CreateApiKey(memberCtx, &controlv1.CreateApiKeyRequest{
		ProjectId: projectID,
		Name:      "ci-member",
	})
	if err != nil {
		t.Fatalf("CreateApiKey member: %v", err)
	}
	memberKeyID := created2.GetApiKey().GetId()
	memberSecret := created2.GetSecret()

	// outsider cannot create
	_, err = svc.CreateApiKey(outsiderCtx, &controlv1.CreateApiKeyRequest{
		ProjectId: projectID,
		Name:      "nope",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("outsider create: want PermissionDenied, got %v", err)
	}

	// invalid name
	_, err = svc.CreateApiKey(ownerCtx, &controlv1.CreateApiKeyRequest{
		ProjectId: projectID,
		Name:      "   ",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty name: want InvalidArgument, got %v", err)
	}

	// list — member sees both; secret not in list
	list, err := svc.ListApiKeys(memberCtx, &controlv1.ListApiKeysRequest{ProjectId: projectID})
	if err != nil {
		t.Fatalf("ListApiKeys: %v", err)
	}
	if len(list.GetApiKeys()) < 2 {
		t.Fatalf("want >= 2 keys, got %d", len(list.GetApiKeys()))
	}
	for _, k := range list.GetApiKeys() {
		if k.GetKeyPrefix() == "" {
			t.Fatal("list item missing prefix")
		}
	}

	// outsider list denied
	_, err = svc.ListApiKeys(outsiderCtx, &controlv1.ListApiKeysRequest{ProjectId: projectID})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("outsider list: want PermissionDenied, got %v", err)
	}

	// VerifySecret active
	got, err := svc.VerifySecret(context.Background(), secret)
	if err != nil {
		t.Fatalf("VerifySecret: %v", err)
	}
	if got.ID.String() != keyID {
		t.Fatalf("verify id: want %s got %s", keyID, got.ID)
	}
	if got.LastUsedAt == nil {
		t.Fatal("expected last_used_at after VerifySecret")
	}

	// wrong secret
	_, err = svc.VerifySecret(context.Background(), "pk_live_notarealsecretvalue000")
	if err != apikey.ErrNotFound {
		t.Fatalf("bad secret: want ErrNotFound, got %v", err)
	}

	// hash in DB matches secret
	sum := sha256.Sum256([]byte(secret))
	wantHash := hex.EncodeToString(sum[:])
	row, err := apiKeyRepo.FindActiveByHash(context.Background(), wantHash)
	if err != nil {
		t.Fatalf("FindActiveByHash: %v", err)
	}
	if row.ID.String() != keyID {
		t.Fatalf("hash lookup id mismatch")
	}

	// other MEMBER cannot revoke owner's key
	_, err = svc.RevokeApiKey(otherMemberCtx, &controlv1.RevokeApiKeyRequest{Id: keyID})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("other member revoke owner key: want PermissionDenied, got %v", err)
	}

	// MEMBER can revoke own key
	_, err = svc.RevokeApiKey(memberCtx, &controlv1.RevokeApiKeyRequest{Id: memberKeyID})
	if err != nil {
		t.Fatalf("member revoke own: %v", err)
	}

	// revoked secret no longer verifies
	_, err = svc.VerifySecret(context.Background(), memberSecret)
	if err != apikey.ErrNotFound {
		t.Fatalf("revoked verify: want ErrNotFound, got %v", err)
	}

	// OWNER can revoke any
	_, err = svc.RevokeApiKey(ownerCtx, &controlv1.RevokeApiKeyRequest{Id: keyID})
	if err != nil {
		t.Fatalf("owner revoke: %v", err)
	}

	// idempotent revoke
	_, err = svc.RevokeApiKey(ownerCtx, &controlv1.RevokeApiKeyRequest{Id: keyID})
	if err != nil {
		t.Fatalf("revoke already revoked: %v", err)
	}

	_, err = svc.VerifySecret(context.Background(), secret)
	if err != apikey.ErrNotFound {
		t.Fatalf("owner key after revoke: want ErrNotFound, got %v", err)
	}
}

func TestApiKey_Unauthenticated(t *testing.T) {
	testutil.MigrateUp(t)
	pool := testutil.NewPool(t)

	svc := apikey.NewService(
		apikey.NewRepository(pool),
		project.NewRepository(pool),
		org.NewRepository(pool),
	)

	_, err := svc.CreateApiKey(context.Background(), &controlv1.CreateApiKeyRequest{
		ProjectId: uuid.New().String(),
		Name:      "x",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestApiKey_InvalidIDs(t *testing.T) {
	testutil.MigrateUp(t)
	pool := testutil.NewPool(t)

	svc := apikey.NewService(
		apikey.NewRepository(pool),
		project.NewRepository(pool),
		org.NewRepository(pool),
	)
	ctx := ctxUser(uuid.New())

	_, err := svc.CreateApiKey(ctx, &controlv1.CreateApiKeyRequest{
		ProjectId: "not-uuid",
		Name:      "x",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("create: want InvalidArgument, got %v", err)
	}

	_, err = svc.RevokeApiKey(ctx, &controlv1.RevokeApiKeyRequest{Id: "not-uuid"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("revoke: want InvalidArgument, got %v", err)
	}
}