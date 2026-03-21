package converters

import (
	"database/sql"
	"encoding/json"
	"fmt"

	authvalueobjects "distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

func SaveCrawlJobConfigToSnapshot(config models.CrawlJobConfig) (*snapshots.CrawlJobConfigSnapshot, error) {
	snapshot := &snapshots.CrawlJobConfigSnapshot{
		ID:   config.ID.String(),
		Name: config.Name,
	}
	if !config.UserID.IsEmpty() {
		snapshot.UserID = sql.NullString{
			String: config.UserID.String(),
			Valid:  true,
		}
	}

	// Convert ExtractionSpec to JSONB
	extractionSpecJSON, err := json.Marshal(config.ExtractionSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ExtractionSpec: %w", err)
	}
	var extractionSpecMap map[string]interface{}
	if err := json.Unmarshal(extractionSpecJSON, &extractionSpecMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ExtractionSpec to map: %w", err)
	}
	snapshot.ExtractionSpec = extractionSpecMap

	// Convert Scopes to JSONB
	scopesJSON, err := json.Marshal(config.Scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Scopes: %w", err)
	}
	var scopesMap map[string]interface{}
	if err := json.Unmarshal(scopesJSON, &scopesMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Scopes to map: %w", err)
	}
	snapshot.Scopes = scopesMap

	// Convert Seeds to JSONBArray
	seedsJSON, err := json.Marshal(config.Seeds)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Seeds: %w", err)
	}
	var seedsArray []interface{}
	if err := json.Unmarshal(seedsJSON, &seedsArray); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Seeds to array: %w", err)
	}
	snapshot.Seeds = seedsArray

	// Convert RateLimit to JSONB
	rateLimitJSON, err := json.Marshal(config.RateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RateLimit: %w", err)
	}
	var rateLimitMap map[string]interface{}
	if err := json.Unmarshal(rateLimitJSON, &rateLimitMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RateLimit to map: %w", err)
	}
	snapshot.RateLimit = rateLimitMap

	// Convert Retries to JSONB
	retriesJSON, err := json.Marshal(config.Retries)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Retries: %w", err)
	}
	var retriesMap map[string]interface{}
	if err := json.Unmarshal(retriesJSON, &retriesMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Retries to map: %w", err)
	}
	snapshot.Retries = retriesMap

	// Convert Auth to JSONB
	authJSON, err := json.Marshal(config.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Auth: %w", err)
	}
	var authMap map[string]interface{}
	if err := json.Unmarshal(authJSON, &authMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Auth to map: %w", err)
	}
	snapshot.Auth = authMap

	// Convert Schedule to JSONB
	scheduleJSON, err := json.Marshal(config.Schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Schedule: %w", err)
	}
	var scheduleMap map[string]interface{}
	if err := json.Unmarshal(scheduleJSON, &scheduleMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Schedule to map: %w", err)
	}
	snapshot.Schedule = scheduleMap

	// Set JobType (default to ONCE if empty)
	jobType := string(config.JobType)
	if jobType == "" {
		jobType = string(models.JobTypeOnce)
	}
	snapshot.JobType = jobType

	// Set RespectRobotsTxt
	snapshot.RespectRobotsTxt = config.RespectRobotsTxt

	// Set CrawlMode (default to empty string if not set; runtime treats as pagination_and_links)
	snapshot.CrawlMode = string(config.CrawlMode)

	// Pass through QueueEndpointAssignments (stored in join table, not a column)
	snapshot.QueueEndpointAssignments = make([]snapshots.QueueEndpointAssignmentSnap, len(config.QueueEndpointAssignments))
	for i, a := range config.QueueEndpointAssignments {
		snapshot.QueueEndpointAssignments[i] = snapshots.QueueEndpointAssignmentSnap{
			EndpointID: a.EndpointID,
			Weight:     a.Weight,
		}
	}

	return snapshot, nil
}

func RestoreCrawlJobConfigFromSnapshot(snapshot snapshots.CrawlJobConfigSnapshot) (*models.CrawlJobConfig, error) {
	id, err := valueobjects.NewID(snapshot.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID: %w", err)
	}

	config := &models.CrawlJobConfig{
		ID:   id,
		Name: snapshot.Name,
	}
	if snapshot.UserID.Valid {
		userID, err := authvalueobjects.NewUserID(snapshot.UserID.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user id: %w", err)
		}
		config.UserID = userID
	}

	// Restore ExtractionSpec
	extractionSpecJSON, err := json.Marshal(snapshot.ExtractionSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ExtractionSpec from snapshot: %w", err)
	}
	if err := json.Unmarshal(extractionSpecJSON, &config.ExtractionSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ExtractionSpec: %w", err)
	}

	// Restore Scopes
	scopesJSON, err := json.Marshal(snapshot.Scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Scopes from snapshot: %w", err)
	}
	if err := json.Unmarshal(scopesJSON, &config.Scopes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Scopes: %w", err)
	}

	// Restore Seeds
	seedsJSON, err := json.Marshal(snapshot.Seeds)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Seeds from snapshot: %w", err)
	}
	if err := json.Unmarshal(seedsJSON, &config.Seeds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Seeds: %w", err)
	}

	// Restore RateLimit
	rateLimitJSON, err := json.Marshal(snapshot.RateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RateLimit from snapshot: %w", err)
	}
	if err := json.Unmarshal(rateLimitJSON, &config.RateLimit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RateLimit: %w", err)
	}

	// Restore Retries
	retriesJSON, err := json.Marshal(snapshot.Retries)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Retries from snapshot: %w", err)
	}
	if err := json.Unmarshal(retriesJSON, &config.Retries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Retries: %w", err)
	}

	// Restore Auth
	authJSON, err := json.Marshal(snapshot.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Auth from snapshot: %w", err)
	}
	if err := json.Unmarshal(authJSON, &config.Auth); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Auth: %w", err)
	}

	// Restore Schedule
	scheduleJSON, err := json.Marshal(snapshot.Schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Schedule from snapshot: %w", err)
	}
	if err := json.Unmarshal(scheduleJSON, &config.Schedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Schedule: %w", err)
	}

	// Restore JobType (default to ONCE if empty)
	if snapshot.JobType != "" {
		config.JobType = models.JobType(snapshot.JobType)
	} else {
		config.JobType = models.JobTypeOnce
	}

	// Restore RespectRobotsTxt
	config.RespectRobotsTxt = snapshot.RespectRobotsTxt

	// Restore CrawlMode
	config.CrawlMode = models.CrawlMode(snapshot.CrawlMode)

	// Pass through QueueEndpointAssignments (populated by repo from join table)
	config.QueueEndpointAssignments = make([]models.QueueEndpointAssignment, len(snapshot.QueueEndpointAssignments))
	for i, a := range snapshot.QueueEndpointAssignments {
		config.QueueEndpointAssignments[i] = models.QueueEndpointAssignment{
			EndpointID: a.EndpointID,
			Weight:     a.Weight,
		}
	}

	return config, nil
}
