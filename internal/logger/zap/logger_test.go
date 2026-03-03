package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{"debug_level", "debug", false},
		{"info_level", "info", false},
		{"error_level", "error", false},
		{"invalid_level", "unknown_level", true}, // Проверяем ошибку парсинга уровня
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("New() returned nil logger")
			}
		})
	}
}
