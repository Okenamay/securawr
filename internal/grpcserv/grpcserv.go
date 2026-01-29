package grpcserv

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/handlers"
	"github.com/Okenamay/securawr/internal/server/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func RunServer(conf *config.Config, log *zap.Logger, creds credentials.TransportCredentials,
	store *storage.Storage) {

	lis, err := net.Listen("tcp", conf.GRPCPort)
	if err != nil {
		log.Fatal("Failed to listen", zap.String("port", conf.GRPCPort), zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
	)

	h := handlers.New(store, log)
	pb.RegisterSecuRawrServiceServer(grpcServer, h)

	// Graceful Shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Info("Shutting down server...")
		grpcServer.GracefulStop()
	}()

	log.Info("Server listening", zap.String("address", conf.GRPCPort))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("Server error", zap.Error(err))
	}
}
