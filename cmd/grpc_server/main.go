package main

import (
	"context"
	"distributed-crawler/internal/app"
	"log"
)

func main() {
	ctx := context.Background()

	app, err := app.NewAPIApp(ctx)
	if err != nil {
		log.Fatalf("failed to init app: %s", err.Error())
	}

	err = app.Run()
	if err != nil {
		log.Fatalf("failed to run app: %s", err.Error())
	}
}

// func (s *mock) CreateJob(ctx context.Context, req *crawlergrpc.CreateJobRequest) (*crawlergrpc.CreateJobResponse, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	header := metadata.Pairs("header-key", "some-key")
// 	grpc.SetHeader(ctx, header)

// 	if err := s.validator.Validate(req); err != nil {
// 		st := status.New(codes.InvalidArgument, codes.InvalidArgument.String())
// 		st, _ = st.WithDetails(&errdetails.BadRequest{
// 			FieldViolations: []*errdetails.BadRequest_FieldViolation{
// 				{
// 					Field:       "some field",
// 					Description: err.Error(),
// 				},
// 			},
// 		})

// 		return nil, st.Err()
// 	}

// 	s.jobCounter++
// 	jobID := fmt.Sprintf("job-%d", s.jobCounter)

// 	job := &crawlergrpc.CrawlJob{
// 		Id:        jobID,
// 		Name:      req.Name,
// 		Status:    req.Status,
// 		CreatedAt: timestamppb.New(time.Now()),
// 	}

// 	s.jobs[jobID] = job

// 	return &crawlergrpc.CreateJobResponse{
// 		Job: job,
// 	}, nil
// }
