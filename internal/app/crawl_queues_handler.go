package app

import (
	"encoding/json"
	"net/http"
	"strings"

	crawljob "distributed-crawler/internal/api/crawl_job"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/auth"
	authdomain "distributed-crawler/internal/domain/auth/models"
	"distributed-crawler/internal/domain/crawl/models"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

type crawlQueuesResponse struct {
	Queues []string `json:"queues"`
}

// crawlQueuesHandlerFunc handles GET /api/v1/crawl-queues.
// Returns all configured crawl queue names.
func (a *APIApp) crawlQueuesHandlerFunc() runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		if _, ok := a.verifyAuth(w, r, authdomain.RoleRead); !ok {
			return
		}

		queues := a.serviceProvider.AvailableQueues()
		if queues == nil {
			queues = []string{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(crawlQueuesResponse{Queues: queues})
	}
}

// createJobHandlerFunc handles POST /api/v1/jobs with queue_weights support.
// The body format is: { "config": { ...CrawlJobConfig fields..., "queue_weights": [...] } }
func (a *APIApp) createJobHandlerFunc() runtime.HandlerFunc {
	pj := protojson.UnmarshalOptions{DiscardUnknown: true}

	return func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		claims, ok := a.verifyAuth(w, r, authdomain.RoleReadWrite)
		if !ok {
			return
		}

		// Decode body preserving the raw "config" portion for both proto unmarshalling
		// and queue_weights extraction.
		var envelope struct {
			Config json.RawMessage `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
		if envelope.Config == nil {
			writeJSONError(w, http.StatusBadRequest, "missing config field")
			return
		}

		// Unmarshal proto config (accepts both camelCase and snake_case keys).
		var protoConfig crawlergrpc.CrawlJobConfig
		if err := pj.Unmarshal(envelope.Config, &protoConfig); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid config: "+err.Error())
			return
		}

		// Extract queue_weights separately since proto doesn't carry them.
		var queueWeightsWrapper struct {
			QueueWeights []models.QueueWeight `json:"queue_weights"`
		}
		_ = json.Unmarshal(envelope.Config, &queueWeightsWrapper)

		config := crawljob.FromProtoCrawlJobConfig(&protoConfig)

		availableQueues := a.serviceProvider.AvailableQueues()
		// Queue routing only makes sense when more than one queue is configured.
		// In single-region mode ignore any submitted weights to avoid routing
		// tasks to queues that do not exist.
		if len(availableQueues) > 1 {
			config.QueueWeights = queueWeightsWrapper.QueueWeights
		}

		ctx := r.Context()
		id, err := a.serviceProvider.CrawlJobService(ctx).CreateCrawlJob(ctx, service.CreateCrawlJobCommand{
			Config:          config,
			UserID:          claims.UserID,
			AvailableQueues: availableQueues,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// getJobHandlerFunc handles GET /api/v1/jobs/{id} with queue_weights in the response.
func (a *APIApp) getJobHandlerFunc() runtime.HandlerFunc {
	pjMarshal := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}

	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		if _, ok := a.verifyAuth(w, r, authdomain.RoleRead); !ok {
			return
		}

		id := pathParams["id"]
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "missing job id")
			return
		}

		ctx := r.Context()
		job, err := a.serviceProvider.CrawlJobService(ctx).GetCrawlJob(ctx, service.GetCrawlJobQuery{ID: id})
		if err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}

		// Marshal to proto JSON, then inject queue_weights.
		protoJob := crawljob.ToProtoCrawlJob(job)
		protoResp := &crawlergrpc.GetJobResponse{Job: protoJob}

		protoBytes, err := pjMarshal.Marshal(protoResp)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to marshal response: "+err.Error())
			return
		}

		// Inject queue_weights into job.jobConfig.
		var resp map[string]interface{}
		if err := json.Unmarshal(protoBytes, &resp); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to process response")
			return
		}

		if job.JobConfig != nil && len(job.JobConfig.QueueWeights) > 0 {
			if jobMap, ok := resp["job"].(map[string]interface{}); ok {
				if configMap, ok := jobMap["job_config"].(map[string]interface{}); ok {
					configMap["queue_weights"] = job.JobConfig.QueueWeights
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// verifyAuth extracts and validates the JWT from the Authorization header.
// Returns the claims and true on success; writes an error and returns false on failure.
func (a *APIApp) verifyAuth(w http.ResponseWriter, r *http.Request, minRole authdomain.Role) (*auth.Claims, bool) {
	authHeader := r.Header.Get("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		writeJSONError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return nil, false
	}

	claims, err := a.serviceProvider.JWTService().VerifyToken(parts[1])
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
		return nil, false
	}

	role, parseErr := authdomain.ParseRole(claims.Role)
	if parseErr != nil || role.Level() < minRole.Level() {
		writeJSONError(w, http.StatusForbidden, "access denied")
		return nil, false
	}

	return claims, true
}
