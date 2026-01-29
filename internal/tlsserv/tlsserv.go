package tlsserv

import (
	"crypto/tls"

	"github.com/Okenamay/securawr/internal/server/config"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
)

func TLSInitialize(conf *config.Config, log *zap.Logger) (creds credentials.TransportCredentials, err error) {
	cert, err := tls.LoadX509KeyPair(conf.CertFile, conf.KeyFile)

	if err != nil {
		log.Fatal("Failed to load TLS keys", zap.Error(err))
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}
	creds = credentials.NewTLS(tlsConfig)

	return creds, err
}
