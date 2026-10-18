package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxRequestBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"message": "request body too large"})
			return false
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid json"})
		return false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid json"})
		return false
	}
	return true
}

type Server struct {
	validator *auth.JWTValidator
	orgs      controlv1.OrganizationServiceServer
	projects  controlv1.ProjectServiceServer
	apiKeys   controlv1.ApiKeyServiceServer
	mux       *http.ServeMux
	marshal   protojson.MarshalOptions
	ping      func(ctx context.Context) error
}

func New(
	validator *auth.JWTValidator,
	orgs controlv1.OrganizationServiceServer,
	projects controlv1.ProjectServiceServer,
	apiKeys controlv1.ApiKeyServiceServer,
	ping func(ctx context.Context) error,
) *Server {
	s := &Server{
		validator: validator,
		orgs:      orgs,
		projects:  projects,
		apiKeys:   apiKeys,
		ping:      ping,
		mux:       http.NewServeMux(),
		marshal:   protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true},
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.cors(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if s.ping != nil {
			if err := s.ping(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{
					"status": "DOWN",
					"error":  "database unavailable",
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "UP"})
	})

	authz := s.validator.Middleware

	s.mux.Handle("GET /api/v1/orgs", authz(http.HandlerFunc(s.listOrgs)))
	s.mux.Handle("POST /api/v1/orgs", authz(http.HandlerFunc(s.createOrg)))
	s.mux.Handle("GET /api/v1/orgs/{orgId}/members", authz(http.HandlerFunc(s.listMembers)))
	s.mux.Handle("POST /api/v1/orgs/{orgId}/members", authz(http.HandlerFunc(s.addMember)))
	s.mux.Handle("PATCH /api/v1/orgs/{orgId}/members/{userId}", authz(http.HandlerFunc(s.updateMemberRole)))
	s.mux.Handle("DELETE /api/v1/orgs/{orgId}/members/{userId}", authz(http.HandlerFunc(s.removeMember)))
	s.mux.Handle("GET /api/v1/orgs/{orgId}/projects", authz(http.HandlerFunc(s.listProjects)))
	s.mux.Handle("POST /api/v1/orgs/{orgId}/projects", authz(http.HandlerFunc(s.createProject)))
	s.mux.Handle("GET /api/v1/projects/{projectId}/api-keys", authz(http.HandlerFunc(s.listApiKeys)))
	s.mux.Handle("POST /api/v1/projects/{projectId}/api-keys", authz(http.HandlerFunc(s.createApiKey)))
	s.mux.Handle("DELETE /api/v1/api-keys/{id}", authz(http.HandlerFunc(s.revokeApiKey)))
	s.mux.Handle("GET /api/v1/projects/{projectId}", authz(http.HandlerFunc(s.getProject)))
	s.mux.Handle("PATCH /api/v1/projects/{projectId}", authz(http.HandlerFunc(s.updateProject)))
	s.mux.Handle("DELETE /api/v1/projects/{projectId}", authz(http.HandlerFunc(s.deleteProject)))
}

func (s *Server) listOrgs(w http.ResponseWriter, r *http.Request) {
	resp, err := s.orgs.ListMyOrganizations(r.Context(), &controlv1.ListMyOrganizationsRequest{})
	s.writeProto(w, resp, err)
}

func (s *Server) createOrg(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := decodeJSON; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid json"})
		return
	}
	resp, err := s.orgs.CreateOrganization(r.Context(), &controlv1.CreateOrganizationRequest{
		Name: body.Name,
		Slug: body.Slug,
	})
	s.writeProtoStatus(w, resp, err, http.StatusCreated)
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	resp, err := s.orgs.ListMembers(r.Context(), &controlv1.ListMembersRequest{
		OrgId: r.PathValue("orgId"),
	})
	s.writeProto(w, resp, err)
}

func (s *Server) addMember(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	if err := decodeJSON; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid json"})
		return
	}
	resp, err := s.orgs.AddMember(r.Context(), &controlv1.AddMemberRequest{
		OrgId:  r.PathValue("orgId"),
		UserId: body.UserID,
		Role:   body.Role,
	})
	s.writeProtoStatus(w, resp, err, http.StatusCreated)
}

