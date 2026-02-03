package proto_test

import (
	"bytes"
	"testing"

	pb "github.com/Okenamay/securawr/gen/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// TestDataTypeEnum проверяет, что перечисления (Enums) корректно преобразуются
// в строки
func TestDataTypeEnum(t *testing.T) {
	tests := []struct {
		d        pb.DataType
		expected string
		number   int32
	}{
		{pb.DataType_UNKNOWN, "UNKNOWN", 0},
		{pb.DataType_TEXT, "TEXT", 1},
		{pb.DataType_BINARY, "BINARY", 2},
		{pb.DataType_BANK_CARD, "BANK_CARD", 3},
		{pb.DataType_CREDENTIALS, "CREDENTIALS", 4},
	}

	for _, tt := range tests {
		// Проверка метода String()
		if got := tt.d.String(); got != tt.expected {
			t.Errorf("DataType(%d).String() = %s, want %s", tt.number, got, tt.expected)
		}
		// Проверка числового значения
		if int32(tt.d) != tt.number {
			t.Errorf("DataType(%s) number = %d, want %d", tt.expected, tt.d, tt.number)
		}
	}
}

// TestSerialization_RegisterRequest проверяет базовую сериализацию Protobuf
func TestSerialization_RegisterRequest(t *testing.T) {
	// 1. Создаем исходный объект
	original := &pb.RegisterRequest{
		Login:          "test_user",
		AuthKey:        []byte("secret_auth_key"),
		AuthSalt:       []byte("random_salt_1"),
		EncryptionSalt: []byte("random_salt_2"),
	}

	// 2. Сериализуем в байты (wire format)
	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("proto.Marshal failed: %v", err)
	}

	// 3. Десериализуем обратно в новую структуру
	restored := &pb.RegisterRequest{}
	if err := proto.Unmarshal(data, restored); err != nil {
		t.Fatalf("proto.Unmarshal failed: %v", err)
	}

	// 4. Сравниваем поля
	if original.Login != restored.Login {
		t.Errorf("Login mismatch: want %s, got %s", original.Login, restored.Login)
	}
	if !bytes.Equal(original.AuthKey, restored.AuthKey) {
		t.Error("AuthKey mismatch")
	}
	if !bytes.Equal(original.AuthSalt, restored.AuthSalt) {
		t.Error("AuthSalt mismatch")
	}
}

// TestJSON_AddDataRequest проверяет сериализацию в JSON (используя protojson)
func TestJSON_AddDataRequest(t *testing.T) {
	req := &pb.AddDataRequest{
		Type:          pb.DataType_TEXT,
		EncryptedData: []byte("encrypted_content"),
		MetaInfo:      `{"filename":"secret.txt"}`,
	}

	// Используем protojson для корректной работы с proto-специфичными полями
	marshaller := protojson.MarshalOptions{
		UseProtoNames: true, // Использовать имена из .proto (snake_case), а не
		// CamelCase
	}

	jsonData, err := marshaller.Marshal(req)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}

	// Проверяем наличие ключевых полей в JSON строке
	jsonStr := string(jsonData)
	expectedSubstrings := []string{
		`"type":"TEXT"`, // Enum по умолчанию сериализуется как строка
		`"encrypted_data"`,
		`"meta_info"`,
	}

	for _, sub := range expectedSubstrings {
		if !bytes.Contains(jsonData, []byte(sub)) {
			t.Errorf("JSON output missing %s. Got: %s", sub, jsonStr)
		}
	}
}

// TestGetters проверяет, что геттеры обрабатывают nil-receiver корректно (безопасность).
func TestGetters(t *testing.T) {
	var req *pb.LoginRequest // nil указатель

	// GetLogin должен вернуть пустую строку, а не паниковать
	if val := req.GetLogin(); val != "" {
		t.Errorf("Expected empty string from nil struct getter, got %s", val)
	}

	// GetAuthKey должен вернуть nil
	if val := req.GetAuthKey(); val != nil {
		t.Errorf("Expected nil byte slice from nil struct getter, got %v", val)
	}
}
