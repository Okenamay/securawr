package cli

import (
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		wantErr bool
	}{
		{"Short password", "123", true},
		{"Only letters", "abcdefgh", true},
		{"Only digits", "12345678", true},
		{"Valid password", "abc12345", false},
		{"Valid complex", "Password123!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.pass)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthCommandsStructure(t *testing.T) {
	// Проверка структуры команд без их выполнения
	if authCmd.Use != "auth" {
		t.Error("authCmd use string is incorrect")
	}

	foundRegister := false
	foundLogin := false

	for _, cmd := range authCmd.Commands() {
		if cmd.Use == "register" {
			foundRegister = true
		}
		if cmd.Use == "login" {
			foundLogin = true
		}
	}

	if !foundRegister {
		t.Error("register command not added to authCmd")
	}
	if !foundLogin {
		t.Error("login command not added to authCmd")
	}
}