func (s *Server) updateMemberRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Role string `json:"role"`
	}
	if err := decodeJSON; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid json"})
		return
	}
	resp, err := s.orgs.UpdateMemberRole(r.Context(), &controlv1.UpdateMemberRoleRequest{
		OrgId:  r.PathValue("orgId"),
		UserId: r.PathValue("userId"),
		Role:   body.Role,
	})
	s.writeProto(w, resp, err)
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	_, err := s.orgs.RemoveMember(r.Context(), &controlv1.RemoveMemberRequest{
		OrgId:  r.PathValue("orgId"),
		UserId: r.PathValue("userId"),
	})
	if err != nil {
		s.writeGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	resp, err := s.projects.ListProjects(r.Context(), &controlv1.ListProjectsRequest{
		OrgId: r.PathValue("orgId"),
	})
	s.writeProto(w, resp, err)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := decodeJSON; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid json"})
		return
	}
	resp, err := s.projects.CreateProject(r.Context(), &controlv1.CreateProjectRequest{
		OrgId: r.PathValue("orgId"),
		Name:  body.Name,
		Slug:  body.Slug,
	})
	s.writeProtoStatus(w, resp, err, http.StatusCreated)
}

func (s *Server) listApiKeys(w http.ResponseWriter, r *http.Request) {
	resp, err := s.apiKeys.ListApiKeys(r.Context(), &controlv1.ListApiKeysRequest{
		ProjectId: r.PathValue("projectId"),
	})
	s.writeProto(w, resp, err)
}

func (s *Server) createApiKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid json"})
		return
	}
	resp, err := s.apiKeys.CreateApiKey(r.Context(), &controlv1.CreateApiKeyRequest{
		ProjectId: r.PathValue("projectId"),
		Name:      body.Name,
	})
	s.writeProtoStatus(w, resp, err, http.StatusCreated)
}

func (s *Server) revokeApiKey(w http.ResponseWriter, r *http.Request) {
	_, err := s.apiKeys.RevokeApiKey(r.Context(), &controlv1.RevokeApiKeyRequest{
		Id: r.PathValue("id"),
	})
	if err != nil {
		s.writeGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) writeProto(w http.ResponseWriter, msg proto.Message, err error) {
	s.writeProtoStatus(w, msg, err, http.StatusOK)
}

func (s *Server) writeProtoStatus(w http.ResponseWriter, msg proto.Message, err error, okStatus int) {
	if err != nil {
		s.writeGRPCError(w, err)
		return
	}
	b, mErr := s.marshal.Marshal(msg)
	if mErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "encode failed"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(okStatus)
	_, _ = w.Write(b)
}

func (s *Server) writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	code := http.StatusInternalServerError
	switch st.Code() {
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.Unauthenticated:
		code = http.StatusUnauthorized
	case codes.PermissionDenied:
		code = http.StatusForbidden
	case codes.NotFound:
		code = http.StatusNotFound
	case codes.AlreadyExists:
		code = http.StatusConflict
	case codes.FailedPrecondition:
		code = http.StatusPreconditionFailed
	}
	writeJSON(w, code, map[string]string{"message": st.Message()})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) cors(next http.Handler) http.Handler {
	origins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
	if len(origins) == 1 && strings.TrimSpace(origins[0]) == "" {
		origins = []string{"http://localhost:3000", "http://localhost:5173"}
	}
	allowed := map[string]bool{}
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	resp, err := s.projects.GetProject(r.Context(), &controlv1.GetProjectRequest{
		ProjectId: r.PathValue("projectId"),
	})
	s.writeProto(w, resp, err)
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := decodeJSON; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid json"})
		return
	}
	resp, err := s.projects.UpdateProject(r.Context(), &controlv1.UpdateProjectRequest{
		ProjectId: r.PathValue("projectId"),
		Name:      body.Name,
		Slug:      body.Slug,
	})
	s.writeProto(w, resp, err)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	_, err := s.projects.DeleteProject(r.Context(), &controlv1.DeleteProjectRequest{
		ProjectId: r.PathValue("projectId"),
	})
	if err != nil {
		s.writeGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}
