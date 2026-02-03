package grpcclient

import (
	"context"
	"fmt"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Okenamay/securawr/gen/proto"
)

// --- Mocks ---

// mockAuthServer имитирует работу AuthService
type mockAuthServer struct {
	pb.UnimplementedAuthServiceServer
	// Поля для настройки поведения мока
	SaltToReturn  []byte
	TokenToReturn string
	RegisterError bool
}

func (m *mockAuthServer) GetAuthParams(ctx context.Context, req *pb.AuthParamsRequest) (*pb.AuthParamsResponse, error) {
	if req.Login == "error_user" {
		return nil, fmt.Errorf("user not found")
	}
	return &pb.AuthParamsResponse{AuthSalt: m.SaltToReturn}, nil
}

func (m *mockAuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if m.RegisterError {
		return &pb.RegisterResponse{Success: false, Message: "mock error"}, nil
	}
	return &pb.RegisterResponse{Success: true}, nil
}

func (m *mockAuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Login == "bad_creds" {
		return nil, fmt.Errorf("wrong password")
	}
	return &pb.LoginResponse{Token: m.TokenToReturn, EncryptionSalt: []byte("enc_salt")}, nil
}

// mockDataServer имитирует работу DataService
type mockDataServer struct {
	pb.UnimplementedDataServiceServer
}

func (m *mockDataServer) AddData(ctx context.Context, req *pb.AddDataRequest) (*pb.AddDataResponse, error) {
	return &pb.AddDataResponse{Id: "new-uuid-123"}, nil
}

// --- Helpers ---

// startTestServer запускает gRPC сервер на случайном свободном порту
// Возвращает адрес сервера и функцию для его остановки
func startTestServer(t *testing.T, authSrv pb.AuthServiceServer, dataSrv pb.DataServiceServer) (string, func()) {
	// :0 означает, что ОС сама выберет свободный порт
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	baseServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	pb.RegisterAuthServiceServer(baseServer, authSrv)
	pb.RegisterDataServiceServer(baseServer, dataSrv)

	go func() {
		if err := baseServer.Serve(lis); err != nil {
			// Ошибка при остановке сервера допустима
			t.Logf("Server stopped: %v", err)
		}
	}()

	return lis.Addr().String(), func() {
		baseServer.Stop()
		lis.Close()
	}
}

// --- Tests ---

func TestNewClient(t *testing.T) {
	// Проверяем создание клиента
	// Запускаем пустой сервер, чтобы Dial прошел успешно
	addr, stop := startTestServer(t, &mockAuthServer{}, &mockDataServer{})
	defer stop()

	// 1. Insecure Connection
	client, err := NewClient(addr, "")
	if err != nil {
		t.Fatalf("NewClient(insecure) failed: %v", err)
	}
	if client == nil {
		t.Fatal("Client is nil")
	}
	client.Close()

	// 2. TLS Connection (ожидаем ошибку, т.к. файл сертификата не существует)
	_, err = NewClient(addr, "non_existent_cert.pem")
	if err == nil {
		t.Fatal("Expected error for missing cert file, got nil")
	}
}

func TestClient_AuthMethods(t *testing.T) {
	// Настройка мока
	expectedSalt := []byte("test_salt")
	expectedToken := "jwt_token_example"

	mockAuth := &mockAuthServer{
		SaltToReturn:  expectedSalt,
		TokenToReturn: expectedToken,
	}

	addr, stop := startTestServer(t, mockAuth, &mockDataServer{})
	defer stop()

	client, err := NewClient(addr, "")
	if err != nil {
		t.Fatalf("Setup client failed: %v", err)
	}
	defer client.Close()
	ctx := context.Background()

	// --- Test GetAuthParams ---
	t.Run("GetAuthParams Success", func(t *testing.T) {
		salt, err := client.GetAuthParams(ctx, "test_user")
		if err != nil {
			t.Fatalf("GetAuthParams failed: %v", err)
		}
		if string(salt) != string(expectedSalt) {
			t.Errorf("Salt mismatch: got %v, want %v", salt, expectedSalt)
		}
	})

	t.Run("GetAuthParams Error", func(t *testing.T) {
		_, err := client.GetAuthParams(ctx, "error_user")
		if err == nil {
			t.Error("Expected error for 'error_user', got nil")
		}
	})

	// --- Test Register ---
	t.Run("Register Success", func(t *testing.T) {
		err := client.Register(ctx, "new_user", []byte("key"), []byte("salt"), []byte("enc"))
		if err != nil {
			t.Errorf("Register failed: %v", err)
		}
	})

	t.Run("Register Failure Response", func(t *testing.T) {
		mockAuth.RegisterError = true
		defer func() { mockAuth.RegisterError = false }() // Reset

		err := client.Register(ctx, "fail_user", nil, nil, nil)
		if err == nil {
			t.Error("Expected error when server returns success=false")
		}
	})

	// --- Test Login ---
	t.Run("Login Success", func(t *testing.T) {
		token, encSalt, err := client.Login(ctx, "user", []byte("key"))
		if err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if token != expectedToken {
			t.Errorf("Token mismatch: got %s, want %s", token, expectedToken)
		}
		if string(encSalt) != "enc_salt" {
			t.Error("EncSalt mismatch")
		}
	})
}

func TestClient_DataMethods(t *testing.T) {
	addr, stop := startTestServer(t, &mockAuthServer{}, &mockDataServer{})
	defer stop()

	client, _ := NewClient(addr, "")
	defer client.Close()

	// Проверяем проксирование метода AddData
	resp, err := client.AddData(context.Background(), &pb.AddDataRequest{})
	if err != nil {
		t.Fatalf("AddData failed: %v", err)
	}
	if resp.Id != "new-uuid-123" {
		t.Errorf("AddData returned unexpected ID: %s", resp.Id)
	}
}

func TestClient_DataMethods_Extended(t *testing.T) {
	// Расширяем мок для поддержки новых методов
	mockData := &mockDataServer{}

	addr, stop := startTestServer(t, &mockAuthServer{}, mockData)
	defer stop()

	client, err := NewClient(addr, "")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()
	ctx := context.Background()

	// 1. ListData
	_, err = client.ListData(ctx, &pb.ListDataRequest{
		TypeFilter: pb.DataType_TEXT, // 1 соответствует TEXT
	})
	if err != nil {
		t.Errorf("ListData failed: %v", err)
	}

	// 2. GetData
	_, err = client.GetData(ctx, &pb.GetDataRequest{
		Id: "some-uuid",
	})
	if err != nil {
		t.Errorf("GetData failed: %v", err)
	}

	// 3. DeleteData
	_, err = client.DeleteData(ctx, &pb.DeleteDataRequest{
		Id: "some-uuid",
	})
	if err != nil {
		t.Errorf("DeleteData failed: %v", err)
	}
}

// Дополним mockDataServer методами
func (m *mockDataServer) ListData(ctx context.Context, req *pb.ListDataRequest) (*pb.ListDataResponse, error) {
	return &pb.ListDataResponse{Items: []*pb.DataRecordInfo{}}, nil
}
func (m *mockDataServer) GetData(ctx context.Context, req *pb.GetDataRequest) (*pb.GetDataResponse, error) {
	return &pb.GetDataResponse{Id: req.Id}, nil
}
func (m *mockDataServer) DeleteData(ctx context.Context, req *pb.DeleteDataRequest) (*pb.DeleteDataResponse, error) {
	return &pb.DeleteDataResponse{Success: true}, nil
}
