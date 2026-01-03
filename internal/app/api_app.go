package app

import (
	"context"
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"distributed-crawler/internal/config"
	crawlergrpc "distributed-crawler/pkg/v1"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

type APIApp struct {
	serviceProvider *serviceProvider
	grpcServer      *grpc.Server
}

func NewAPIApp(ctx context.Context) (*APIApp, error) {
	a := &APIApp{}

	err := a.initDeps(ctx)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *APIApp) Run() error {
	defer func() {
		if err := a.serviceProvider.Close(); err != nil {
			log.Printf("failed to close resources: %v", err)
		}
	}()

	return a.runGRPCServer()
}

func (a *APIApp) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initServiceProvider,
		a.initGRPCServer,
	}

	for _, f := range inits {
		err := f(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *APIApp) initConfig(_ context.Context) error {
	err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	return nil
}

func (a *APIApp) initServiceProvider(_ context.Context) error {
	a.serviceProvider = newServiceProvider()
	return nil
}

func (a *APIApp) initGRPCServer(ctx context.Context) error {
	a.grpcServer = grpc.NewServer(grpc.Creds(insecure.NewCredentials()))

	reflection.Register(a.grpcServer)

	crawlergrpc.RegisterCrawlerServiceServer(a.grpcServer, a.serviceProvider.CrawlerServiceImpl(ctx))

	return nil
}

func (a *APIApp) runGRPCServer() error {
	log.Printf("GRPC server is running on %s", a.serviceProvider.GRPCConfig().Address())

	list, err := net.Listen("tcp", a.serviceProvider.GRPCConfig().Address())
	if err != nil {
		return err
	}

	err = a.grpcServer.Serve(list)
	if err != nil {
		return err
	}

	return nil
}
