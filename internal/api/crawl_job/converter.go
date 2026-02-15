package crawljob

import (
	"distributed-crawler/internal/domain/crawl/models"
	crawlergrpc "distributed-crawler/pkg/v1"
	"encoding/json"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToProtoAuthOptions converts domain AuthOptions to protobuf
func ToProtoAuthOptions(auth models.AuthOptions) *crawlergrpc.AuthOptions {
	return &crawlergrpc.AuthOptions{
		Cookie:        auth.Cookie,
		BasicUser:     auth.BasicUser,
		BasicPassword: auth.BasicPassword,
		BearerToken:   auth.BearerToken,
	}
}

// ToProtoRateLimitPolicy converts domain RateLimitPolicy to protobuf
func ToProtoRateLimitPolicy(rateLimit models.RateLimitPolicy) *crawlergrpc.RateLimitPolicy {
	return &crawlergrpc.RateLimitPolicy{
		Rps: rateLimit.Rps,
	}
}

// ToProtoRetryPolicy converts domain RetryPolicy to protobuf
func ToProtoRetryPolicy(retry models.RetryPolicy) *crawlergrpc.RetryPolicy {
	return &crawlergrpc.RetryPolicy{
		MaxAttempts:       retry.MaxAttempts,
		BackoffInitialMs:  retry.BackoffInitialMs,
		BackoffMultiplier: retry.BackoffMultiplier,
	}
}

// ToProtoScheduleOptions converts domain ScheduleOptions to protobuf
func ToProtoScheduleOptions(schedule models.ScheduleOptions) *crawlergrpc.ScheduleOptions {
	return &crawlergrpc.ScheduleOptions{
		Cron: schedule.Cron,
	}
}

// ToProtoScopeRules converts domain ScopeRules to protobuf
func ToProtoScopeRules(scope models.ScopeRules) *crawlergrpc.ScopeRules {
	return &crawlergrpc.ScopeRules{
		MaxDepth:        scope.MaxDepth,
		AllowedDomains:  scope.AllowedDomains,
		DenyUrlPatterns: scope.DenyUrlPatterns,
	}
}

// ToProtoSeed converts domain Seed to protobuf
func ToProtoSeed(seed models.Seed) *crawlergrpc.Seed {
	return &crawlergrpc.Seed{
		Url: seed.Url,
	}
}

// ToProtoExtractorSpec converts domain ExtractorSpec to protobuf
func ToProtoExtractorSpec(spec models.ExtractorSpec) *crawlergrpc.ExtractorSpec {
	protoSpec := &crawlergrpc.ExtractorSpec{
		Selector:  spec.Selector,
		Attribute: spec.Attribute,
		Multiple:  spec.Multiple,
	}

	if spec.Index != nil {
		idx := int32(*spec.Index)
		protoSpec.Index = &idx
	}

	return protoSpec
}

// ToProtoTransformSpec converts domain TransformSpec to protobuf
func ToProtoTransformSpec(spec models.TransformSpec) *crawlergrpc.TransformSpec {
	protoSpec := &crawlergrpc.TransformSpec{
		Op: string(spec.Op),
	}

	if spec.Arg != nil {
		// Marshal arg to JSON string
		argJSON, _ := json.Marshal(spec.Arg)
		protoSpec.Arg = string(argJSON)
	}

	return protoSpec
}

// ToProtoFieldSpec converts domain FieldSpec to protobuf
func ToProtoFieldSpec(field models.FieldSpec) *crawlergrpc.FieldSpec {
	transforms := make([]*crawlergrpc.TransformSpec, len(field.Transforms))
	for i, t := range field.Transforms {
		transforms[i] = ToProtoTransformSpec(t)
	}

	return &crawlergrpc.FieldSpec{
		Name:       field.Name,
		Type:       string(field.Type),
		Required:   field.Required,
		Extractor:  ToProtoExtractorSpec(field.Extractor),
		Transforms: transforms,
	}
}

// ToProtoMetricSpec converts domain MetricSpec to protobuf
func ToProtoMetricSpec(metric models.MetricSpec) *crawlergrpc.MetricSpec {
	return &crawlergrpc.MetricSpec{
		Name:  metric.Name,
		Op:    string(metric.Op),
		Input: metric.Input,
	}
}

// ToProtoPaginationSpec converts domain PaginationSpec to protobuf
func ToProtoPaginationSpec(spec models.PaginationSpec) *crawlergrpc.PaginationSpec {
	return &crawlergrpc.PaginationSpec{
		Name:      spec.Name,
		Selector:  spec.Selector,
		Attribute: spec.Attribute,
		Multiple:  spec.Multiple,
	}
}

// ToProtoExtractionSpec converts domain ExtractionSpec to protobuf
func ToProtoExtractionSpec(spec models.ExtractionSpec) *crawlergrpc.ExtractionSpec {
	fields := make([]*crawlergrpc.FieldSpec, len(spec.Fields))
	for i, f := range spec.Fields {
		fields[i] = ToProtoFieldSpec(f)
	}

	metrics := make([]*crawlergrpc.MetricSpec, len(spec.Metrics))
	for i, m := range spec.Metrics {
		metrics[i] = ToProtoMetricSpec(m)
	}

	pagination := make([]*crawlergrpc.PaginationSpec, len(spec.Pagination))
	for i, p := range spec.Pagination {
		pagination[i] = ToProtoPaginationSpec(p)
	}

	return &crawlergrpc.ExtractionSpec{
		Fields:     fields,
		Metrics:    metrics,
		Pagination: pagination,
	}
}

// ToProtoJobType converts domain JobType to protobuf
func ToProtoJobType(jobType models.JobType) crawlergrpc.JobType {
	switch jobType {
	case models.JobTypeScheduled:
		return crawlergrpc.JobType_JOB_TYPE_SCHEDULED
	case models.JobTypeOnce:
		return crawlergrpc.JobType_JOB_TYPE_ONCE
	default:
		return crawlergrpc.JobType_JOB_TYPE_ONCE
	}
}

// ToProtoCrawlJobConfig converts domain CrawlJobConfig to protobuf
func ToProtoCrawlJobConfig(config *models.CrawlJobConfig) *crawlergrpc.CrawlJobConfig {
	if config == nil {
		return nil
	}

	seeds := make([]*crawlergrpc.Seed, len(config.Seeds))
	for i, s := range config.Seeds {
		seeds[i] = ToProtoSeed(s)
	}

	return &crawlergrpc.CrawlJobConfig{
		Id:               config.ID.String(),
		Name:             config.Name,
		ExtractionSpec:   ToProtoExtractionSpec(config.ExtractionSpec),
		Scopes:           ToProtoScopeRules(config.Scopes),
		Seeds:            seeds,
		RateLimit:        ToProtoRateLimitPolicy(config.RateLimit),
		Retries:          ToProtoRetryPolicy(config.Retries),
		Auth:             ToProtoAuthOptions(config.Auth),
		Schedule:         ToProtoScheduleOptions(config.Schedule),
		JobType:          ToProtoJobType(config.JobType),
		RespectRobotsTxt: config.RespectRobotsTxt,
	}
}

// ToProtoCrawlJob converts domain CrawlJob to protobuf CrawlJob
func ToProtoCrawlJob(job *models.CrawlJob) *crawlergrpc.CrawlJob {
	if job == nil {
		return nil
	}

	protoJob := &crawlergrpc.CrawlJob{
		Id:           job.ID.String(),
		JobConfigId:  job.JobConfigID.String(),
		JobConfig:    ToProtoCrawlJobConfig(job.JobConfig),
		Status:       job.Status.String(),
		CreatedAt:    timestamppb.New(job.CreatedAt),
		ExportStatus: job.ExportStatus.String(),
	}

	if job.CompletedAt != nil {
		protoJob.CompletedAt = timestamppb.New(*job.CompletedAt)
	}

	if job.ExportJSONKey != nil {
		protoJob.ExportJsonKey = job.ExportJSONKey
	}

	if job.ExportCSVKey != nil {
		protoJob.ExportCsvKey = job.ExportCSVKey
	}

	if job.ExportedAt != nil {
		protoJob.ExportedAt = timestamppb.New(*job.ExportedAt)
	}

	return protoJob
}

// ToProtoCrawlTask converts domain CrawlTask to protobuf CrawlTask
func ToProtoCrawlTask(task *models.CrawlTask) *crawlergrpc.CrawlTask {
	if task == nil {
		return nil
	}

	protoTask := &crawlergrpc.CrawlTask{
		Id:             task.ID.String(),
		JobId:          task.JobID.String(),
		Job:            ToProtoCrawlJob(task.Job),
		Url:            task.URL,
		Status:         task.Status.String(),
		EnqueuedAt:     timestamppb.New(task.EnqueuedAt),
		Depth:          task.Depth,
		MinioObjectKey: task.MinioObjectKey,
	}

	if task.FinalURL != nil {
		protoTask.FinalUrl = task.FinalURL
	}

	if task.ResultObjectKey != nil {
		protoTask.ResultObjectKey = task.ResultObjectKey
	}

	if task.ErrorMessage != nil {
		protoTask.ErrorMessage = task.ErrorMessage
	}

	return protoTask
}

// FromProtoAuthOptions converts protobuf AuthOptions to domain
func FromProtoAuthOptions(proto *crawlergrpc.AuthOptions) models.AuthOptions {
	if proto == nil {
		return models.AuthOptions{}
	}

	return models.AuthOptions{
		Cookie:        proto.Cookie,
		BasicUser:     proto.BasicUser,
		BasicPassword: proto.BasicPassword,
		BearerToken:   proto.BearerToken,
	}
}

// FromProtoRateLimitPolicy converts protobuf RateLimitPolicy to domain
func FromProtoRateLimitPolicy(proto *crawlergrpc.RateLimitPolicy) models.RateLimitPolicy {
	if proto == nil {
		return models.RateLimitPolicy{}
	}

	return models.RateLimitPolicy{
		Rps: proto.Rps,
	}
}

// FromProtoRetryPolicy converts protobuf RetryPolicy to domain
func FromProtoRetryPolicy(proto *crawlergrpc.RetryPolicy) models.RetryPolicy {
	if proto == nil {
		return models.RetryPolicy{}
	}

	return models.RetryPolicy{
		MaxAttempts:       proto.MaxAttempts,
		BackoffInitialMs:  proto.BackoffInitialMs,
		BackoffMultiplier: proto.BackoffMultiplier,
	}
}

// FromProtoScheduleOptions converts protobuf ScheduleOptions to domain
func FromProtoScheduleOptions(proto *crawlergrpc.ScheduleOptions) models.ScheduleOptions {
	if proto == nil {
		return models.ScheduleOptions{}
	}

	return models.ScheduleOptions{
		Cron: proto.Cron,
	}
}

// FromProtoScopeRules converts protobuf ScopeRules to domain
func FromProtoScopeRules(proto *crawlergrpc.ScopeRules) models.ScopeRules {
	if proto == nil {
		return models.ScopeRules{}
	}

	return models.ScopeRules{
		MaxDepth:        proto.MaxDepth,
		AllowedDomains:  proto.AllowedDomains,
		DenyUrlPatterns: proto.DenyUrlPatterns,
	}
}

// FromProtoSeed converts protobuf Seed to domain
func FromProtoSeed(proto *crawlergrpc.Seed) models.Seed {
	if proto == nil {
		return models.Seed{}
	}

	return models.Seed{
		Url: proto.Url,
	}
}

// FromProtoExtractorSpec converts protobuf ExtractorSpec to domain
func FromProtoExtractorSpec(proto *crawlergrpc.ExtractorSpec) models.ExtractorSpec {
	if proto == nil {
		return models.ExtractorSpec{}
	}

	spec := models.ExtractorSpec{
		Selector:  proto.Selector,
		Attribute: proto.Attribute,
		Multiple:  proto.Multiple,
	}

	if proto.Index != nil {
		idx := int(*proto.Index)
		spec.Index = &idx
	}

	return spec
}

// FromProtoTransformSpec converts protobuf TransformSpec to domain
func FromProtoTransformSpec(proto *crawlergrpc.TransformSpec) models.TransformSpec {
	if proto == nil {
		return models.TransformSpec{}
	}

	spec := models.TransformSpec{
		Op: models.TransformOp(proto.Op),
	}

	if proto.Arg != "" {
		var arg any
		_ = json.Unmarshal([]byte(proto.Arg), &arg)
		spec.Arg = arg
	}

	return spec
}

// FromProtoFieldSpec converts protobuf FieldSpec to domain
func FromProtoFieldSpec(proto *crawlergrpc.FieldSpec) models.FieldSpec {
	if proto == nil {
		return models.FieldSpec{}
	}

	transforms := make([]models.TransformSpec, len(proto.Transforms))
	for i, t := range proto.Transforms {
		transforms[i] = FromProtoTransformSpec(t)
	}

	return models.FieldSpec{
		Name:       proto.Name,
		Type:       models.ValueType(proto.Type),
		Required:   proto.Required,
		Extractor:  FromProtoExtractorSpec(proto.Extractor),
		Transforms: transforms,
	}
}

// FromProtoMetricSpec converts protobuf MetricSpec to domain
func FromProtoMetricSpec(proto *crawlergrpc.MetricSpec) models.MetricSpec {
	if proto == nil {
		return models.MetricSpec{}
	}

	return models.MetricSpec{
		Name:  proto.Name,
		Op:    models.MetricOp(proto.Op),
		Input: proto.Input,
	}
}

// FromProtoPaginationSpec converts protobuf PaginationSpec to domain
func FromProtoPaginationSpec(proto *crawlergrpc.PaginationSpec) models.PaginationSpec {
	if proto == nil {
		return models.PaginationSpec{}
	}

	return models.PaginationSpec{
		Name:      proto.Name,
		Selector:  proto.Selector,
		Attribute: proto.Attribute,
		Multiple:  proto.Multiple,
	}
}

// FromProtoExtractionSpec converts protobuf ExtractionSpec to domain
func FromProtoExtractionSpec(proto *crawlergrpc.ExtractionSpec) models.ExtractionSpec {
	if proto == nil {
		return models.ExtractionSpec{}
	}

	fields := make([]models.FieldSpec, len(proto.Fields))
	for i, f := range proto.Fields {
		fields[i] = FromProtoFieldSpec(f)
	}

	metrics := make([]models.MetricSpec, len(proto.Metrics))
	for i, m := range proto.Metrics {
		metrics[i] = FromProtoMetricSpec(m)
	}

	pagination := make([]models.PaginationSpec, len(proto.Pagination))
	for i, p := range proto.Pagination {
		pagination[i] = FromProtoPaginationSpec(p)
	}

	return models.ExtractionSpec{
		Fields:     fields,
		Metrics:    metrics,
		Pagination: pagination,
	}
}

// FromProtoJobType converts protobuf JobType to domain
func FromProtoJobType(jobType crawlergrpc.JobType) models.JobType {
	switch jobType {
	case crawlergrpc.JobType_JOB_TYPE_SCHEDULED:
		return models.JobTypeScheduled
	case crawlergrpc.JobType_JOB_TYPE_ONCE:
		return models.JobTypeOnce
	default:
		return models.JobTypeOnce
	}
}

// FromProtoCrawlJobConfig converts protobuf CrawlJobConfig to domain
func FromProtoCrawlJobConfig(proto *crawlergrpc.CrawlJobConfig) models.CrawlJobConfig {
	if proto == nil {
		return models.CrawlJobConfig{}
	}

	seeds := make([]models.Seed, len(proto.Seeds))
	for i, s := range proto.Seeds {
		seeds[i] = FromProtoSeed(s)
	}

	return models.CrawlJobConfig{
		// ID will be generated in the service layer
		Name:             proto.Name,
		ExtractionSpec:   FromProtoExtractionSpec(proto.ExtractionSpec),
		Scopes:           FromProtoScopeRules(proto.Scopes),
		Seeds:            seeds,
		RateLimit:        FromProtoRateLimitPolicy(proto.RateLimit),
		Retries:          FromProtoRetryPolicy(proto.Retries),
		Auth:             FromProtoAuthOptions(proto.Auth),
		Schedule:         FromProtoScheduleOptions(proto.Schedule),
		JobType:          FromProtoJobType(proto.JobType),
		RespectRobotsTxt: proto.RespectRobotsTxt,
	}
}
