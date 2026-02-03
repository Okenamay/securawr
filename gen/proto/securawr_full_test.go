package proto_test

import (
	"strings"
	"testing"

	pb "github.com/Okenamay/securawr/gen/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestDataType_Enum проверяет все значения перечисления DataType.
func TestDataType_Enum(t *testing.T) {
	tests := []struct {
		val  pb.DataType
		name string
		num  int32
	}{
		{pb.DataType_UNKNOWN, "UNKNOWN", 0},
		{pb.DataType_TEXT, "TEXT", 1},
		{pb.DataType_BINARY, "BINARY", 2},
		{pb.DataType_BANK_CARD, "BANK_CARD", 3},
		{pb.DataType_CREDENTIALS, "CREDENTIALS", 4},
	}

	for _, tt := range tests {
		// 1. Проверка String()
		if got := tt.val.String(); got != tt.name {
			t.Errorf("DataType(%d).String() = %s, want %s", tt.num, got, tt.name)
		}
		// 2. Проверка числового значения
		if int32(tt.val) != tt.num {
			t.Errorf("DataType(%s) number = %d, want %d", tt.name, tt.val, tt.num)
		}
		// 3. Проверка Descriptor (просто что не падает)
		if tt.val.Descriptor() == nil {
			t.Error("Enum Descriptor is nil")
		}
		if tt.val.Type() == nil {
			t.Error("Enum Type is nil")
		}
	}
}

// TestMessages_RoundTripSerialization проверяет, что все типы сообщений
// успешно сериализуются и десериализуются, сохраняя данные.
func TestMessages_RoundTripSerialization(t *testing.T) {
	now := timestamppb.Now()

	// Набор тестовых сообщений со всеми заполненными полями
	messages := []proto.Message{
		&pb.AuthParamsRequest{Login: "test_user"},
		&pb.AuthParamsResponse{AuthSalt: []byte("salt")},
		&pb.RegisterRequest{
			Login:          "user",
			AuthKey:        []byte("key"),
			AuthSalt:       []byte("salt"),
			EncryptionSalt: []byte("enc"),
		},
		&pb.RegisterResponse{Success: true, Message: "OK"},
		&pb.LoginRequest{Login: "user", AuthKey: []byte("key")},
		&pb.LoginResponse{Token: "jwt.token", EncryptionSalt: []byte("salt")},
		&pb.PingRequest{Message: "ping"},
		&pb.PingResponse{Message: "pong", Version: "v1"},
		&pb.AddDataRequest{
			Type:          pb.DataType_BINARY,
			EncryptedData: []byte("blob"),
			MetaInfo:      `{"a":1}`,
		},
		&pb.AddDataResponse{Id: "uuid-1"},
		&pb.ListDataRequest{TypeFilter: pb.DataType_TEXT},
		&pb.ListDataResponse{
			Items: []*pb.DataRecordInfo{
				{Id: "1", Type: pb.DataType_TEXT, MetaInfo: "{}", CreatedAt: "2023-01-01"},
			},
		},
		&pb.GetDataRequest{Id: "uuid-2"},
		&pb.GetDataResponse{
			Id:            "uuid-2",
			Type:          pb.DataType_UNKNOWN,
			EncryptedData: []byte("1234"),
			MetaInfo:      "{}",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		&pb.DeleteDataRequest{Id: "uuid-3"},
		&pb.DeleteDataResponse{Success: true},
	}

	for _, msg := range messages {
		t.Run(string(msg.ProtoReflect().Descriptor().Name()), func(t *testing.T) {
			// 1. Marshal
			data, err := proto.Marshal(msg)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// 2. Unmarshal
			// Создаем новый экземпляр того же типа
			newMsg := msg.ProtoReflect().New().Interface()
			err = proto.Unmarshal(data, newMsg)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// 3. Compare
			if !proto.Equal(msg, newMsg) {
				t.Error("Messages are not equal after round-trip serialization")
			}
		})
	}
}

// TestMessages_NilGetters проверяет безопасность вызова геттеров на nil-указателях.
// Это критично для избежания паник (NPE) в рантайме.
func TestMessages_NilGetters(t *testing.T) {
	t.Run("AuthParamsRequest", func(t *testing.T) {
		var m *pb.AuthParamsRequest
		if m.GetLogin() != "" {
			t.Error("GetLogin")
		}
	})
	t.Run("AuthParamsResponse", func(t *testing.T) {
		var m *pb.AuthParamsResponse
		if m.GetAuthSalt() != nil {
			t.Error("GetAuthSalt")
		}
	})
	t.Run("RegisterRequest", func(t *testing.T) {
		var m *pb.RegisterRequest
		if m.GetLogin() != "" {
			t.Error("GetLogin")
		}
		if m.GetAuthKey() != nil {
			t.Error("GetAuthKey")
		}
		if m.GetAuthSalt() != nil {
			t.Error("GetAuthSalt")
		}
		if m.GetEncryptionSalt() != nil {
			t.Error("GetEncryptionSalt")
		}
	})
	t.Run("RegisterResponse", func(t *testing.T) {
		var m *pb.RegisterResponse
		if m.GetSuccess() != false {
			t.Error("GetSuccess")
		}
		if m.GetMessage() != "" {
			t.Error("GetMessage")
		}
	})
	t.Run("LoginRequest", func(t *testing.T) {
		var m *pb.LoginRequest
		if m.GetLogin() != "" {
			t.Error("GetLogin")
		}
		if m.GetAuthKey() != nil {
			t.Error("GetAuthKey")
		}
	})
	t.Run("LoginResponse", func(t *testing.T) {
		var m *pb.LoginResponse
		if m.GetToken() != "" {
			t.Error("GetToken")
		}
		if m.GetEncryptionSalt() != nil {
			t.Error("GetEncryptionSalt")
		}
	})
	t.Run("PingRequest", func(t *testing.T) {
		var m *pb.PingRequest
		if m.GetMessage() != "" {
			t.Error("GetMessage")
		}
	})
	t.Run("PingResponse", func(t *testing.T) {
		var m *pb.PingResponse
		if m.GetMessage() != "" {
			t.Error("GetMessage")
		}
		if m.GetVersion() != "" {
			t.Error("GetVersion")
		}
	})
	t.Run("AddDataRequest", func(t *testing.T) {
		var m *pb.AddDataRequest
		if m.GetType() != pb.DataType_UNKNOWN {
			t.Error("GetType")
		}
		if m.GetEncryptedData() != nil {
			t.Error("GetEncryptedData")
		}
		if m.GetMetaInfo() != "" {
			t.Error("GetMetaInfo")
		}
	})
	t.Run("AddDataResponse", func(t *testing.T) {
		var m *pb.AddDataResponse
		if m.GetId() != "" {
			t.Error("GetId")
		}
	})
	t.Run("ListDataRequest", func(t *testing.T) {
		var m *pb.ListDataRequest
		if m.GetTypeFilter() != pb.DataType_UNKNOWN {
			t.Error("GetTypeFilter")
		}
	})
	t.Run("DataRecordInfo", func(t *testing.T) {
		var m *pb.DataRecordInfo
		if m.GetId() != "" {
			t.Error("GetId")
		}
		if m.GetType() != pb.DataType_UNKNOWN {
			t.Error("GetType")
		}
		if m.GetMetaInfo() != "" {
			t.Error("GetMetaInfo")
		}
		if m.GetCreatedAt() != "" {
			t.Error("GetCreatedAt")
		}
	})
	t.Run("ListDataResponse", func(t *testing.T) {
		var m *pb.ListDataResponse
		if m.GetItems() != nil {
			t.Error("GetItems")
		}
	})
	t.Run("GetDataRequest", func(t *testing.T) {
		var m *pb.GetDataRequest
		if m.GetId() != "" {
			t.Error("GetId")
		}
	})
	t.Run("GetDataResponse", func(t *testing.T) {
		var m *pb.GetDataResponse
		if m.GetId() != "" {
			t.Error("GetId")
		}
		if m.GetType() != pb.DataType_UNKNOWN {
			t.Error("GetType")
		}
		if m.GetEncryptedData() != nil {
			t.Error("GetEncryptedData")
		}
		if m.GetMetaInfo() != "" {
			t.Error("GetMetaInfo")
		}
		if m.GetCreatedAt() != nil {
			t.Error("GetCreatedAt")
		}
		if m.GetUpdatedAt() != nil {
			t.Error("GetUpdatedAt")
		}
	})
	t.Run("DeleteDataRequest", func(t *testing.T) {
		var m *pb.DeleteDataRequest
		if m.GetId() != "" {
			t.Error("GetId")
		}
	})
	t.Run("DeleteDataResponse", func(t *testing.T) {
		var m *pb.DeleteDataResponse
		if m.GetSuccess() != false {
			t.Error("GetSuccess")
		}
	})
}

// TestJSON_Marshaling проверяет, что JSON-сериализация работает (используя
// имена полей snake_case)
func TestJSON_Marshaling(t *testing.T) {
	req := &pb.AddDataRequest{
		Type:          pb.DataType_TEXT,
		EncryptedData: []byte("abc"),
		MetaInfo:      `{"key":"val"}`,
	}

	opts := protojson.MarshalOptions{
		UseProtoNames: true, // Использовать snake_case из .proto
	}

	b, err := opts.Marshal(req)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}

	jsonStr := string(b)

	// Проверяем наличие ключевых полей
	expectedFields := []string{
		`"type":"TEXT"`,           // Enum как строка
		`"encrypted_data":"YWJj"`, // []byte как base64 строка ("abc" -> "YWJj")
		`"meta_info":"{\"key\":\"val\"}"`,
	}

	for _, f := range expectedFields {
		if !strings.Contains(jsonStr, f) {
			t.Errorf("JSON output missing field: %s. Got: %s", f, jsonStr)
		}
	}

	// Десериализация (Round trip)
	var req2 pb.AddDataRequest
	if err := protojson.Unmarshal(b, &req2); err != nil {
		t.Fatalf("protojson.Unmarshal failed: %v", err)
	}

	if !proto.Equal(req, &req2) {
		t.Error("JSON RoundTrip mismatch")
	}
}
