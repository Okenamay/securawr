package config

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

// resetFlags сбрасывает глобальный CommandLine, чтобы parseFlags мог
// регистрировать флаги заново в каждом тесте
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

// clearEnv очищает все переменные окружения, которые могут повлиять на конфиг
func clearEnv() {
	envs := []string{
		"GRPC_PORT", "DATABASE_DSN", "LOG_LEVEL", "PEPPER",
		"JWT_SECRET", "USER_QUOTA", "MAX_FILE_SIZE", "CONFIG",
	}
	for _, e := range envs {
		os.Unsetenv(e)
	}
}

func TestParseFlags_Defaults(t *testing.T) {
	resetFlags()
	clearEnv()
	// Сохраняем старые аргументы и восстанавливаем в конце
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Эмулируем запуск без флагов
	os.Args = []string{"cmd"}

	cfg, err := parseFlags()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Проверяем дефолтные значения
	if cfg.GRPCPort != defaultGRPCPort {
		t.Errorf("Expected default port %s, got %s", defaultGRPCPort, cfg.GRPCPort)
	}
	if cfg.DSN != defaultDSN {
		t.Errorf("Expected default DSN %s, got %s", defaultDSN, cfg.DSN)
	}
	if cfg.LogLevel != defaultLogLevel {
		t.Errorf("Expected default LogLevel %s, got %s", defaultLogLevel, cfg.LogLevel)
	}
}

func TestParseFlags_Flags(t *testing.T) {
	resetFlags()
	clearEnv()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	expectedPort := ":9090"
	expectedDSN := "postgres://flag:5432/db"

	// Эмулируем передачу флагов
	os.Args = []string{
		"cmd",
		"-port", expectedPort,
		"-dsn", expectedDSN,
	}

	cfg, err := parseFlags()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.GRPCPort != expectedPort {
		t.Errorf("Expected port %s, got %s", expectedPort, cfg.GRPCPort)
	}
	if cfg.DSN != expectedDSN {
		t.Errorf("Expected DSN %s, got %s", expectedDSN, cfg.DSN)
	}
}

func TestParseFlags_Env(t *testing.T) {
	resetFlags()
	clearEnv()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	expectedPort := ":9999"
	expectedQuota := "1024"

	// Устанавливаем ENV
	os.Setenv("GRPC_PORT", expectedPort)
	os.Setenv("USER_QUOTA", expectedQuota)
	defer clearEnv()

	cfg, err := parseFlags()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.GRPCPort != expectedPort {
		t.Errorf("Expected port %s, got %s", expectedPort, cfg.GRPCPort)
	}
	if cfg.UserQuota != 1024 {
		t.Errorf("Expected UserQuota 1024, got %d", cfg.UserQuota)
	}
}

func TestParseFlags_File(t *testing.T) {
	resetFlags()
	clearEnv()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// 1. Создаем временный файл конфигурации
	port := ":7070"
	dsn := "postgres://file"

	fContent := fileConfig{
		GRPCPort: &port,
		DSN:      &dsn,
	}

	tmpFile, err := os.CreateTemp("", "config_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if err := json.NewEncoder(tmpFile).Encode(fContent); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// 2. Передаем путь к файлу через флаг -config
	os.Args = []string{"cmd", "-config", tmpFile.Name()}

	cfg, err := parseFlags()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.GRPCPort != port {
		t.Errorf("Expected port %s, got %s", port, cfg.GRPCPort)
	}
	if cfg.DSN != dsn {
		t.Errorf("Expected DSN %s, got %s", dsn, cfg.DSN)
	}
}

func TestParseFlags_Precedence(t *testing.T) {
	// Приоритет: Flag > Env > File > Default
	resetFlags()
	clearEnv()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Setup File
	filePort := ":1111"
	tmpFile, err := os.CreateTemp("", "config_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	fContent := fileConfig{GRPCPort: &filePort}
	json.NewEncoder(tmpFile).Encode(fContent)
	tmpFile.Close()

	// Setup Env
	envPort := ":2222"
	os.Setenv("GRPC_PORT", envPort)
	defer os.Unsetenv("GRPC_PORT")

	// Setup Flag
	flagPort := ":3333"

	// Сценарий 1: Flag vs Env vs File -> Flag должен победить
	os.Args = []string{"cmd", "-config", tmpFile.Name(), "-port", flagPort}

	cfg, err := parseFlags()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if cfg.GRPCPort != flagPort {
		t.Errorf("Precedence fail: expected Flag val %s, got %s", flagPort, cfg.GRPCPort)
	}

	// Сценарий 2: Env vs File -> Env должен победить
	resetFlags()
	os.Args = []string{"cmd", "-config", tmpFile.Name()} // Флаг порта убрали

	cfg, err = parseFlags()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if cfg.GRPCPort != envPort {
		t.Errorf("Precedence fail: expected Env val %s, got %s", envPort, cfg.GRPCPort)
	}

	// Сценарий 3: File only -> File должен победить Default
	resetFlags()
	os.Unsetenv("GRPC_PORT") // Убрали ENV
	os.Args = []string{"cmd", "-config", tmpFile.Name()}

	cfg, err = parseFlags()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if cfg.GRPCPort != filePort {
		t.Errorf("Precedence fail: expected File val %s, got %s", filePort, cfg.GRPCPort)
	}
}

func TestInitConfig_Priority(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		wantPort string
		wantDSN  string
	}{
		{
			name:     "Default values",
			args:     []string{"cmd"},
			wantPort: ":8080",                                                                     // defaultGRPCPort
			wantDSN:  "postgres://postgres:postgres@localhost:5432/securawr_test?sslmode=disable", // defaultDSN
		},
		{
			name:     "Flags priority",
			args:     []string{"cmd", "-port", ":9999", "-dsn", "postgres://flag_db"},
			wantPort: ":9999",
			wantDSN:  "postgres://flag_db",
		},
		{
			name: "Env priority (over defaults)",
			args: []string{"cmd"},
			env: map[string]string{
				"GRPC_PORT":    ":7777",
				"DATABASE_DSN": "postgres://env_db",
			},
			wantPort: ":7777",
			wantDSN:  "postgres://env_db",
		},
		{
			name: "Flags override Env (Flag > Env)",
			args: []string{"cmd", "-port", ":1111"},
			env: map[string]string{
				"GRPC_PORT": ":2222",
			},
			wantPort: ":1111", // Должен победить флаг
			wantDSN:  "postgres://postgres:postgres@localhost:5432/securawr_test?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Сброс глобального состояния флагов
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// 2. Установка аргументов
			os.Args = tt.args

			// 3. Установка переменных окружения
			os.Clearenv()
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			// 4. Вызов внутренней функции parseFlags
			cfg, err := parseFlags()
			if err != nil {
				t.Fatalf("parseFlags() unexpected error: %v", err)
			}

			// 5. Проверки полей (используем GRPCPort и DSN)
			if cfg.GRPCPort != tt.wantPort {
				t.Errorf("GRPCPort = %q, want %q", cfg.GRPCPort, tt.wantPort)
			}
			if cfg.DSN != tt.wantDSN {
				t.Errorf("DSN = %q, want %q", cfg.DSN, tt.wantDSN)
			}
		})
	}
}
