package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	crawljob "distributed-crawler/internal/api/crawl_job"
	"distributed-crawler/internal/application/service"
	serviceMocks "distributed-crawler/internal/application/service/mocks"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func TestCreateJob(t *testing.T) {
	t.Parallel()
	type crawlJobServiceMockFunc func(mc *minimock.Controller) service.CrawlJobService

	type args struct {
		ctx context.Context
		req *crawlergrpc.CreateJobRequest
	}

	var (
		ctx = context.Background()
		mc  = minimock.NewController(t)

		id     = valueobjects.GenerateCrawlJobID()
		name   = gofakeit.Animal()
		status = gofakeit.RandomString([]string{"pending", "running", "completed", "failed"})

		serviceErr = fmt.Errorf("service error")

		req = &crawlergrpc.CreateJobRequest{
			Name:   name,
			Status: status,
		}

		command = service.CreateCrawlJobCommand{
			Name:   name,
			Status: status,
		}

		res = &crawlergrpc.CreateJobResponse{
			Id: id.String(),
		}
	)
	defer t.Cleanup(mc.Finish)

	tests := []struct {
		name                string
		args                args
		want                *crawlergrpc.CreateJobResponse
		err                 error
		crawlJobServiceMock crawlJobServiceMockFunc
	}{
		{
			name: "success case",
			args: args{
				ctx: ctx,
				req: req,
			},
			want: res,
			err:  nil,
			crawlJobServiceMock: func(mc *minimock.Controller) service.CrawlJobService {
				mock := serviceMocks.NewCrawlJobServiceMock(mc)
				mock.CreateCrawlJobMock.Expect(ctx, command).Return(id, nil)
				return mock
			},
		},
		{
			name: "service error case",
			args: args{
				ctx: ctx,
				req: req,
			},
			want: nil,
			err:  serviceErr,
			crawlJobServiceMock: func(mc *minimock.Controller) service.CrawlJobService {
				mock := serviceMocks.NewCrawlJobServiceMock(mc)
				mock.CreateCrawlJobMock.Expect(ctx, command).Return(valueobjects.CrawlJobID{}, serviceErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			crawlJobServiceMock := tt.crawlJobServiceMock(mc)
			api := crawljob.NewImplementation(crawlJobServiceMock)

			response, err := api.CreateJob(tt.args.ctx, tt.args.req)
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.want, response)
		})
	}
}
