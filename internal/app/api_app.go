package app

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"distributed-crawler/internal/auth"
	"distributed-crawler/internal/config"
	"distributed-crawler/internal/config/env"
	"distributed-crawler/internal/infra/logger"
	"distributed-crawler/internal/interceptor"
	crawlergrpc "distributed-crawler/pkg/v1"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

type APIApp struct {
	serviceProvider *serviceProvider
	grpcServer      *grpc.Server
	httpServer      *http.Server
	workerCtx       context.Context
	workerCancel    context.CancelFunc
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
		a.serviceProvider.Close()
		logger.Sync()
	}()

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()

		err := a.runGRPCServer()
		if err != nil {
			log.Fatalf("failed to run GRPC server: %v", err)
		}
	}()

	go func() {
		defer wg.Done()

		err := a.runHTTPServer()
		if err != nil {
			log.Fatalf("failed to run HTTP server: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		a.runWorker()
	}()

	wg.Wait()

	return nil
}

func (a *APIApp) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initServiceProvider,
		a.initDefaultAdmin,
		a.initGRPCServer,
		a.initHTTPServer,
		a.initWorker,
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

func (a *APIApp) initLogger(_ context.Context) error {
	loggerConfig, err := env.NewLoggerConfig()
	if err != nil {
		log.Fatalf("failed to get logger config: %v", err)
	}

	err = logger.InitWithConfig(loggerConfig.Level(), loggerConfig.Env())
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	return nil
}

func (a *APIApp) initServiceProvider(_ context.Context) error {
	a.serviceProvider = newServiceProvider()
	return nil
}

func (a *APIApp) initDefaultAdmin(ctx context.Context) error {
	if err := a.serviceProvider.EnsureDefaultAdmin(ctx); err != nil {
		log.Fatalf("failed to ensure default admin: %v", err)
	}
	return nil
}

func (a *APIApp) initGRPCServer(ctx context.Context) error {
	a.grpcServer = grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.UnaryInterceptor(
			grpcMiddleware.ChainUnaryServer(
				interceptor.LogInterceptor,
				interceptor.ValidateInterceptor,
				auth.JWTAuthInterceptor(a.serviceProvider.JWTService()),
				auth.RBACInterceptor(),
			),
		),
	)

	reflection.Register(a.grpcServer)

	crawlergrpc.RegisterCrawlerServiceServer(a.grpcServer, a.serviceProvider.CrawlerServiceImpl(ctx))
	crawlergrpc.RegisterPreviewServiceServer(a.grpcServer, a.serviceProvider.PreviewServiceImpl(ctx))
	crawlergrpc.RegisterAuthServiceServer(a.grpcServer, a.serviceProvider.AuthServiceImpl(ctx))
	crawlergrpc.RegisterUserServiceServer(a.grpcServer, a.serviceProvider.UserServiceImpl(ctx))
	crawlergrpc.RegisterWorkerServiceServer(a.grpcServer, a.serviceProvider.WorkerServiceImpl(ctx))

	return nil
}

func (a *APIApp) initHTTPServer(ctx context.Context) error {
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if strings.EqualFold(key, "x-preview-cookie") {
				return "x-preview-cookie", true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	err := crawlergrpc.RegisterCrawlerServiceHandlerFromEndpoint(ctx, mux, a.serviceProvider.GRPCConfig().Address(), opts)
	if err != nil {
		return err
	}

	err = crawlergrpc.RegisterPreviewServiceHandlerFromEndpoint(ctx, mux, a.serviceProvider.GRPCConfig().Address(), opts)
	if err != nil {
		return err
	}

	err = crawlergrpc.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, a.serviceProvider.GRPCConfig().Address(), opts)
	if err != nil {
		return err
	}

	err = crawlergrpc.RegisterUserServiceHandlerFromEndpoint(ctx, mux, a.serviceProvider.GRPCConfig().Address(), opts)
	if err != nil {
		return err
	}

	err = crawlergrpc.RegisterWorkerServiceHandlerFromEndpoint(ctx, mux, a.serviceProvider.GRPCConfig().Address(), opts)
	if err != nil {
		return err
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"}, // или []string{"*"} если без credentials
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept", "X-Preview-Cookie"},
		ExposedHeaders:   []string{"Grpc-Status", "Grpc-Message", "Grpc-Metadata-*", "Location"},
		AllowCredentials: true,
	})

	a.httpServer = &http.Server{
		Addr:    a.serviceProvider.HTTPConfig().Address(),
		Handler: c.Handler(mux),
	}

	return nil
}

func (a *APIApp) initWorker(ctx context.Context) error {
	a.workerCtx, a.workerCancel = context.WithCancel(ctx)
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

func (a *APIApp) runHTTPServer() error {
	log.Printf("HTTP server is running on %s", a.serviceProvider.HTTPConfig().Address())

	err := a.httpServer.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

func (a *APIApp) runWorker() {
	log.Printf("Outbox publisher worker is starting...")

	// Get worker from service provider with background context
	worker := a.serviceProvider.OutboxPublisher(context.Background())
	_ = worker

	// Start worker with worker context (can be cancelled)
	worker.Start(a.workerCtx)

	log.Printf("Outbox publisher worker stopped")
}
