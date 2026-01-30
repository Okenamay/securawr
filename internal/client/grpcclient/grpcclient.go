package grpcclient

import (
	"context"
	"fmt"
	"time"

	pb "github.com/Okenamay/securawr/gen/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn       *grpc.ClientConn
	authClient pb.AuthServiceClient
	dataClient pb.DataServiceClient
}

// New создает нового клиента
// Если certFile пустой, используется insecure соединение (только для тестов)
func New(addr string, certFile string) (*Client, error) {
	var opts []grpc.DialOption

	if certFile != "" {
		creds, err := credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS cert: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return &Client{
		conn:       conn,
		authClient: pb.NewAuthServiceClient(conn),
		dataClient: pb.NewDataServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// --- Auth Methods ---

// GetAuthParams запрашивает соль аутентификации для логина.
func (c *Client) GetAuthParams(ctx context.Context, login string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.authClient.GetAuthParams(ctx, &pb.AuthParamsRequest{Login: login})
	if err != nil {
		return nil, fmt.Errorf("GetAuthParams failed: %w", err)
	}
	return resp.AuthSalt, nil
}

// Register регистрирует пользователя с переданными солями и ключом.
func (c *Client) Register(ctx context.Context, login string, authKey, authSalt, encSalt []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.authClient.Register(ctx, &pb.RegisterRequest{
		Login:          login,
		AuthKey:        authKey,
		AuthSalt:       authSalt,
		EncryptionSalt: encSalt,
	})
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration returned failure: %s", resp.Message)
	}
	return nil
}

// Login выполняет вход и возвращает токен и соль шифрования.
func (c *Client) Login(ctx context.Context, login string, authKey []byte) (string, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.authClient.Login(ctx, &pb.LoginRequest{
		Login:   login,
		AuthKey: authKey,
	})
	if err != nil {
		return "", nil, fmt.Errorf("login failed: %w", err)
	}

	return resp.Token, resp.EncryptionSalt, nil
}

// Data Methods

func (c *Client) AddData(ctx context.Context, req *pb.AddDataRequest) (*pb.AddDataResponse, error) {
	return c.dataClient.AddData(ctx, req)
}

func (c *Client) ListData(ctx context.Context, req *pb.ListDataRequest) (*pb.ListDataResponse, error) {
	return c.dataClient.ListData(ctx, req)
}

func (c *Client) GetData(ctx context.Context, req *pb.GetDataRequest) (*pb.GetDataResponse, error) {
	return c.dataClient.GetData(ctx, req)
}

func (c *Client) DeleteData(ctx context.Context, req *pb.DeleteDataRequest) (*pb.DeleteDataResponse, error) {
	return c.dataClient.DeleteData(ctx, req)
}
