/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/hpe/access-manager/internal/services/identity"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hpe/access-manager/config"
	access_manager "github.com/hpe/access-manager/internal/services/access-manager"
	"github.com/hpe/access-manager/internal/services/metadata"
	"github.com/hpe/access-manager/pkg/logger"
	"github.com/hpe/access-manager/pkg/metrics"
	"github.com/hpe/access-manager/pkg/middleware"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	serviceName = "access-manager"
)

func main() {
	if err := run(context.Background()); err != nil {
		logger.GetLogger().Err(err).Msg("could not start")
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.ReadAccessManagerConfig()
	if err != nil {
		return err
	}
	logLevel := os.Getenv("LOG_LEVEL")
	logger.InitLogger(logLevel, serviceName, os.Stdout)

	metricsService := metrics.NewMetrics()
	go metricsService.StartMetricsServer(fmt.Sprintf(":%d", cfg.MetricServerPort))

	s := grpc.NewServer(grpc.UnaryInterceptor(middleware.TraceIDInterceptor))

	am, ms, err := initiateMetadataConnections()
	if err != nil {
		return err
	}
	access_manager.RegisterAccessManagerServer(
		s,
		access_manager.NewAccessManager(am, metricsService),
	)
	reflection.Register(s)

	// Implement health check for access manager
	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthService)

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start GRPC server before the gateway
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return err
	}
	// Start the service listening for requests.
	go func() {
		logger.GetLogger().Info().Msg(fmt.Sprintf("GRPC server listening on %s", strconv.Itoa(cfg.Port)))
		serverErrors <- s.Serve(listener)
	}()

	// Configure Rest Gateway to expose RPC via Rest
	// ===========>
	mux := runtime.NewServeMux(middleware.AddTraceIDInGateway())
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if err = access_manager.RegisterAccessManagerHandlerFromEndpoint(ctx, mux, fmt.Sprintf(":%d", cfg.Port), opts); err != nil {
		return errors.Wrap(err, "unable to register access manager gateway handler")
	}
	err = mux.HandlePath(http.MethodGet, "/health", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
		w.WriteHeader(http.StatusOK)
	})
	if err != nil {
		return err
	}
	gatewayHTTPServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.GatewayPort),
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           middleware.AllowCORS(mux),
	}
	go func() {
		logger.GetLogger().Info().Msg(fmt.Sprintf("Gateway server started at %s", strconv.Itoa(cfg.GatewayPort)))
		serverErrors <- gatewayHTTPServer.ListenAndServe()
	}()

	// set up the integrated identity service
	identity.NewIntegratedIdentityService(cfg.ServerKey, 2222, ms)
	// ============================

	// Shutdown

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")

	case sig := <-shutdown:
		logger.GetLogger().Info().Msg(fmt.Sprintf("main: %v: Start shutdown", sig))
		// gracefully shutdown and shed load.
		s.GracefulStop()
		err := gatewayHTTPServer.Shutdown(ctx)
		if err != nil {
			logger.GetLogger().Error().Err(err).Msg("could not stop gateway server")
			return err
		}
	}

	return nil
}

func initiateMetadataConnections() (access_manager.PermissionLogic, metadata.MetaStore, error) {
	metaStore, err := metadata.OpenEtcdMetaStore([]string{"http://localhost:2379"})
	if err != nil {
		logger.GetLogger().Err(err).Msg("unable open etcd connection")
		return nil, nil, err
	}
	am := access_manager.NewPermissionLogic(metaStore)
	return am, metaStore, nil
}
