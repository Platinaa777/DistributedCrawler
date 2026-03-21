package app

import (
	"encoding/json"
	"net/http"
	"strings"

	"distributed-crawler/internal/application/service"
	authdomain "distributed-crawler/internal/domain/auth/models"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// deleteJobHandlerFunc returns a grpc-gateway HandlerFunc for DELETE /api/v1/jobs/{id}.
// Auth is validated manually because this route is registered outside the gRPC interceptor chain.
func (a *APIApp) deleteJobHandlerFunc() runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		// Extract JWT from Authorization header
		authHeader := r.Header.Get("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeJSONError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}

		claims, err := a.serviceProvider.JWTService().VerifyToken(parts[1])
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		role, err := authdomain.ParseRole(claims.Role)
		if err != nil || role.Level() < authdomain.RoleReadWrite.Level() {
			writeJSONError(w, http.StatusForbidden, "access denied")
			return
		}

		id := pathParams["id"]
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "missing job id")
			return
		}

		ctx := r.Context()
		if err := a.serviceProvider.CrawlJobService(ctx).DeleteCrawlJob(ctx, service.DeleteCrawlJobCommand{
			JobID: id,
		}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
