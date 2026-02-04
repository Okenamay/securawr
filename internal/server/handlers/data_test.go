package handlers

import (
	"context"
	"testing"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDataHandler_CRUD(t *testing.T) {
	store, cfg, logger, teardown := setupTestEnv(t)
	defer teardown()

	handler := NewDataHandler(store, cfg, logger)
	ctx := context.Background()

	// 1. Создаем пользователя в БД, чтобы получить ID
	user, err := store.CreateUser(ctx, "data_user", "hash", []byte("s"), []byte("s"))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Подготавливаем контекст с UserID (имитация AuthInterceptor)
	ctx = auth.ContextWithUserID(ctx, user.ID.String())

	// --- Test AddData ---
	dataBlob := []byte("my secret data")
	reqAdd := &pb.AddDataRequest{
		Type:          pb.DataType_UNKNOWN,
		EncryptedData: dataBlob,
		MetaInfo:      `{"name": "test.txt"}`,
	}

	respAdd, err := handler.AddData(ctx, reqAdd)
	if err != nil {
		t.Fatalf("AddData failed: %v", err)
	}
	if respAdd.Id == "" {
		t.Error("Returned empty ID")
	}
	dataID := respAdd.Id

	// --- Test GetData ---
	reqGet := &pb.GetDataRequest{Id: dataID}
	respGet, err := handler.GetData(ctx, reqGet)
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if string(respGet.EncryptedData) != string(dataBlob) {
		t.Error("Data mismatch")
	}
	if respGet.Type != pb.DataType_UNKNOWN {
		t.Error("Type mismatch")
	}

	// --- Test ListData ---
	reqList := &pb.ListDataRequest{}
	respList, err := handler.ListData(ctx, reqList)
	if err != nil {
		t.Fatalf("ListData failed: %v", err)
	}
	if len(respList.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(respList.Items))
	}
	if respList.Items[0].Id != dataID {
		t.Error("List returned wrong ID")
	}

	// --- Test DeleteData ---
	reqDel := &pb.DeleteDataRequest{Id: dataID}
	_, err = handler.DeleteData(ctx, reqDel)
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	// Проверяем удаление
	_, err = handler.GetData(ctx, reqGet)
	if status.Code(err) != codes.NotFound {
		t.Errorf("Expected NotFound after delete, got: %v", err)
	}
}

func TestDataHandler_Quota(t *testing.T) {
	store, cfg, logger, teardown := setupTestEnv(t)
	defer teardown()

	// Устанавливаем маленькую квоту
	cfg.UserQuota = 100 // байт
	handler := NewDataHandler(store, cfg, logger)

	ctx := context.Background()
	user, _ := store.CreateUser(ctx, "quota_user", "h", []byte("s"), []byte("s"))
	ctx = auth.ContextWithUserID(ctx, user.ID.String())

	// 1. Пытаемся добавить данные больше квоты
	largeData := make([]byte, 150)
	req := &pb.AddDataRequest{EncryptedData: largeData, Type: pb.DataType_UNKNOWN}

	_, err := handler.AddData(ctx, req)
	if status.Code(err) != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted, got: %v", err)
	}

	// 2. Добавляем данные в рамках квоты
	smallData := make([]byte, 50)
	reqSmall := &pb.AddDataRequest{EncryptedData: smallData, Type: pb.DataType_UNKNOWN}
	_, err = handler.AddData(ctx, reqSmall)
	if err != nil {
		t.Fatalf("Failed to add data within quota: %v", err)
	}

	// 3. Добавляем еще, чтобы превысить суммарную квоту (50 + 60 > 100)
	reqOverflow := &pb.AddDataRequest{EncryptedData: make([]byte, 60), Type: pb.DataType_UNKNOWN}
	_, err = handler.AddData(ctx, reqOverflow)
	if status.Code(err) != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted on second upload, got: %v", err)
	}
}
